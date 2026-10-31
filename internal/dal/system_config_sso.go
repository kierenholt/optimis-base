package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type SystemConfigEntry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

type SSOConfig struct {
	Provider    string `json:"provider"`
	IssuerURL   string `json:"issuer_url"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}

type RepairSSOConfigInput struct {
	Provider    string `json:"provider,omitempty"`
	IssuerURL   string `json:"issuer_url"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}

func (d *CoreDAL) GetSystemConfigValue(ctx context.Context, key string) (string, error) {
	entry, err := d.GetSystemConfigEntry(ctx, key)
	if err != nil {
		return "", err
	}
	return entry.Value, nil
}

func (d *CoreDAL) GetSystemConfigEntry(ctx context.Context, key string) (*SystemConfigEntry, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return nil, errors.New("missing mandatory parameter: key")
	}

	var entry SystemConfigEntry
	err := d.readDB.QueryRowContext(
		ctx,
		`SELECT key, value, updated_at FROM __system_config WHERE key = ?`,
		trimmedKey,
	).Scan(&entry.Key, &entry.Value, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("missing mandatory __system_config key: %s", trimmedKey)
		}
		return nil, fmt.Errorf("failed to query __system_config key %s: %w", trimmedKey, err)
	}
	if strings.TrimSpace(entry.Value) == "" {
		return nil, fmt.Errorf("__system_config key %s has empty value", trimmedKey)
	}
	return &entry, nil
}

func (d *CoreDAL) ListSystemConfigEntries(ctx context.Context) ([]SystemConfigEntry, error) {
	rows, err := d.readDB.QueryContext(
		ctx,
		`SELECT key, value, updated_at FROM __system_config ORDER BY key ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list __system_config entries: %w", err)
	}
	defer rows.Close()

	entries := make([]SystemConfigEntry, 0, 8)
	for rows.Next() {
		var e SystemConfigEntry
		if err := rows.Scan(&e.Key, &e.Value, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan __system_config row: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating __system_config rows: %w", err)
	}
	return entries, nil
}

func (d *CoreDAL) SetSystemConfigValue(ctx context.Context, key string, value string) (*SystemConfigEntry, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return nil, errors.New("missing mandatory parameter: key")
	}
	trimmedVal := strings.TrimSpace(value)
	if trimmedVal == "" {
		return nil, errors.New("missing mandatory parameter: value")
	}

	if _, err := d.writeDB.ExecContext(
		ctx,
		`INSERT INTO __system_config (key, value, updated_at)
		 VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		 ON CONFLICT(key) DO UPDATE SET
		     value = excluded.value,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		trimmedKey,
		trimmedVal,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert __system_config key %s: %w", trimmedKey, err)
	}

	return d.GetSystemConfigEntry(ctx, trimmedKey)
}

func (d *CoreDAL) DeleteSystemConfigKey(ctx context.Context, key string) error {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return errors.New("missing mandatory parameter: key")
	}
	if _, err := d.writeDB.ExecContext(
		ctx,
		`DELETE FROM __system_config WHERE key = ?`,
		trimmedKey,
	); err != nil {
		return fmt.Errorf("failed to delete __system_config key %s: %w", trimmedKey, err)
	}
	return nil
}

func validateHTTPSOriginOrCallbackURL(rawURL string, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("missing mandatory parameter: %s", fieldName)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid %s %q: must be a valid https:// URL", fieldName, rawURL)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("invalid %s %q: scheme must be https://", fieldName, rawURL)
	}
	return trimmed, nil
}

func resolveAndValidateSSOProvider(rawProvider string, issuerURL string) (string, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(rawProvider))
	if trimmed == "" {
		lowerIssuer := strings.ToLower(issuerURL)
		if strings.Contains(lowerIssuer, "login.microsoftonline.com") {
			return "ENTRA_ID", nil
		}
		if strings.Contains(lowerIssuer, "accounts.google.com") {
			return "GOOGLE_WORKSPACE", nil
		}
		return "OIDC", nil
	}
	switch trimmed {
	case "ENTRA_ID", "GOOGLE_WORKSPACE", "OIDC":
		return trimmed, nil
	default:
		return "", fmt.Errorf("invalid SSO provider %q: must be ENTRA_ID, GOOGLE_WORKSPACE, or OIDC", rawProvider)
	}
}

func (d *CoreDAL) ValidateAndLoadSSOConfig(ctx context.Context) (*SSOConfig, error) {
	rawIssuer, err := d.GetSystemConfigValue(ctx, "sso_oidc_issuer_url")
	if err != nil {
		return nil, err
	}
	issuerURL, err := validateHTTPSOriginOrCallbackURL(rawIssuer, "sso_oidc_issuer_url")
	if err != nil {
		return nil, err
	}

	clientID, err := d.GetSystemConfigValue(ctx, "sso_oidc_client_id")
	if err != nil {
		return nil, err
	}

	rawProvider, _ := d.GetSystemConfigValue(ctx, "sso_provider")
	provider, err := resolveAndValidateSSOProvider(rawProvider, issuerURL)
	if err != nil {
		return nil, err
	}

	redirectURI, _ := d.GetSystemConfigValue(ctx, "sso_redirect_uri")
	if strings.TrimSpace(redirectURI) != "" {
		validatedRedirect, rErr := validateHTTPSOriginOrCallbackURL(redirectURI, "sso_redirect_uri")
		if rErr != nil {
			return nil, rErr
		}
		redirectURI = validatedRedirect
	}

	return &SSOConfig{
		Provider:    provider,
		IssuerURL:   issuerURL,
		ClientID:    strings.TrimSpace(clientID),
		RedirectURI: strings.TrimSpace(redirectURI),
	}, nil
}

func (d *CoreDAL) RepairSSOConfig(ctx context.Context, in RepairSSOConfigInput) (*SSOConfig, error) {
	issuerURL, err := validateHTTPSOriginOrCallbackURL(in.IssuerURL, "issuer-url")
	if err != nil {
		return nil, err
	}
	clientID := strings.TrimSpace(in.ClientID)
	if clientID == "" {
		return nil, errors.New("missing mandatory parameter: client-id")
	}
	provider, err := resolveAndValidateSSOProvider(in.Provider, issuerURL)
	if err != nil {
		return nil, err
	}
	redirectURI := strings.TrimSpace(in.RedirectURI)
	if redirectURI != "" {
		validatedRedirect, rErr := validateHTTPSOriginOrCallbackURL(redirectURI, "redirect-uri")
		if rErr != nil {
			return nil, rErr
		}
		redirectURI = validatedRedirect
	}

	tx, err := d.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin RepairSSOConfig transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	pairs := [][2]string{
		{"sso_provider", provider},
		{"sso_oidc_issuer_url", issuerURL},
		{"sso_oidc_client_id", clientID},
	}
	if redirectURI != "" {
		pairs = append(pairs, [2]string{"sso_redirect_uri", redirectURI})
	}

	for _, kv := range pairs {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO __system_config (key, value, updated_at)
			 VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
			 ON CONFLICT(key) DO UPDATE SET
			     value = excluded.value,
			     updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
			kv[0],
			kv[1],
		); err != nil {
			return nil, fmt.Errorf("failed to write __system_config key %s during SSO repair: %w", kv[0], err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit RepairSSOConfig transaction: %w", err)
	}

	return &SSOConfig{
		Provider:    provider,
		IssuerURL:   issuerURL,
		ClientID:    clientID,
		RedirectURI: redirectURI,
	}, nil
}
