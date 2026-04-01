# pixel-id

Generate unique 64-bit snowflake IDs and render deterministic pixel avatars from them. Same ID = same avatar, in Go and TypeScript.

```
┌──────┬───────────────────────┬────────────────┬──────────────┐
│ Sign │ Timestamp (41 bits)   │ Machine (10)   │ Sequence (12)│
│  0   │ ms since 2025-01-01   │ 0..1023        │ 0..4095      │
└──────┴───────────────────────┴────────────────┴──────────────┘
         ↓ FNV-1a hash + bit extraction
    ┌─────────┐
    │ # . # . │  ← deterministic pixel avatar
    │ . # # . │     (vertically symmetric)
    │ # # # # │
    │ . # # . │
    │ # . # . │
    └─────────┘
```

## Go

### Install

```bash
go get github.com/matthewblackburn/pixel-id/go
```

### Generate IDs

```go
import pixelid "github.com/matthewblackburn/pixel-id/go"

gen := pixelid.NewGenerator(pixelid.WithMachineID(1))
id, err := gen.Generate() // int64 snowflake ID
```

### Render avatars

```go
svg := pixelid.RenderSVG(id)                          // SVG string
png, err := pixelid.RenderPNG(id, pixelid.WithSize(128)) // PNG bytes

// Custom grid size
svg = pixelid.RenderSVG(id, pixelid.WithGrid(8, 8))
```

### Parse IDs

```go
ts, machineID, sequence := pixelid.ParseID(id)
```

## TypeScript / React

### Install

```bash
npm install pixel-id
```

### Render avatars

```typescript
import { derive, renderSVG } from "pixel-id";

const data = derive("123456789012345678"); // AvatarData
const svg = renderSVG("123456789012345678"); // SVG string
```

### React component

```tsx
import { PixelAvatar } from "pixel-id/react";

<PixelAvatar id="123456789012345678" size={64} />
<PixelAvatar id={someBigInt} gridWidth={8} gridHeight={8} />
```

### API

```typescript
// Core (zero dependencies)
derive(id: string | bigint, gridWidth?: number, gridHeight?: number): AvatarData
renderSVG(id: string | bigint, options?: AvatarOptions): string

// React (peer dep on react)
PixelAvatar(props: { id: string | bigint } & AvatarOptions): JSX.Element
```

IDs must be passed as strings or BigInts — JavaScript `Number` loses precision beyond 2^53.

## Cross-language determinism

The avatar algorithm (FNV-1a hash, bit extraction, palette lookup) is identical in Go and TypeScript. Both test suites assert against shared vectors in `spec/vectors.json`.

**Design principle:** The algorithm is immutable per major version. Changing how an ID maps to an avatar is a semver major bump.

## Example (Dockerized)

The `example/` directory contains a full-stack demo: React frontend + Go backend, orchestrated with Docker Compose.

```bash
cd example
make up
# or: docker compose up --build
```

Then open **http://localhost:4200** in your browser. Click "Generate ID" to hit the Go backend, which returns a snowflake ID. The frontend displays the ID, its parsed components, and both SVG and PNG avatars rendered by the backend.

```
Browser (localhost:4200)
    │
    ├─ React app (Vite dev server)
    │
    └─ /api/* ──proxy──► Go API (localhost:4100)
                           ├─ POST /api/id         → generate ID
                           ├─ GET  /api/avatar/{id}.svg → SVG avatar
                           └─ GET  /api/avatar/{id}.png → PNG avatar
```

Ports are configurable: `API_PORT=9000 WEB_PORT=9001 docker compose up`

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `size` | 256 | Output size in pixels (PNG max: 2048) |
| `gridWidth` | 5 | Avatar grid width |
| `gridHeight` | 5 | Avatar grid height |
| `padding` | 0.08 | Padding as fraction of size |

## License

MIT
