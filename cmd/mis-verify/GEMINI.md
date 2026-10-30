# `mis-verify` CLI (`cmd/mis-verify`)

CI Contract & UK GDPR Gatekeeper binary (`bin/mis-verify`, Sections 8B & 10A).

## Usage

```bash
./bin/mis-verify <plugin-dir>
# or
./bin/mis-verify --plugin <plugin-dir>
```

## Files

* [`main.go`](file:///home/kieren/Documents/code/optimis/cmd/mis-verify/main.go): Parses `--plugin` or positional `<plugin-dir>` and invokes [`sdk.VerifyPluginDir`](file:///home/kieren/Documents/code/optimis/internal/sdk/verify.go).
