# socialsight

Command-line tool for generating images and videos with [SocialSight](https://socialsight.ai) models.

> **Status:** all v1 commands work end-to-end (ENG-261). v0.1.0 is tagged; curl, Homebrew, and npm installs below all work today. Both the Homebrew tap and npm publishing are now automated on every future release -- see [CONTRIBUTING.md](CONTRIBUTING.md#npm-package).

## Install

```bash
# curl
curl -fsSL https://raw.githubusercontent.com/SocialSight/cli/main/install.sh | sh

# Homebrew
brew install socialsight/tap/socialsight

# npm
npm install -g @socialsight/cli
```

## Usage

```bash
socialsight auth login                   # opens your browser to sign in
socialsight auth login --key <api-key>   # or --paste to be prompted -- for CI/headless use
socialsight auth whoami
socialsight auth logout

socialsight model list [--type image|video]
socialsight model info <model_id>

socialsight generate image --model <id> --prompt "..." [--aspect-ratio ...] [--quality ...] [--wait]
socialsight generate video --model <id> --prompt "..." [--duration ...] [--resolution ...] [--wait]
socialsight generate cost image --model <id> ...   # preview credit cost before running
socialsight generate cost video --model <id> ...

socialsight jobs get <job_id>
socialsight jobs wait <job_id>
```

`auth login` signs in via your browser by default (the same OAuth flow used
when wiring SocialSight up as an MCP server) and stores the session in
`~/.socialsight/config`. Sessions last a day; once expired, just run
`socialsight auth login` again -- there's no auto-refresh (yet). For CI or
headless boxes without a browser, use `--key <api-key>` (from the SocialSight
web dashboard) or `--paste` to be prompted; set `SOCIALSIGHT_API_KEY` to
override the stored credential entirely.

Add `--wait` to `generate image`/`generate video` to block until the job
finishes instead of just printing its ID (shows a spinner on an interactive
terminal); `jobs wait` does the same for a job you already have the ID for.
Both accept `--wait-interval`/`--wait-timeout` to override the 3s/10m
defaults. Add the global `--json` flag to any command for machine-readable
output instead of text.

Run `socialsight --help` for the full command list.

## Development

Requires Go 1.25+.

```bash
go build -o socialsight ./cmd/socialsight
go test ./...
go vet ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
