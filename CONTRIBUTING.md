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
- `internal/client/` — generated/hand-written client for the SocialSight API
