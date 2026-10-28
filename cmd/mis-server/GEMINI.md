# Go Host Server Binary Entrypoint (`cmd/mis-server`)

This package provides the main executable entrypoint (`./bin/mis-server`) for the Optimis School MIS Host Container.

## Architectural Responsibilities

* **Fail-Fast Bootstrap Configuration ([`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-server/main.go))**:
  - Loads `.env` and validates mandatory bootstrap environment variables (`DATA_DIR_PATH`, `SCHOOL_URN`, `HOST_KMS_KEY_HANDLE`, `PORT`) with zero silent default fallbacks.
  - Initializes [`host.NewHost`](file:///home/kieren/Documents/code/optimis/internal/host/host.go#L93-L126) (which boots `core.db` in WAL mode via `CoreDAL`), loads `test_plugin`, and starts the HTTP/RPC gateway (`NewHTTPHandler`).
