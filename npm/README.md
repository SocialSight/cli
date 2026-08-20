# @socialsight/cli

npm wrapper for the [SocialSight CLI](https://github.com/SocialSight/cli).
`postinstall` downloads the matching platform binary from the GitHub Release
matching this package's version (checksum-verified) and installs it as
`socialsight-bin` next to the `socialsight` shim.

```bash
npm install -g @socialsight/cli
socialsight auth login
```

Requires `tar` on `PATH` (present by default on macOS, Linux, and Windows 10+).

See the main repo's [README](https://github.com/SocialSight/cli#readme) for
full usage.
