# JW Scripts Functional Test Suite

This directory contains comprehensive functional tests for the JW Scripts applications (`jwb-index` and `jwb-offline`). The test suite validates all command-line flags, error handling, and integration workflows.

## Overview

The functional test suite is designed to:
- Test all command-line flags and their functionality
- Validate error handling and edge cases
- Test realistic usage scenarios and workflows
- Ensure cross-application compatibility
- Provide comprehensive coverage of the CLI interface

## Test Structure

### Core Test Files

- `functional_test.go` - Test harness and utility functions
- `jwb_index_test.go` - Comprehensive tests for the jwb-index application
- `jwb_offline_test.go` - Comprehensive tests for the jwb-offline application
- `integration_test.go` - Cross-application and complex workflow tests
- `run_tests.sh` - Test runner script with detailed reporting

### Test Categories

#### jwb-index Application Tests

**Basic Functionality:**
- Help output (`--help`, `-h`)
- Invalid flag handling
- Missing required parameters

**Language Features:**
- Language listing (`--languages`, `-L`)
- Language selection (`--lang`, `-l`)
- Language code validation

**Category Management:**
- Category listing (`--list-categories`, `-C`)
- Category filtering (`--category`, `-c`)
- Category exclusion (`--exclude`)
- Latest videos (`--latest`)

**Output Modes:**
- stdout mode
- txt mode
- html mode
- m3u mode
- filesystem mode
- run mode with custom commands

**Quality and Media Settings:**
- Quality selection (`--quality`, `-Q`)
- Rate limiting (`--limit-rate`, `-R`)
- Friendly filenames (`--friendly`, `-H`)
- Hard subtitles (`--hard-subtitles`)
- Checksum validation (`--checksum`)

**File Management:**
- Download flags (`--download`, `-d`)
- Subtitle downloads (`--download-subtitles`)
- Append mode (`--append`)
- Free space management (`--free`)
- Broken file fixing (`--fix-broken`)

**Sorting and Filtering:**
- Sort options (`--sort`: newest, oldest, name, random)
- Date filtering (`--since`)
- Update mode (`--update`)

**Verbosity and Output Control:**
- Quiet levels (`--quiet`, `-q`)
- Warning control (`--no-warning`)

#### jwb-offline Application Tests

**Basic Functionality:**
- Help output (`--help`, `-h`)
- Invalid flag handling

**Player Configuration:**
- Custom player commands (`--cmd`)
- Replay timing (`--replay-sec`)
- Verbosity control (`--quiet`, `-q`)

**Directory Handling:**
- Directory argument processing
- Empty directory handling
- Non-existent directory handling
- Video file discovery

**Error Conditions:**
- Invalid parameter values
- Multiple directory arguments
- Command validation

#### Integration Tests

**Cross-Application Workflows:**
- jwb-index → jwb-offline pipeline
- File system mode integration
- Directory structure validation

**Complex Flag Combinations:**
- Multiple flags working together
- Real-world usage patterns
- Error condition testing

**Output Mode Testing:**
- HTML output validation
- M3U playlist generation
- Text output formatting
- Filesystem structure creation

## Running the Tests

### Quick Start

```bash
# Run all functional tests
./test/functional/run_tests.sh
```

### Manual Execution

```bash
# Build applications first
go build -o bin/ ./...

# Run specific test files
go test -v ./test/functional/... -run TestJwbIndexHelp
go test -v ./test/functional/... -run TestJwbOffline
go test -v ./test/functional/... -run TestIntegration

# Run with custom timeout
TEST_TIMEOUT=600 ./test/functional/run_tests.sh
```

### Test Environment

The tests automatically:
- Build the applications if needed
- Create temporary directories for test files
- Handle cleanup after test completion
- Manage timeouts for network-dependent operations

## Network Dependencies

Many tests interact with external services (jw.org APIs) and may:
- Timeout due to network latency
- Fail due to connectivity issues
- Return different results based on content availability

This is expected behavior. The tests are designed to be resilient and report network-related failures as informational rather than critical errors.

## Test Coverage

### jwb-index Flags Tested

