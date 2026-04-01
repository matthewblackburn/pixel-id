# pixel-id

Generate unique 64-bit snowflake IDs and render deterministic pixel avatars from them. Same ID, same avatar — guaranteed across Go and JavaScript through a shared WASM core.

## Project structure

```
go/          Go package — ID generation + SVG/PNG avatar rendering (server-side)
wasm/        TinyGo WASM source — same algorithm compiled for browser use
ts/          TypeScript/npm package — thin wrapper around WASM, React component
spec/        Cross-language test vectors (JSON)
scripts/     Vector generation script
example/     Dockerized demo app (Go API + React frontend)
```

## Key design decisions

- **Algorithm immutability**: The derivation algorithm (FNV-1a hash → grid + colors + corners) is the contract. Changing how an ID maps to an avatar is a semver major bump.
- **WASM for cross-language parity**: The TypeScript package runs the Go algorithm compiled to WASM via TinyGo. This guarantees identical output — no maintaining parallel implementations.
- **The Go `go/` package is the source of truth** for the algorithm. The `wasm/main.go` reimplements it without stdlib deps that TinyGo doesn't support (`encoding/binary`, `math/bits`, `fmt`). Test vectors in `spec/` ensure parity.

## Build commands

```bash
# Go tests
cd go && go test ./...

# WASM rebuild (requires TinyGo)
cd wasm && ./build.sh

# TypeScript tests
cd ts && npm test

# Example app
cd example && make up
```

## WASM build pipeline

`wasm/build.sh` compiles `wasm/main.go` with TinyGo, base64-encodes the binary into `ts/src/wasm-binary.ts`, and copies the TinyGo JS glue to `ts/src/wasm_exec.js`. These generated files are committed so npm consumers don't need TinyGo.

## Configurable features

- **Grid size**: `WithGrid(w, h)` — max depends on other settings, see `MaxGridSize()`
- **Number of colors**: `WithColors(1..4)` — multiple foreground colors per avatar
- **Curved corners**: `WithCurves(true)` — some cell corners get rounded, derived from hash bits

## Testing

Test vectors in `spec/vectors.json` are generated from `scripts/generate-vectors.go` (standalone, duplicates the algorithm). Both Go and TypeScript test suites assert against these vectors. SVG cross-language parity is verified in `spec/svg_vectors.json`.

When modifying the algorithm:
1. Update `go/algorithm.go` (the reference)
2. Update `wasm/main.go` (TinyGo-compatible mirror)
3. Update `scripts/generate-vectors.go` (standalone mirror)
4. Run `cd scripts && go run generate-vectors.go` to regenerate vectors
5. Run `cd wasm && ./build.sh` to rebuild WASM
6. Run tests in both `go/` and `ts/`
