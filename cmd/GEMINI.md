# Go Command Line & Server Binaries (`cmd`)

This directory contains the Go `main` package entrypoints for the Optimis platform.

## Subdirectories

* [`cmd/mis-server`](file:///home/kieren/Documents/code/optimis/cmd/mis-server/GEMINI.md): Main Go Host Container binary (`./bin/mis-server`) enforcing fail-fast `.env` validation (`PORT`, `DATA_DIR_PATH`, `PLUGINS_DIR`, `SCHOOL_URN`, `HOST_KMS_KEY_HANDLE`, `ENABLE_DEV_PERSONA_SWITCHER`, `ENABLE_REAL_HTTP_EGRESS`, `ENABLE_REAL_CLOUD_KMS`, `ENABLE_REAL_PARENT_DMZ_SYNC`, `ENABLE_REAL_MAT_SYNC`, `ENABLE_MOCK_DFE_CENSUS_SERVICE`), loading `test_plugin` and `dfe_census_plugin`, and serving the embedded React Host UI Shell, Lua RPC gateway, and Core DAL.
* [`cmd/mis-seed-qa`](file:///home/kieren/Documents/code/optimis/cmd/mis-seed-qa/GEMINI.md): Deterministic synthetic UK All-Through school QA database generator (`npm run seed:qa`) populating all 45 core tables in `<DATA_DIR_PATH>/qa_core.db`.
