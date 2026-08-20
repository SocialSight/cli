# Contributing

## Setup

```bash
git clone https://github.com/SocialSight/cli.git
cd cli
go build -o socialsight ./cmd/socialsight
```

## Workflow

1. Open an issue or pick up an existing one before starting non-trivial work.
2. Branch off `main`.
3. Keep PRs scoped to one change; include tests for new behavior.
4. Run before pushing:
   ```bash
   go build ./...
   go vet ./...
   go test ./...
   golangci-lint run
   ```
5. Open a PR — CI runs the same checks.

## Project layout

- `cmd/socialsight/` — main package / entrypoint
- `internal/cli/` — command definitions (cobra)
- `internal/client/` — generated client for the SocialSight API (`client.gen.go`), plus hand-written helpers
- `openapi/socialsight.json` — the subset of SocialSight's public OpenAPI spec (`GET /openapi.json` on `services/api`) this CLI's client is generated from

## Regenerating the API client

After `openapi/socialsight.json` changes (or to pick up new endpoints), re-run:

```bash
go generate ./...
```

This runs `oapi-codegen` (tracked as a `tool` dependency in `go.mod`, no separate install needed) and rewrites `internal/client/client.gen.go`.
