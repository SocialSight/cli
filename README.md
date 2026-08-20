# socialsight

Command-line tool for generating images and videos with [SocialSight](https://socialsight.ai) models.

> **Status:** core v1 commands work end-to-end (ENG-261). `--wait`/`--json` flags and installers (curl/brew/npm) are still pending.

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

socialsight generate image --model <id> --prompt "..." [--aspect-ratio ...] [--quality ...]
socialsight generate video --model <id> --prompt "..." [--duration ...] [--resolution ...]
socialsight generate cost image --model <id> ...   # preview credit cost before running
socialsight generate cost video --model <id> ...

socialsight jobs get <job_id>
socialsight jobs wait <job_id>   # polls until completed/error (fixed interval for now)
```

The API key comes from the SocialSight web dashboard. It's saved to
`~/.socialsight/config`; set `SOCIALSIGHT_API_KEY` to override it (e.g. in CI).

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
