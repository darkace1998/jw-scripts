# Copilot instructions for `jw-scripts`

## Build, test, and lint commands

Use these from the repository root:

```bash
# Download deps
go mod download
go mod verify

# Build primary CLI binaries only (matches README usage)
go build -o bin/ ./cmd/...

# Build everything (matches CI behavior)
go build -v -o bin/ ./...

# Run full test suite
go test -v ./...

# Run race-enabled tests (CI-equivalent test mode)
go test -v -race ./...

# Run one package
go test -v ./internal/api

# Run one test by name
go test -v ./internal/api -run TestGetBestVideo

# Run one CLI test by name
go test -v ./cmd/jwb-index -run TestJwbIndexHelp

# Lint (uses .golangci.yml)
golangci-lint run --timeout=5m
```

## High-level architecture

- This is a Go CLI application repository with four user-facing binaries:
  - `cmd/jwb-index`: indexes JW media categories, optionally downloads media/subtitles, and optionally emits playlists/output files.
  - `cmd/jwb-music`: same pipeline shape as `jwb-index`, specialized for music categories plus JW Broadcasting MP3 handling.
  - `cmd/jwb-offline`: offline playback loop for downloaded videos.
  - `cmd/jwb-books`: publication download/search flow using the `internal/books` stack.
- The core media pipeline for `jwb-index` and `jwb-music` is:
  1. Parse flags into `internal/config.Settings`
  2. Fetch/category-shape media via `internal/api.Client`
  3. Optionally download via `internal/downloader`
  4. Optionally generate output via `internal/output`
- `internal/api` and `internal/books` are separate API stacks:
  - `internal/api`: JW media/category endpoints (`data.jw-api.org`) and media selection logic.
  - `internal/books`: publication/media links (`b.jw-cdn.org/apis/pub-media/GETPUBMEDIALINKS`) and publication model/downloader.
- `internal/player` is independent of the downloader pipeline and persists playback state in `dump.json` inside the selected work directory.
- `cmd/*analysis` binaries are diagnostic utilities used for API/content investigation and are not part of release assets (release workflow publishes only `jwb-index`, `jwb-offline`, `jwb-books`, `jwb-music`).

## Key conventions specific to this codebase

- Shared behavior is driven by a mutable `*config.Settings` passed through API/downloader/output components; preserve this flow when adding flags or behaviors.
- `internal/api.Category.Contents` is intentionally heterogeneous (`[]interface{}` containing `*api.Category` and `*api.Media`); downstream code uses type assertions in multiple packages.
- `jwb-index` and `jwb-music` intentionally share most flag semantics (`--download`, `--mode`, `--import`, `--since`, `--update`, `--friendly`, `--safe-filenames`, etc.). Keep parity unless divergence is deliberate.
- `--update` behavior is compound: code sets append/sort/date-related behavior automatically rather than treating it as an isolated switch.
- Filename handling and Windows compatibility are centralized in `internal/api/client.go` helper functions (`formatFilename`, `makeUniqueFilename`, safe filename behavior).
- Downloader behavior expects `.part` files for resume and may perform checksum/size validation and disk cleanup before final rename.
- CLI parsing is intentionally mixed today: `jwb-index`, `jwb-music`, `jwb-offline` use Cobra; `jwb-books` uses the standard `flag` package.
- Several command tests shell out to `go run` and may exercise live API/network behavior; avoid introducing assumptions that all tests are purely offline unit tests.
