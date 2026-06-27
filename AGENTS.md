# AGENTS.md

## Verification Checklist

Run these before submitting any change. All must pass with zero output (where applicable).

### Build

```bash
go build ./cmd/server
go build ./cmd/scraper
```

### Tests

```bash
go test ./internal/...
```

### Static analysis

```bash
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
```

### LSP diagnostics

Open the workspace in an editor with `gopls`. Confirm zero diagnostics across all files. Key analyzers that must be clean:

- `go build` — compilation errors
- `go vet` — suspicious constructs
- `staticcheck` — unused code, style violations (ST1005, U1000, etc.)
- `sqlrowserr` — missing `rows.Err()` after `rows.Next()` loops

### Formatting

```bash
go fmt ./...
```

### Smoke test (optional, when touching API)

```bash
# Start server
go run ./cmd/server &
sleep 3

# Core endpoints
curl -s http://localhost:8080/api/zones | head -c 200
curl -s "http://localhost:8080/api/solat/SGR01?month=6&year=2026" | head -c 200
curl -s http://localhost:8080/api/last-update | head -c 200

# GPS detection
curl -s http://localhost:8080/api/zones/3.068498/101.630263

# Error paths
curl -s "http://localhost:8080/api/solat/XXXXX?month=6&year=2026"

kill %1
```

## Project conventions

- **Go >= 1.25** required (gofiber v3 minimum).
- Fiber v3: handlers take `fiber.Ctx` (interface), not `*fiber.Ctx` (pointer).
- Import path: `github.com/gofiber/fiber/v3` (not `/v2`).
- Error strings lowercase per Go convention (ST1005).
- All `sql.Rows` loops must check `rows.Err()` after iteration.
- No unused types, variables, or imports (U1000).
- Zone codes must match JAKIM's actual codes (e.g. `SWK`, not `SRW`).
- Embedded `zones_data.json` is the authoritative zone list — validate against JAKIM API before changing.
