# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **`--metadata` now writes an `.nfo` file next to each download instead of modifying the media files.** NFO is the Kodi XML format that Jellyfin, Emby and Kodi read natively (Plex via an NFO agent such as XBMCnfoMoviesImporter). Each file is described as a `<movie>` with title, release date, runtime, category, studio and source URL. MP3/MP4 files are no longer rewritten with ID3 tags or MP4 atoms, so their size and MD5 match the API again. JSON sidecar files from earlier versions are removed; tags already embedded by earlier versions are left in place. Media servers read music metadata only from embedded tags, so NFO files next to MP3s are informational.
- `--quiet`/`-q` is now a counter as documented: use `-q`, `-qq` or `--quiet=2`. The old `-q 2` form is rejected with a hint instead of treating `2` as the download directory.
- Subtitles are named after their video (`<video name>.vtt`) so players and media servers load them automatically. Existing subtitle files keep their old names and are downloaded again under the new name.
- Commands exit with a non-zero status when anything failed (categories that could not be indexed, failed downloads, missing publications) after finishing all work that could still be done. A playlist file is not rewritten from an incomplete index unless `--append` is used.
- The Docker image runs commands as `PUID:PGID` (default `1000:1000`) instead of root, and moves a root-owned data directory over once. Set `PUID=0 PGID=0` to keep running as root. Arguments after the image name are now run as a one-off command.
- Downloaded files, directories, playlists and metadata files are readable by other users (for media servers running as a different user).
- `jwb-books` computes year-based publication codes (daily text, publications index, circuit assembly, convention) from the current date and falls back to the previous edition, instead of using hard-coded 2024/2025 codes. The `magazines` category downloads the latest Watchtower and Awake! issues, or the one given with the new `--issue YYYYMM` flag.
- Diagnostic tools moved from `cmd/` to `tools/`, so `go build ./cmd/...` and the Docker image contain only the four commands.

### Added
- `--version` for all commands; release and Docker builds embed the version.
- `jwb-books`: `--issue`, `--quiet`, `--limit-rate` and `--version` flags.
- API requests send a User-Agent and are retried on network errors, HTTP 429 and 5xx responses.
- CI runs `govulncheck`; Dependabot also updates GitHub Actions and Docker base images.

### Fixed
- Downloads could hang forever on a stalled connection; connection setup and response headers now time out and a download that receives no data for 2 minutes is aborted.
- A complete `.part` file failed with HTTP 416 on every run and was never finished; it is now finished, and a stale partial file is downloaded again.
- `--checksum` had no effect on new downloads; new downloads are now checked against the API size (and MD5 with `--checksum`) before they are moved into place.
- `--import` failed for every file when downloading was enabled (the default in `jwb-music`); imported files are now copied into the library.
- Media that appears in several categories was downloaded once per category under different names; it is now downloaded once.
- `--friendly` names for media with the same title depended on the order of API results and could swap between runs; the oldest item now keeps the plain name. Names are also compared case-insensitively for Windows and macOS.
- A media item without subtitles could leave its subtitle name to another video with the same title, so players showed the wrong subtitles.
- `--free` aborted the whole run when no MP4 file was left to delete; the download is now skipped with a clear message. Removing an old video also removes its subtitle and metadata files.
- Filesystem mode stopped at the first category name containing `/`; names from the API are sanitized so they cannot create paths outside the work directory, and one failing link no longer stops the others.
- Titles with line breaks could inject extra entries into M3U/TXT playlists.
- Playlists are replaced atomically, write errors are reported, and paths are relative to the playlist file (also with `--output sub/list.m3u`).
- An invalid `--mode` is reported before indexing and downloading start.
- `jwb-books` filenames from the API are sanitized (a crafted URL could write outside the output directory on Windows), downloads go through a `.part` file, and errors go to stderr with a non-zero exit status.
- `jwb-offline` had a data race when saving its state on Ctrl+C/SIGTERM and printed a false shutdown message on errors; the player is now stopped cleanly and the position is saved. A deleted video from the saved state is skipped.
- The Docker container stopped before starting the schedule when the first run failed.
- The Docker base image moved from the end-of-life Alpine 3.20 to Alpine 3.22; `supercronic` is built from source and checked against the Go checksum database instead of being downloaded without verification.
- Release builds set `main.version` (the flag was a no-op), manual release runs build the requested tag, the tag input can no longer inject shell commands, and pre-release tags are published as pre-releases without updating the Docker `latest` tag.
- CI pins GitHub Actions to commit SHAs and golangci-lint to a fixed version; Codecov uploads use the current action inputs.
- `go.mod` requests a patched Go 1.25 toolchain; builds on platforms other than Linux, macOS and Windows no longer fail (`--free` reports that it is unsupported there).

