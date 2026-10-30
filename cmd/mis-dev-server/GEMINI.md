# `mis-dev-server` CLI (`cmd/mis-dev-server`)

Distributable Local Testbed Server (`bin/mis-dev-server`, Sections 8A, 10A & 10D) that boots the Go Host Container with an auto-seeded synthetic UK school `core.db`, hot-reload plugin mounting, persona switching (`--persona`, `?__persona=`, `X-MIS-Persona`), and single-line structured `stdout` observability.

## Usage

```bash
./bin/mis-dev-server --plugin ./plugins/behaviour_plugin --persona stf_tch_7a --port 18080
# Dry-run readiness check:
./bin/mis-dev-server --plugin ./plugins/behaviour_plugin --persona stf_tch_7a --dry-run
```

## Files

* [`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-dev-server/main.go): Parses `--plugin`, `--persona`, `--port`, `--db`, and `--dry-run`, seeds the synthetic school cohort via [`sdk.SeedSyntheticDevCohort`](file:///home/kieren/Documents/code/optimis/internal/sdk/test_runner.go), mounts the target plugin, and serves the Host HTTP gateway with structured `stdout` logging.
