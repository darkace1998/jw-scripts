# GitHub Actions CI/CD Examples

## Example: Creating a Release

### Automatic Release (Recommended)
1. Create and push a git tag:
```bash
git tag v1.0.0
git push origin v1.0.0
```

2. The release workflow will automatically:
   - Build binaries for 6 platform/architecture combinations
   - Create a GitHub release with generated release notes
   - Upload binaries and checksums as release assets

### Manual Release
1. Go to the GitHub Actions tab in your repository
2. Select the "Release" workflow
3. Click "Run workflow"
4. Enter the desired tag version (e.g., `v1.0.1`)
5. Click "Run workflow"

## Example: CI Workflow Triggers

The CI workflow runs automatically on:

- **Push to main/master branch**:
```bash
git push origin main
```

- **Pull requests**:
```bash
# CI runs automatically when PR is opened/updated
gh pr create --title "My feature" --body "Description"
```

- **Weekly schedule**: Every Sunday at 00:00 UTC (automatic)

## Release Assets Generated

For each release, the following files are created:

### Individual Binaries
- `jwb-index-linux-amd64`
- `jwb-index-linux-arm64`
- `jwb-index-windows-amd64.exe`
- `jwb-index-windows-arm64.exe`
- `jwb-index-darwin-amd64`
- `jwb-index-darwin-arm64`
- `jwb-offline-linux-amd64`
- `jwb-offline-linux-arm64`
- `jwb-offline-windows-amd64.exe`
- `jwb-offline-windows-arm64.exe`
- `jwb-offline-darwin-amd64`
- `jwb-offline-darwin-arm64`

### Platform Archives
- `jw-scripts-v1.0.0-linux-amd64.tar.gz`
- `jw-scripts-v1.0.0-linux-arm64.tar.gz`
- `jw-scripts-v1.0.0-windows-amd64.zip`
- `jw-scripts-v1.0.0-windows-arm64.zip`
- `jw-scripts-v1.0.0-darwin-amd64.tar.gz`
- `jw-scripts-v1.0.0-darwin-arm64.tar.gz`

### Verification
- `checksums.txt` - SHA256 checksums for all files

## Using Released Binaries

### Linux/macOS
```bash
# Download the appropriate binary
curl -L -o jwb-index https://github.com/darkace1998/jw-scripts/releases/latest/download/jwb-index-linux-amd64

# Make it executable
chmod +x jwb-index

# Use it
./jwb-index --help
```

### Windows
```cmd
REM Download the .exe file and run it directly
jwb-index-windows-amd64.exe --help
```

### Verification
```bash
# Download checksums
curl -L -o checksums.txt https://github.com/darkace1998/jw-scripts/releases/latest/download/checksums.txt

# Verify a binary (Linux/macOS)
sha256sum -c checksums.txt

# Verify a binary (Windows)
certutil -hashfile jwb-index-windows-amd64.exe SHA256
```