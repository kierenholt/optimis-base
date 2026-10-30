# `mis-pack` CLI (`cmd/mis-pack`)

Lucide SVG Extractor & Deterministic `.mispkg` Archive Bundler (`bin/mis-pack`, Sections 8C & 10A).

## Usage

```bash
./bin/mis-pack <plugin-dir> --out dist/plugin.mispkg
# or
./bin/mis-pack --plugin <plugin-dir> --out dist/plugin.mispkg
```

## Files

* [`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-pack/main.go): Parses `--plugin` / positional `<plugin-dir>` and `--out` and invokes [`sdk.PackPlugin`](file:///home/kieren/Documents/code/optimis/internal/sdk/pack.go).
