# Contributing to JW Scripts

Thank you for your interest in contributing to JW Scripts!

## Development Setup

1. Ensure you have Go 1.24 or later installed
2. Clone the repository
3. Run `go mod download` to install dependencies
4. Build the project: `go build -o bin/ ./...`

## Testing

Run tests with:
```bash
go test -v ./...
```

For race condition testing:
```bash
go test -v -race ./...
```

## Code Quality

This project uses automated code quality checks:

### Linting
The project uses `golangci-lint` for code linting. Configuration is in `.golangci.yml`.

To run locally:
```bash
golangci-lint run
```

### Formatting
Code should be formatted with `gofmt` and imports organized with `goimports`.

## Continuous Integration

The project uses GitHub Actions for CI/CD:

### CI Workflow (`.github/workflows/ci.yml`)
- Runs on push to main/master and pull requests
- Tests against multiple Go versions (1.24, 1.25)
- Includes linting, security scanning, and race condition testing
- Builds binaries and tests basic functionality

### Release Workflow (`.github/workflows/release.yml`)
- Triggered by pushing tags matching `v*` pattern
- Builds binaries for multiple platforms:
  - Linux (amd64, arm64)
  - Windows (amd64, arm64)
  - macOS (amd64, arm64)
- Creates GitHub releases with binary assets
- Generates checksums for verification

## Creating a Release

To create a new release:

1. Tag your commit with a version:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. The release workflow will automatically:
   - Build binaries for all platforms
   - Create a GitHub release
   - Upload binaries as release assets
   - Generate checksums

3. You can also manually trigger a release from the GitHub Actions tab

## Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Ensure all CI checks pass
6. Submit a pull request

The CI workflow will automatically run tests and checks on your pull request.