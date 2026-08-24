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
- `internal/oauth/` — the browser-based OAuth 2.0 Authorization Code + PKCE login flow against the MCP service (see below)
- `internal/config/` — the local credential file (`~/.socialsight/config`), holding either an API key or an OAuth session
- `openapi/socialsight.json` — the subset of SocialSight's public OpenAPI spec (`GET /openapi.json` on `services/api`) this CLI's client is generated from

## Regenerating the API client

After `openapi/socialsight.json` changes (or to pick up new endpoints), re-run:

```bash
go generate ./...
```

This runs `oapi-codegen` (tracked as a `tool` dependency in `go.mod`, no separate install needed) and rewrites `internal/client/client.gen.go`.

## Cutting a release

Pushing a tag matching `v*` (e.g. `v0.1.0`) triggers `.github/workflows/release.yml`,
which runs [GoReleaser](https://goreleaser.com) (config: `.goreleaser.yaml`) to
cross-compile `socialsight` for darwin/linux/windows × amd64/arm64, archive
each binary, and publish them as GitHub Release assets with a checksums file
and changelog. Version/commit/date are baked into the binary via `-ldflags`
(`socialsight version` reads them from `internal/cli/version.go`).

To dry-run the whole pipeline locally without pushing a tag or publishing
anything (useful after changing `.goreleaser.yaml`):

```bash
brew install goreleaser   # if not already installed
goreleaser release --snapshot --clean --skip=publish
```

This is also the foundation the curl installer (ENG-270), Homebrew tap
(ENG-271), and npm wrapper (ENG-272) pull release assets from.

## Homebrew tap

The `brews:` section in `.goreleaser.yaml` pushes an updated formula to
[SocialSight/homebrew-tap](https://github.com/SocialSight/homebrew-tap) on
every release, so `brew install socialsight/tap/socialsight` stays current.
This needs a `HOMEBREW_TAP_GITHUB_TOKEN` repository secret here (a
fine-grained PAT with write access to that repo) -- without it, `skip_upload`
is templated to skip the tap update gracefully rather than failing the whole
release, so the GitHub Release itself and `install.sh` still work either way.

Note: GoReleaser soft-deprecates `brews:` in favor of `homebrew_casks:` (see
https://goreleaser.com/deprecations#brews). `brews:` still works fully today
(confirmed by a real `brew install`/`test`/`audit` run against the tap), and
migrating would mean restructuring the tap repo (`Casks/` instead of
`Formula/`, different Ruby DSL) -- worth revisiting if it's ever actually
removed, not before.

## Testing install.sh without a real release

`install.sh` downloads from `https://github.com/SocialSight/cli/releases/...`,
so exercising it without tagging a real release means pointing it at a stand-in
server instead. After a local snapshot build (see above), serve `dist/` and
point a scratch copy of the script at it:

```bash
python3 -m http.server 8931 --directory dist &
sed 's#base_url="https://github.com/${repo}/releases/download/${version}"#base_url="http://127.0.0.1:8931"#' install.sh > /tmp/install-test.sh
PREFIX=/tmp/socialsight-test sh /tmp/install-test.sh --version v0.0.0-SNAPSHOT-<commit>   # matches the snapshot's version string
```

(`<commit>` is the short SHA GoReleaser used in the snapshot version, visible
in the `dist/*.tar.gz` filenames.)

## npm package

`npm/` is a thin wrapper published as `@socialsight/cli`: its `postinstall`
downloads the release matching `npm/package.json`'s own `version` field
(checksum-verified against that release's `checksums.txt`), extracts it with
the system `tar` (present by default on macOS, Linux, and Windows 10+ --
bsdtar there also handles `.zip`, so this needs no extra npm dependency), and
installs it as `bin/socialsight-bin` next to the `bin/socialsight.js` shim.

**`npm/package.json`'s version must match an already-published GitHub
release** (e.g. only bump it to `0.2.0` after `v0.2.0` is tagged and released,
not before) -- `.github/workflows/ci.yml`'s `npm-smoke` job packs and
installs it on every push specifically to catch that class of drift early.

Publishing is automated: `.github/workflows/release.yml` bumps
`npm/package.json`'s version to match the tag, runs `npm publish
--access public`, then commits that version bump back to `main` so the repo
never drifts from what's actually published. This uses npm's
[Trusted Publisher](https://docs.npmjs.com/trusted-publishers) (OIDC) --
no `NPM_TOKEN` secret, just `permissions: id-token: write` in the workflow.
`@socialsight/cli`'s Trusted Publisher on npmjs.com is configured for the
`SocialSight/cli` repo, workflow file `release.yml`, action `npm publish`.

To publish by hand instead (e.g. Trusted Publisher misconfigured, or from a
machine without OIDC):

```bash
cd npm
npm version <new-version> --no-git-tag-version   # after the matching vX.Y.Z release exists
npm publish --access public
```

## Browser login (`internal/oauth`)

`socialsight auth login` (no flags) runs a real OAuth 2.0 Authorization Code
+ PKCE flow against the MCP service's OAuth proxy (`mcp.socialsight.ai` /
`staging-mcp.socialsight.ai` -- separate hosts from the API, derived in
`internal/client.MCPBaseURL()`), which forwards to Clerk. This is the same
mechanism used when wiring SocialSight up as an MCP server in Claude/other
AI clients (`services/mcp/app/oauth_facade.py` in `socialsight-backend`).
See ENG-285 for the full design writeup.

Key decisions, confirmed against real staging rather than assumed:

- **No client_id caching.** A fresh OAuth client is dynamically registered
  (`POST /oauth2/register`) on every login, scoped to that run's ephemeral
  loopback redirect URI. Caching a client_id across logins would mean the
  registered `redirect_uris` wouldn't match a new run's port, and RFC 8252
  doesn't guarantee authorization servers ignore the port when matching --
  registering fresh avoids that ambiguity entirely, and Clerk's DCR endpoint
  is cheap and designed for this per-session usage (it's how MCP clients
  like Claude Desktop already use it).
- **Scope is `email offline_access profile`, not `openid email
  offline_access`.** The discovery metadata's `scopes_supported` lists
  `openid`, but requesting it is actually rejected for dynamically
  registered clients ("not allowed to request scope 'openid'") -- caught by
  testing against real staging, not from reading the metadata alone.
- **No token refresh in v1.** Clerk's OAuth access tokens last 1 day and
  refresh tokens never expire (confirmed directly from Clerk's docs), so
  "run `socialsight auth login` again" on expiry (via the same 401/403
  handling every authenticated command shares, see `authError` in
  `internal/cli/client.go`) is a fine v1 UX. The refresh token is still
  stored (`internal/config.Credential.RefreshToken`) since it costs nothing
  and never expires -- auto-refresh is a straightforward follow-up if this
  becomes annoying in practice.
- **No device-code flow.** Clerk's proxy only supports
  `response_types_supported: ["code"]`; the loopback-redirect pattern
  requires the browser and the CLI to share a machine. For SSH/headless
  boxes, `--key`/`--paste` remain the only path -- not a gap to fill later.

### Testing this without a real browser

The mechanics (PKCE, loopback listener, discovery, registration, token
exchange) are fully covered by `internal/oauth`'s tests against fake
`httptest` servers -- including a full end-to-end flow test where the
"browser" is just an `http.Get` that follows the fake server's redirect
chain onto the CLI's own loopback listener, exactly as a real browser
redirect would, with the fake token endpoint independently re-deriving the
PKCE challenge from the verifier to confirm the real cross-request math
holds. What can't be scripted is a human actually completing Clerk's hosted
login page -- verify that by hand:

```bash
export SOCIALSIGHT_API_BASE_URL=https://staging-api.socialsight.ai
go run ./cmd/socialsight auth login
```
