# socialsight

Command-line tool for generating images and videos with [SocialSight](https://socialsight.ai) models.

> **Status:** early scaffolding (ENG-261). `auth` is implemented; `model`/`generate`/`jobs` below describe the intended v1 surface and aren't wired up yet.

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

socialsight model list
socialsight generate image --model <id> --prompt "..." --wait
socialsight jobs get <job_id>
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
