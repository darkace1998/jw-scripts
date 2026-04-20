# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.6.4] - 2026-04-20

### Added
- Added `--output` / `-o` flag to `jwb-index` and `jwb-music` for `txt`, `m3u`, and `html` modes.
- Added default output filename behavior in playlist modes (`playlist.txt`, `playlist.m3u`, `playlist.html`) when `--output` is not set.
- Added regression test coverage for default output filename behavior in `internal/output`.
- Added `.github/copilot-instructions.md` to improve future Copilot-assisted development in this repository.

### Changed
- Updated CI to run on pull requests targeting `main` and `master` in addition to existing triggers.
- Updated `--latest` config test expectations to match current behavior (past 31 days through end of today) and use UTC-stable calculations.

### Fixed
- Fixed playlist output modes failing with `output filename is required for txt mode` when no explicit output filename was provided.
- Fixed stale README link by replacing missing `docs/BOOK_DOWNLOAD_ANALYSIS.md` reference with `jwb-books.md`.

### Documentation
- Updated `CONTRIBUTING.md` to match current Go/CI expectations (Go 1.25+, CI matrix 1.25/1.26, build command guidance).
- Updated `WIKI.md` and `jwb-music.md` flag tables for `--output`, `--latest`, and `--limit-rate` accuracy.

## [v1.6.3] - 2026-03-23

- Previous release (see Git tags and GitHub Releases for full details).
