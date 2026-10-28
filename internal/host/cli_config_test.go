package host_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"optimis/internal/host"
)

func TestConfigCLI_ListGetAndSet_InspectsAndMutatesSystemConfigInCoreDB_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	t.Cleanup(func() {
		for _, entry := range h.LogsSince("0") {
			t.Logf("[service-log] %s", entry)
		}
	})

	// 1. config list returns seeded school_urn
	var stdout, stderr bytes.Buffer
	err := host.RunConfigCLIWithDAL(ctx, h.DAL(), []string{"list"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("config list failed: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "school_urn=100001") {
		t.Errorf("expected config list output to contain school_urn=100001, got: %s", stdout.String())
	}

	// 2. config get school_urn
	stdout.Reset()
	stderr.Reset()
	err = host.RunConfigCLIWithDAL(ctx, h.DAL(), []string{"get", "school_urn"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("config get school_urn failed: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "100001" {
		t.Errorf("expected school_urn value 100001, got %q", strings.TrimSpace(stdout.String()))
	}

	// 3. config set custom_key custom_val
	stdout.Reset()
	stderr.Reset()
	err = host.RunConfigCLIWithDAL(ctx, h.DAL(), []string{"set", "mat_sync_feed_url", "https://mat.oakridge.uk/v1/feed"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}
	t.Cleanup(func() {
		_ = h.DAL().DeleteSystemConfigKey(context.Background(), "mat_sync_feed_url")
	})

	// 4. config get mat_sync_feed_url
	stdout.Reset()
	stderr.Reset()
	err = host.RunConfigCLIWithDAL(ctx, h.DAL(), []string{"get", "mat_sync_feed_url"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("config get mat_sync_feed_url failed: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "https://mat.oakridge.uk/v1/feed" {
		t.Errorf("expected https://mat.oakridge.uk/v1/feed, got %q", strings.TrimSpace(stdout.String()))
	}
}

func TestConfigCLI_RepairSSO_RepairsMisconfiguredSSOAndUnblocksRealSSOAuthMode_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	t.Cleanup(func() {
		_ = h.DAL().DeleteSystemConfigKey(context.Background(), "sso_oidc_issuer_url")
		_ = h.DAL().DeleteSystemConfigKey(context.Background(), "sso_oidc_client_id")
		_ = h.DAL().DeleteSystemConfigKey(context.Background(), "sso_provider")
		_ = h.DAL().DeleteSystemConfigKey(context.Background(), "sso_redirect_uri")
		for _, entry := range h.LogsSince("0") {
			t.Logf("[service-log] %s", entry)
		}
	})

	// 1. Seed a broken/insecure sso_oidc_issuer_url and verify ConfigureAuthMode(false) fails fast
	var stdout, stderr bytes.Buffer
	if err := host.RunConfigCLIWithDAL(ctx, h.DAL(), []string{"set", "sso_oidc_issuer_url", "http://broken-idp.local"}, &stdout, &stderr); err != nil {
		t.Fatalf("config set broken sso_oidc_issuer_url failed: %v", err)
	}
	if err := h.ConfigureAuthMode(ctx, false); err == nil || !strings.Contains(err.Error(), "REAL_SSO_NOT_CONFIGURED") {
		t.Fatalf("expected ConfigureAuthMode(false) to fail fast with REAL_SSO_NOT_CONFIGURED on broken SSO config, got: %v", err)
	}

	// 2. Run config repair-sso to repair SSO settings in core.db
	stdout.Reset()
	stderr.Reset()
	err := host.RunConfigCLIWithDAL(
		ctx,
		h.DAL(),
		[]string{
			"repair-sso",
			"--provider", "GOOGLE_WORKSPACE",
			"--issuer-url", "https://accounts.google.com",
			"--client-id", "oakridge-google-client-777.apps.googleusercontent.com",
			"--redirect-uri", "https://mis.oakridge.sch.uk/api/auth/sso/callback",
		},
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("config repair-sso failed: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "sso_provider=GOOGLE_WORKSPACE") {
		t.Errorf("expected repair-sso output to include sso_provider=GOOGLE_WORKSPACE, got: %s", stdout.String())
	}

	// 3. ConfigureAuthMode(false) now succeeds with repaired SSO settings in __system_config
	if err := h.ConfigureAuthMode(ctx, false); err != nil {
		t.Fatalf("expected ConfigureAuthMode(false) to succeed after config repair-sso, got: %v", err)
	}
}

func TestConfigCLI_FailFastOnMissingKeysAndInvalidArguments_ReturnsExplicitErrors_test(t *testing.T) {
	ctx := context.Background()
	h, cleanup := setupIsolatedTestHost(t)
	t.Cleanup(cleanup)

	cases := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{"missing_subcommand", []string{}, "subcommand"},
		{"unknown_subcommand", []string{"purge-all"}, "unknown config subcommand"},
		{"get_missing_key_arg", []string{"get"}, "key"},
		{"get_nonexistent_key", []string{"get", "missing_key_123"}, "missing mandatory __system_config key: missing_key_123"},
		{"set_missing_value_arg", []string{"set", "only_key"}, "value"},
		{"repair_sso_non_https_issuer", []string{"repair-sso", "--issuer-url", "http://insecure.local", "--client-id", "cid"}, "https://"},
		{"repair_sso_missing_client_id", []string{"repair-sso", "--issuer-url", "https://login.microsoftonline.com/tenant/v2.0"}, "client-id"},
		{"repair_sso_invalid_provider", []string{"repair-sso", "--provider", "UNKNOWN_IDP", "--issuer-url", "https://login.microsoftonline.com/tenant/v2.0", "--client-id", "cid"}, "provider"},
	}

	for _, tc := range cases {
		var stdout, stderr bytes.Buffer
		err := host.RunConfigCLIWithDAL(ctx, h.DAL(), tc.args, &stdout, &stderr)
		if err == nil {
			t.Errorf("[%s] expected fail-fast error for args %v, got nil", tc.name, tc.args)
			continue
		}
		if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.expectedErr)) {
			t.Errorf("[%s] expected error containing %q, got %v", tc.name, tc.expectedErr, err)
		}
	}
}
