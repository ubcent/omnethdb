# Releasing OmnethDB

This repository ships release binaries through GitHub Releases.

The release pipeline is built around:

- [`.goreleaser.yaml`](../.goreleaser.yaml)
- [`.github/workflows/release.yml`](../.github/workflows/release.yml)

## What Gets Published

Each tagged release publishes archives containing:

- `omnethdb`
- `omnethdb-mcp`
- `README.md`
- Claude Code starter files from `examples/claude-code/`

Target platforms:

- macOS `amd64`, `arm64`
- Linux `amd64`, `arm64`
- Windows `amd64`, `arm64`

The pipeline also publishes `checksums.txt`.

## Release Trigger

Push a semver-style tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

That triggers the GitHub Actions workflow, which:

1. checks out full git history
2. sets up Go from `go.mod`
3. runs `go test ./...`
4. builds release binaries with GoReleaser
5. uploads archives and checksums to the GitHub Release for that tag

## Local Dry Run

Before cutting a real release, run:

```bash
goreleaser release --snapshot --clean
```

That verifies the release config locally without publishing a GitHub Release.

## Expected Consumer UX

After downloading and unpacking a release archive, the normal runtime entrypoints are:

```bash
omnethdb help
omnethdb-mcp --workspace /absolute/path/to/workspace
```

The MCP config should point directly at the binary, not `go run`.

## Install Script

This repo also ships an install helper:

- [`scripts/install.sh`](../scripts/install.sh)

Typical usage:

```bash
curl -fsSL https://raw.githubusercontent.com/ubcent/omnethdb/main/scripts/install.sh | sh
```

Specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/ubcent/omnethdb/main/scripts/install.sh | VERSION=v0.1.0 sh
```

Custom install directory:

```bash
curl -fsSL https://raw.githubusercontent.com/ubcent/omnethdb/main/scripts/install.sh | INSTALL_DIR=/usr/local/bin sh
```

The script resolves:

- latest release tag by default
- current OS and architecture
- the matching release archive for that platform

## Notes

- The current pipeline is intentionally simple: release archives plus checksums.
- If we later want a smoother install path, the next step should be a Homebrew tap, not more `go run` wrappers.
