# Go Host Server Binary & Local Break-Glass CLI (`cmd/mis-server`)

This package provides the main executable entrypoint (`./bin/mis-server`) for the Optimis School MIS Host Container and its Section 7C Local Break-Glass CLI (`./bin/mis-server config ...`).

## Architectural Responsibilities

* **Minimal Bootstrap + Fail-Fast Configuration ([`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-server/main.go))**:
  - Loads `.env` and validates the three immutable bootstrap environment variables (`DATA_DIR_PATH`, `SCHOOL_URN`, `HOST_KMS_KEY_HANDLE`) before routing either to the Local Break-Glass CLI (`./bin/mis-server config ...`) or the full HTTP Host Container (`PORT`, `PLUGINS_DIR`, `ENABLE_DEV_PERSONA_SWITCHER`, `ENABLE_REAL_HTTP_EGRESS`, `ENABLE_REAL_CLOUD_KMS`, `ENABLE_REAL_PARENT_DMZ_SYNC`, `ENABLE_REAL_MAT_SYNC`, `ENABLE_MOCK_DFE_CENSUS_SERVICE`) with zero silent default fallbacks.
* **Local Break-Glass CLI (`./bin/mis-server config ...` — Section 7C)**:
  - Routes `config list`, `config get <key>`, `config set <key> <value>`, and `config repair-sso` to [`host.RunConfigCLI`](file:///home/kieren/Documents/code/optimis/internal/host/cli_config.go#L16) so School Admins can inspect and repair `__system_config` in `core.db` directly from the terminal without web UI access when federated SSO is misconfigured.
* **Host Container Startup**:
  - Initializes [`host.NewHost`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L139) (which boots `core.db` in WAL mode via `CoreDAL`), configures auth mode (`ConfigureAuthMode`), HTTP egress mode (`ConfigureEgressMode`), external dependency flags (`ConfigureCloudKMSMode`, `ConfigureParentDMZMode`, `ConfigureMATSyncMode`), and Mock DfE COLLECT service mode (`ConfigureMockDfeCensusService`), loads `test_plugin` and `dfe_census_plugin`, and starts the HTTP/RPC gateway (`NewHTTPHandler`).