## [v1.7.1] - 2026-08-04

### Added
- Added `--metadata` flag to `jwb-index`, `jwb-music`, and `jwb-books` that embeds metadata directly in downloaded media files: ID3v2.4 tags for MP3 (title, album/category, artist, date, source URL) and iTunes-style metadata atoms for MP4 (with correct chunk-offset patching for faststart files). Formats that cannot carry embedded tags (PDF, EPUB, RTF, BRL) get a JSON sidecar file (`<filename>.json`) instead. Embedding is idempotent (unchanged files are not rewritten), preserves file modification times, and stale sidecars from earlier runs are cleaned up once metadata is embedded.
- Added new `internal/metadata` package (JSON sidecars, ID3v2.4 writer, MP4 atom writer) with tests.
- Implemented the disk-space warning for `--free` in `jwb-index` and `jwb-music`: a notice that old MP4 files will be deleted when space runs low, plus a warning when free space is already below the configured limit. The previously broken `--no-warning` flag (default was `true` and the setting was never read) now actually suppresses these warnings.
- `jwb-books` now downloads all files of a publication in the requested format (e.g. every MP3 track of an audio publication) instead of only the first one, skips files that are already fully downloaded, and validates MD5 checksums when the API provides them (corrupt downloads are removed).

### Changed
- With `--metadata` enabled, `--fix-broken` size checks accept files that grew due to embedded tags, and checksum verification is skipped for such files (embedding changes the file contents, so the API checksum can no longer match).

### Fixed
- Fixed `--append` being a no-op in `jwb-index` and `jwb-music`: playlist output files (txt/m3u/html) were always overwritten. Append mode now preserves existing entries, deduplicates against them, and keeps a single header/footer. This also fixes `--update`, which implies `--append`.
- Fixed `--clean-symlinks` being a no-op in filesystem mode: stale symlinks in the data directory (and stale home-category links in the work directory) are now removed before the structure is recreated.
- Fixed playlist output files being truncated as soon as the writer was created; files are now only written once indexing succeeds, so a failed run no longer destroys an existing playlist.
- Fixed failed subtitle downloads leaving truncated `.vtt` files behind that were treated as complete on subsequent runs; subtitles are now downloaded to a `.part` file and renamed on success.
- Fixed playlist entries resolving to `.` instead of the media URL for media items without a local filename.
- Fixed offline import (`--import`) creating a broken symlink in filesystem mode when `--friendly` was not set.
- Fixed network-dependent unit tests failing in offline environments; they now skip gracefully when the JW.org API is unreachable.

## [v1.7.0] - 2026-04-20

### Added
- Added Docker runtime support (`Dockerfile`, `.dockerignore`, `docker-entrypoint.sh`) with cron-driven automation via environment variables.
- Added Docker usage documentation at `docs/docker.md`.
- Added `.github/workflows/docker.yml` for Docker build/publish automation.

### Changed
- Updated Docker workflow publishing policy to push GHCR images only on version tags (`v*`), not on branch pushes.

### Documentation
- Updated `README.md` and `CONTRIBUTING.md` with Docker runtime and Docker workflow details.

## [v1.6.4] - 2026-04-20

### Added
- Added `--output` / `-o` flag to `jwb-index` and `jwb-music` for `txt`, `m3u`, and `html` modes.
- Added default output filename behavior in playlist modes (`playlist.txt`, `playlist.m3u`, `playlist.html`) when `--output` is not set.
- Added regression test coverage for default output filename behavior in `internal/output`.
- Added `.github/copilot-instructions.md` to improve future Copilot-assisted development in this repository.

### Changed
- Updated CI to run on pull requests targeting `main` and `master` in addition to existing triggers.
- Updated `--latest` config test expectations to match current behavior (past 31 days through end of today) and use UTC-stable calculations.
- Reworked `README.md` for a more professional structure and moved command-specific markdown docs into `docs/`.

### Fixed
- Fixed playlist output modes failing with `output filename is required for txt mode` when no explicit output filename was provided.
- Fixed stale README link by replacing missing `docs/BOOK_DOWNLOAD_ANALYSIS.md` reference with `docs/jwb-books.md`.

### Documentation
- Updated `CONTRIBUTING.md` to match current Go/CI expectations (Go 1.25+, CI matrix 1.25/1.26, build command guidance).
- Updated `docs/WIKI.md` and `docs/jwb-music.md` flag tables for `--output`, `--latest`, and `--limit-rate` accuracy.

## [v1.6.3] - 2026-03-23

- Previous release (see Git tags and GitHub Releases for full details).