| Flag | Short | Purpose | Tested |
|------|-------|---------|--------|
| `--append` | | Append to file instead of overwriting | ✅ |
| `--category` | `-c` | Categories to index | ✅ |
| `--checksum` | | Validate MD5 checksums | ✅ |
| `--clean-symlinks` | | Remove old symlinks | ✅ |
| `--command` | | Command for run mode | ✅ |
| `--download` | `-d` | Download media files | ✅ |
| `--download-subtitles` | | Download VTT subtitles | ✅ |
| `--exclude` | | Categories to skip | ✅ |
| `--fix-broken` | | Re-download broken files | ✅ |
| `--free` | | Disk space to keep free | ✅ |
| `--friendly` | `-H` | Human readable names | ✅ |
| `--hard-subtitles` | | Prefer hard-coded subtitles | ✅ |
| `--help` | `-h` | Show help | ✅ |
| `--import` | | Import from directory | ✅ |
| `--lang` | `-l` | Language code | ✅ |
| `--languages` | `-L` | List language codes | ✅ |
| `--latest` | | Latest videos only | ✅ |
| `--limit-rate` | `-R` | Download rate limit | ✅ |
| `--list-categories` | `-C` | List categories | ✅ |
| `--mode` | `-m` | Output mode | ✅ |
| `--no-warning` | | Disable warnings | ✅ |
| `--quality` | `-Q` | Video quality | ✅ |
| `--quiet` | `-q` | Reduce verbosity | ✅ |
| `--since` | | Date filtering | ✅ |
| `--sort` | | Sort output | ✅ |
| `--update` | | Update mode | ✅ |

### jwb-offline Flags Tested

| Flag | Short | Purpose | Tested |
|------|-------|---------|--------|
| `--cmd` | | Video player command | ✅ |
| `--help` | `-h` | Show help | ✅ |
| `--quiet` | `-q` | Reduce verbosity | ✅ |
| `--replay-sec` | | Replay timing | ✅ |

### Output Modes Tested

| Mode | Purpose | Tested |
|------|---------|--------|
| `stdout` | Print to standard output | ✅ |
| `txt` | Generate text file | ✅ |
| `html` | Generate HTML page | ✅ |
| `m3u` | Generate M3U playlist | ✅ |
| `filesystem` | Create directory structure | ✅ |
| `run` | Execute custom command | ✅ |

## Error Handling Coverage

The tests validate proper error handling for:
- Invalid flag values
- Missing required parameters
- Network connectivity issues
- File system permissions
- Invalid configuration combinations
- Malformed input data

## Expected Behavior

### Successful Tests
Tests that complete successfully indicate:
- Flags are parsed correctly
- Application logic executes without crashes
- Output is generated in expected format
- Error conditions are handled gracefully

### Timeout/Network-Related Failures
Some tests may timeout or fail due to:
- Slow network connections
- jw.org service availability
- Rate limiting by external services
- Geographic content restrictions

These are not considered test failures but rather environmental limitations.

### Configuration Requirements
The tests assume:
- Go development environment is available
- Applications can be built successfully
- Temporary file creation is permitted
- Network access is available (for some tests)

## Extending the Tests

To add new test cases:

1. **Add flag tests**: Add new test functions in the appropriate test file
2. **Add integration tests**: Create workflow tests in `integration_test.go`
3. **Update documentation**: Add new flags to the coverage table above
4. **Test error conditions**: Include both positive and negative test cases

### Example Test Function

```go
func TestNewFlag(t *testing.T) {
    th := NewTestHarness(t)
    
    stdout, stderr, exitCode := th.RunCommand("jwb-index", "--new-flag", "value")
    th.AssertSuccess(stdout, stderr, exitCode)
    th.AssertContains(stdout, "expected output")
}
```

## Troubleshooting

### Common Issues

**Build Failures:**
- Ensure Go is installed and `go build` works
- Check that all dependencies are available

**Test Timeouts:**
- Increase `TEST_TIMEOUT` environment variable
- Check network connectivity
- Verify jw.org services are accessible

**Permission Errors:**
- Ensure write access to temporary directories
- Check that test runner script is executable

**Missing Output:**
- Some tests may produce no output when running with high quiet levels
- This is expected behavior, not a failure

### Debug Mode

For verbose test output:
```bash
go test -v -run TestSpecificFunction ./test/functional/...
```

For extended timeouts:
```bash
TEST_TIMEOUT=900 ./test/functional/run_tests.sh
```