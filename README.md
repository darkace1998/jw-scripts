# JW Scripts (Go)

[![Go Version](https://img.shields.io/github/go-mod/go-version/darkace1998/jw-scripts)](https://go.dev/)
[![License](https://img.shields.io/github/license/darkace1998/jw-scripts)](./COPYING)
[![CI](https://img.shields.io/github/actions/workflow/status/darkace1998/jw-scripts/ci.yml?branch=master)](https://github.com/darkace1998/jw-scripts/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/darkace1998/jw-scripts)](https://github.com/darkace1998/jw-scripts/releases)
[![Docker](https://img.shields.io/github/actions/workflow/status/darkace1998/jw-scripts/docker.yml?branch=master&label=docker)](https://github.com/darkace1998/jw-scripts/actions/workflows/docker.yml)

`jw-scripts` is a Go CLI suite for indexing, downloading, and playing media from jw.org, plus downloading publication files in multiple formats.

This project is a Go reimplementation of the original Python version: https://github.com/allejok96/jw-scripts

> These access methods are legal according to JW.org terms, but they are not officially supported. Please review [w18.04 30-31](https://wol.jw.org/en/wol/d/r1/lp-e/2018364) and consider official platform apps where available.

## Quick start

### Option 1: Download prebuilt binaries

Download the latest binaries from:
https://github.com/darkace1998/jw-scripts/releases/latest

Supported targets:
- Linux (amd64, arm64)
- Windows (amd64, arm64)
- macOS (amd64, arm64)

### Option 2: Build from source

```bash
go mod download
go build -o bin/ ./cmd/...
```

### Option 3: Run with Docker (scheduled updates)

```bash
docker build -t jw-scripts:latest .
docker run --rm \
  -e CRON_SCHEDULE="0 */6 * * *" \
  -e JW_COMMAND="jwb-index --download --update --lang E /data" \
  -v "$(pwd)/data:/data" \
  jw-scripts:latest
```

Prebuilt image from GitHub Container Registry:

```bash
docker pull ghcr.io/darkace1998/jw-scripts:latest
```

## Commands

### `jwb-index`

Indexes and optionally downloads JW Broadcasting media.

```bash
# Download latest 31-day window in Swedish
./bin/jwb-index --download --latest --lang S

# Generate a playlist file
./bin/jwb-index --mode txt --output playlist.txt
```

### `jwb-music`

Downloads music/audio categories (including optional JW Broadcasting audio).

```bash
# Download all music in English
./bin/jwb-music

# Download selected categories in Spanish
./bin/jwb-music --lang S --category AudioOriginalSongs,SJJChorus
```

### `jwb-books`

Downloads JW publication files and supports search, categories, and format selection.

```bash
# List categories
./bin/jwb-books --list-categories --language E

# Download a category in PDF
./bin/jwb-books --category daily-text --language E --format pdf --output ./books
```

### `jwb-offline`

Plays downloaded local videos with shuffle/replay behavior.

```bash
./bin/jwb-offline /path/to/downloaded/videos
```

## Documentation

- Main command wiki: [docs/WIKI.md](docs/WIKI.md)
- Books command details: [docs/jwb-books.md](docs/jwb-books.md)
- Music command details: [docs/jwb-music.md](docs/jwb-music.md)
- Docker runtime and cron configuration: [docs/docker.md](docs/docker.md)
- Contribution guide: [CONTRIBUTING.md](CONTRIBUTING.md)
- Changelog: [CHANGELOG.md](CHANGELOG.md)

## Development

```bash
# Tests
go test -v ./...

# Race-enabled tests
go test -v -race ./...

# Lint
golangci-lint run --timeout=5m
```

CI and release automation are implemented with GitHub Actions:
- CI: tests, linting, security scan, binary build checks
- Docker: multi-arch image build, published to GHCR on version tags (`v*`)
- Release: multi-platform binaries and archives on `v*` tags

## Legal

JW.org Terms of Use allow:

> distribution of free, non-commercial applications designed to download electronic files (for example, EPUB, PDF, MP3, AAC, MOBI, and MP4 files) from public areas of this site.

Reference: http://www.jw.org/en/terms-of-use/

## License

GNU GPL v3.0 — see [COPYING](COPYING).
