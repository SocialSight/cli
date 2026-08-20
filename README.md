# socialsight

Command-line tool for generating images and videos with [SocialSight](https://socialsight.ai) models.

> **Status:** all v1 commands work end-to-end (ENG-261). v0.1.0 is tagged; curl and Homebrew installs below both work today. The npm package is built and tested but not yet published to the registry -- see [CONTRIBUTING.md](CONTRIBUTING.md#npm-package).

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
socialsight auth login --key <api-key>   # or omit --key to be prompted
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

The API key comes from the SocialSight web dashboard. It's saved to
`~/.socialsight/config`; set `SOCIALSIGHT_API_KEY` to override it (e.g. in CI).

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
