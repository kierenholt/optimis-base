# `mis-test` CLI (`cmd/mis-test`)

Headless Lua Test Runner binary (`bin/mis-test`, Sections 8B, 10A & 10D).

## Usage

```bash
./bin/mis-test <plugin-dir> --persona stf_tch_7a
# or
./bin/mis-test --plugin <plugin-dir> --persona stf_tch_7a
```

## Files

* [`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-test/main.go): Parses `--plugin` / positional `<plugin-dir>` and `--persona` and invokes [`sdk.RunPluginTests`](file:///home/kieren/Documents/code/optimis/internal/sdk/test_runner.go).
