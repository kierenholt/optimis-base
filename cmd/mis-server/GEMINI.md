# Go Host Server Binary Entrypoint (`cmd/mis-server`)

This package provides the main executable entrypoint (`./bin/mis-server`) for the Optimis School MIS Host Container.

## Architectural Responsibilities

* **Fail-Fast Bootstrap Configuration ([`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-server/main.go))**:
  - Loads `.env` and validates mandatory bootstrap environment variables (`DATA_DIR_PATH`, `SCHOOL_URN`, `HOST_KMS_KEY_HANDLE`, `PORT`, `PLUGINS_DIR`, `ENABLE_DEV_PERSONA_SWITCHER`, `ENABLE_REAL_HTTP_EGRESS`) with zero silent default fallbacks.
  - Initializes [`host.NewHost`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L119) (which boots `core.db` in WAL mode via `CoreDAL`), configures auth mode (`ConfigureAuthMode`) and HTTP egress mode (`ConfigureEgressMode`), loads `test_plugin`, and starts the HTTP/RPC gateway (`NewHTTPHandler`).
