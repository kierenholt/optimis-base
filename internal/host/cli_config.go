package host

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"optimis/internal/dal"
)

// RunConfigCLI initializes CoreDAL for the target school data directory and executes
// the Section 7C Local Break-Glass CLI (`./mis-server config ...`).
func RunConfigCLI(
	ctx context.Context,
	dataDirPath string,
	schoolURN string,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if strings.TrimSpace(dataDirPath) == "" {
		return errors.New("missing mandatory parameter: dataDirPath")
	}
	if strings.TrimSpace(schoolURN) == "" {
		return errors.New("missing mandatory parameter: schoolURN")
	}

	coreDAL, err := dal.NewCoreDAL(dataDirPath, schoolURN)
	if err != nil {
		return err
	}
	defer coreDAL.Close()

	return RunConfigCLIWithDAL(ctx, coreDAL, args, stdout, stderr)
}

// RunConfigCLIWithDAL executes `config list`, `config get <key>`, `config set <key> <value>`,
// or `config repair-sso` against `__system_config` in `core.db`.
func RunConfigCLIWithDAL(
	ctx context.Context,
	coreDAL *dal.CoreDAL,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if coreDAL == nil {
		return errors.New("missing mandatory parameter: coreDAL")
	}
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return errors.New("missing mandatory config subcommand: expected 'list', 'get', 'set', or 'repair-sso'")
	}

	subcmd := strings.TrimSpace(args[0])
	subArgs := args[1:]

	switch subcmd {
	case "list":
		fs := flag.NewFlagSet("config list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		asJSON := fs.Bool("json", false, "Output __system_config rows as JSON array")
		if err := fs.Parse(subArgs); err != nil {
			return err
		}
		if fs.NArg() > 0 {
			return fmt.Errorf("config list does not accept positional arguments (got %v)", fs.Args())
		}

		entries, err := coreDAL.ListSystemConfigEntries(ctx)
		if err != nil {
			return err
		}
		if *asJSON {
			enc, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(stdout, string(enc))
			return nil
		}
		for _, e := range entries {
			_, _ = fmt.Fprintf(stdout, "%s=%s (updated_at=%s)\n", e.Key, e.Value, e.UpdatedAt)
		}
		return nil

	case "get":
		fs := flag.NewFlagSet("config get", flag.ContinueOnError)
		fs.SetOutput(stderr)
		asJSON := fs.Bool("json", false, "Output __system_config entry as JSON")
		if err := fs.Parse(subArgs); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return errors.New("usage: mis-server config get <key> (missing mandatory <key> argument)")
		}

		key := strings.TrimSpace(fs.Arg(0))
		entry, err := coreDAL.GetSystemConfigEntry(ctx, key)
		if err != nil {
			return err
		}
		if *asJSON {
			enc, err := json.MarshalIndent(entry, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(stdout, string(enc))
			return nil
		}
		_, _ = fmt.Fprintln(stdout, entry.Value)
		return nil

	case "set":
		fs := flag.NewFlagSet("config set", flag.ContinueOnError)
		fs.SetOutput(stderr)
		asJSON := fs.Bool("json", false, "Output updated __system_config entry as JSON")
		if err := fs.Parse(subArgs); err != nil {
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("usage: mis-server config set <key> <value> (missing mandatory <key> and <value> arguments)")
		}
		if fs.NArg() < 2 {
			return errors.New("usage: mis-server config set <key> <value> (missing mandatory <value> argument)")
		}
		if fs.NArg() > 2 {
			return fmt.Errorf("usage: mis-server config set <key> <value> (unexpected extra arguments: %v)", fs.Args()[2:])
		}

		key := strings.TrimSpace(fs.Arg(0))
		val := strings.TrimSpace(fs.Arg(1))
		entry, err := coreDAL.SetSystemConfigValue(ctx, key, val)
		if err != nil {
			return err
		}
		if *asJSON {
			enc, err := json.MarshalIndent(entry, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(stdout, string(enc))
			return nil
		}
		_, _ = fmt.Fprintf(stdout, "SET %s=%s (updated_at=%s)\n", entry.Key, entry.Value, entry.UpdatedAt)
		return nil

	case "repair-sso":
		fs := flag.NewFlagSet("config repair-sso", flag.ContinueOnError)
		fs.SetOutput(stderr)
		provider := fs.String("provider", "", "Federated SSO provider (ENTRA_ID, GOOGLE_WORKSPACE, or OIDC)")
		issuerURL := fs.String("issuer-url", "", "Mandatory HTTPS OIDC issuer URL")
		clientID := fs.String("client-id", "", "Mandatory OIDC client ID")
		redirectURI := fs.String("redirect-uri", "", "Optional HTTPS OAuth redirect callback URI")
		asJSON := fs.Bool("json", false, "Output repaired SSO configuration as JSON")
		if err := fs.Parse(subArgs); err != nil {
			return err
		}

		resolvedIssuer := strings.TrimSpace(*issuerURL)
		resolvedClient := strings.TrimSpace(*clientID)
		if resolvedIssuer == "" && fs.NArg() >= 1 {
			resolvedIssuer = strings.TrimSpace(fs.Arg(0))
		}
		if resolvedClient == "" && fs.NArg() >= 2 {
			resolvedClient = strings.TrimSpace(fs.Arg(1))
		}

		repaired, err := coreDAL.RepairSSOConfig(ctx, dal.RepairSSOConfigInput{
			Provider:    strings.TrimSpace(*provider),
			IssuerURL:   resolvedIssuer,
			ClientID:    resolvedClient,
			RedirectURI: strings.TrimSpace(*redirectURI),
		})
		if err != nil {
			return err
		}

		if *asJSON {
			enc, err := json.MarshalIndent(repaired, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(stdout, string(enc))
			return nil
		}

		_, _ = fmt.Fprintf(
			stdout,
			"REPAIRED_SSO sso_provider=%s sso_oidc_issuer_url=%s sso_oidc_client_id=%s",
			repaired.Provider,
			repaired.IssuerURL,
			repaired.ClientID,
		)
		if repaired.RedirectURI != "" {
			_, _ = fmt.Fprintf(stdout, " sso_redirect_uri=%s", repaired.RedirectURI)
		}
		_, _ = fmt.Fprintln(stdout)
		return nil

	default:
		return fmt.Errorf("unknown config subcommand %q: expected 'list', 'get', 'set', or 'repair-sso'", subcmd)
	}
}
