# shellodex

[![Go Version](https://img.shields.io/github/go-mod/go-version/ripnet/shellodex)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/ripnet/shellodex)](https://github.com/ripnet/shellodex/releases/latest)

A fast, keyboard-driven TUI for managing SSH and Telnet connections. Inspired by [sshm](https://github.com/nicholasgasior/sshm), built with the [Charmbracelet](https://charm.sh) library stack.

<!--
TODO: Add a screenshot here once the UI is stable.
![shellodex screenshot](docs/screenshot.png)
-->

## Features

- **Launcher-first UX** — your hosts front and center; connect with a keypress
- **Fuzzy search** — press `/` to filter hosts by name, hostname, group, or tag
- **Groups & tags** — organize hosts with nested groups and freeform tags
- **SSH & Telnet** — full support for both protocols with custom ports
- **Jump hosts** — built-in SSH ProxyJump support
- **Credential management** — store and assign credentials to hosts *(in progress)*
- **Config sync** — push/pull/mirror your config via rclone to any cloud backend, with optional auto-sync on startup
- **Self-updating** — run `shellodex --update` to get the latest version

## Installation

### One-line install (Linux & macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/ripnet/shellodex/main/install.sh | bash
```

The script will:
- Download the correct binary for your OS and architecture
- Install it to `~/.local/bin/shellodex`
- Offer to add `~/.local/bin` to your PATH if needed
- Offer to add a short alias (`s`) to your shell config

To update to a newer release, re-run the same command, or use the built-in updater:

```bash
shellodex --update
```

### Quick alias

If you skipped the alias during install, add it manually to `~/.zshrc` or `~/.bashrc`:

```bash
alias s='shellodex'
```

Then reload your shell: `source ~/.zshrc`

### Install with Go

If you have Go 1.21+ installed:

```bash
go install github.com/ripnet/shellodex/cmd/shellodex@latest
```

### Manual download

Download a pre-built binary for your platform from the [Releases](https://github.com/ripnet/shellodex/releases) page, extract it, and move it somewhere on your `$PATH`:

```bash
# Example for Linux amd64
curl -fsSL https://github.com/ripnet/shellodex/releases/latest/download/shellodex_linux_amd64.tar.gz | tar -xz
mv shellodex ~/.local/bin/
```

## Usage

```
shellodex [flags]

Flags:
  -config string   path to config file (default: platform config dir)
  -version         print version and exit
  -update          download and install the latest release
```

### Key bindings

| Key | Action |
|-----|--------|
| `enter` | Connect to selected host |
| `/` | Search / filter hosts |
| `a` or `n` | Add new host |
| `e` | Edit selected host |
| `d` | Delete selected host |
| `c` | Manage credentials |
| `g` | Manage groups |
| `s` | Settings (sync config) |
| `tab` | Switch to tree view |
| `j` / `k` or `↑` / `↓` | Navigate list |
| `ctrl+r` | Sync config via rclone |
| `q` or `ctrl+c` | Quit |

## Configuration

shellodex stores its config at the platform default location:

| Platform | Path |
|----------|------|
| Linux | `~/.config/shellodex/config.json` |
| macOS | `~/Library/Application Support/shellodex/config.json` |

Use `-config /path/to/config.json` to override.

## Config sync

shellodex can sync your config file to any cloud storage backend supported by [rclone](https://rclone.org) (Google Drive, S3, Dropbox, etc.).

### Setup

1. [Install rclone](https://rclone.org/install/) and configure a remote:
   ```bash
   rclone config
   ```
2. Open shellodex settings (`s`) and set the **rclone Remote** to a path on your remote, e.g. `gdrive:shellodex` or `s3:mybucket/shellodex`.
3. Choose a sync direction and optionally enable **Sync on startup**.

### Sync directions

| Direction | Command | Effect |
|-----------|---------|--------|
| Push | `rclone copy` | Copies local config → remote |
| Pull | `rclone copy` | Copies remote config → local |
| Mirror | `rclone sync` | Makes remote identical to local (deletes remote extras) |

### Triggering a sync

- **Ctrl+R** — sync manually at any time from the main screen; result overlay shows on completion
- **Sync on startup** — when enabled in settings, syncs automatically when shellodex launches; the list refreshes in place and the overlay is suppressed on success (errors are always shown)

## Building from source

```bash
git clone https://github.com/ripnet/shellodex.git
cd shellodex
make build          # build ./shellodex with version info injected
make install        # install to $GOPATH/bin
make test           # run tests
make snapshot       # build all platforms locally (requires goreleaser)
```

## Releasing a new version

Tag a commit and push — GitHub Actions handles the rest:

```bash
git tag v1.0.1
git push origin v1.0.1
```

GoReleaser will build binaries for Linux and macOS (amd64 + arm64) and publish them as a GitHub Release.

## License

MIT — see [LICENSE](LICENSE).
