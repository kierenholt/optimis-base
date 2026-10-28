# Go Host Server Binary Entrypoint (`cmd/mis-server`)

This package provides the main executable entrypoint (`./bin/mis-server`) for the Optimis School MIS Host Container.

## Architectural Responsibilities

* **Fail-Fast Bootstrap Configuration ([`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-server/main.go))**:
  - Loads `.env` and validates mandatory bootstrap environment variables (`DATA_DIR_PATH`, `SCHOOL_URN`, `HOST_KMS_KEY_HANDLE`, `PORT`, `PLUGINS_DIR`, `ENABLE_DEV_PERSONA_SWITCHER`, `ENABLE_REAL_HTTP_EGRESS`, `ENABLE_REAL_CLOUD_KMS`, `ENABLE_REAL_PARENT_DMZ_SYNC`, `ENABLE_REAL_MAT_SYNC`, `ENABLE_MOCK_DFE_CENSUS_SERVICE`) with zero silent default fallbacks.
  - Initializes [`host.NewHost`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L130) (which boots `core.db` in WAL mode via `CoreDAL`), configures auth mode (`ConfigureAuthMode`), HTTP egress mode (`ConfigureEgressMode`), external dependency flags (`ConfigureCloudKMSMode`, `ConfigureParentDMZMode`, `ConfigureMATSyncMode`), and Mock DfE COLLECT service mode (`ConfigureMockDfeCensusService`), loads `test_plugin` and `dfe_census_plugin`, and starts the HTTP/RPC gateway (`NewHTTPHandler`).
