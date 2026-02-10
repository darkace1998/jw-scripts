# TODO

A prioritized list of improvements, fixes, and enhancements for the jw-scripts project.

---

## High Priority

### Bugs

- [x] Fix overly permissive API health check in `internal/books/client.go` — returns true for 2xx–4xx status codes (including client errors); should only accept 2xx
- [x] Use `http.StatusOK` instead of hardcoded `200` in `internal/books/client.go` for consistency with the rest of the codebase
- [x] Handle ignored error from `listVideos()` in `internal/player/player.go` — error is silently discarded

### Missing Features

- [x] Implement offline import for `jwb-music` (`cmd/jwb-music/main.go`) and `jwb-index` (`cmd/jwb-index/main.go`)
- [x] Investigate books/publications download support — the books framework works with the real JW.org Publication Media API; actual downloads depend on API availability per publication

### Hardcoded Values

- [x] Parameterize `latestJWBYear` in `internal/api/client.go` — now computed as `time.Now().Year() - jwbYearBase` (no yearly manual update needed)
- [x] Update default player from `omxplayer` (deprecated/removed from most distros) to `mpv`

---

## Medium Priority

### Error Handling

- [ ] Add response body size limit when reading API responses in `internal/books/client.go` (`io.ReadAll` with no size cap)
- [ ] Return errors from symlink creation in `internal/output/writer.go` instead of only logging them
- [ ] Handle `strconv.Atoi` errors when parsing resolution in `internal/api/client.go` instead of silently ignoring them
- [ ] Guard unchecked type assertions (e.g., `item.(*api.Media)`) to prevent potential panics

### Code Quality

- [ ] Extract duplicated `contains()` helper (defined in both `internal/api/client.go` and `internal/player/player.go`) into a shared utility
- [ ] Unify error handling patterns across the codebase — currently a mix of `fmt.Errorf`, log-and-continue, and silent drops
- [ ] Add a comment or constant for magic numbers: `200` (ranking bonus), `100` (subtitle bonus) in `internal/api/client.go`

### Testing

- [ ] Add tests for `internal/player` — currently the only internal package with zero test coverage
- [ ] Add integration tests or CLI smoke tests for `cmd/jwb-index`, `cmd/jwb-music`, `cmd/jwb-books`, and `cmd/jwb-offline`
- [ ] Add benchmark tests for the downloader and API client to track performance

---

## Low Priority

### Documentation

- [ ] Add a `CHANGELOG.md` to track version history and release notes
- [ ] Add a troubleshooting section to `README.md` covering common errors and solutions
- [ ] Document complex flag combinations and add example outputs to `README.md`
- [ ] Clarify that `jwb-books` is a framework/prototype in its documentation

### Project Organization

- [ ] Move analysis/debug tools (`cmd/api-analysis`, `cmd/category-analysis`, `cmd/character-analysis`, `cmd/comprehensive-analysis`, `cmd/media-analysis`, `cmd/subtitle-diagnostic`, `cmd/subtitle-count-test`) into a `cmd/debug/` or `cmd/internal/` subdirectory, or exclude them from release builds
- [ ] Consider adding config file support (e.g., YAML/TOML) alongside CLI flags for persistent settings

### CI / Linting

- [ ] Enable additional golangci-lint checks: `exhaustive`, `nilnil`, `bodyclose`
- [ ] Add `nolintlint` to verify that `#nosec` / `//nolint` comments are still valid
