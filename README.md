# socialsight

Command-line tool for generating images and videos with [SocialSight](https://socialsight.ai) models.

> **Status:** early scaffolding (ENG-261). Commands below describe the intended v1 surface; most are not implemented yet.

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
socialsight auth login --key <api-key>

socialsight model list
socialsight generate image --model <id> --prompt "..." --wait
socialsight jobs get <job_id>
```

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
