# Known Bugs and Issues

This document tracks identified bugs and potential issues in the jw-scripts codebase.

---

## 1. HTTP Client Missing Timeout Configuration

**Severity:** Medium  
**Impact:** Resource exhaustion, potential DoS  

**Description:** The HTTP clients in `internal/api/client.go` and `internal/books/client.go` are created without timeout configuration. This can lead to goroutine leaks and resource exhaustion if the server is slow or unresponsive.

```go
// Current implementation (no timeout)
httpClient: &http.Client{},
```

**Fix Required:** Add timeout to HTTP client initialization:
```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
},
```

**Locations:**
- `internal/api/client.go` line 32-33 (NewClient function)
- `internal/books/client.go` line 47-49 (NewClient function)

---

## 2. Potential Division by Zero in Category Analysis

**Severity:** Low  
**Impact:** Runtime panic  

**Description:** In `cmd/category-analysis/main.go`, the subtitle ratio calculation divides by `totalMedia` without checking if it's zero first.

```go
fmt.Printf("  Subtitle Ratio: %.2f%%\n", float64(totalWithSubtitles)/float64(totalMedia)*100)
```

**Fix Required:** Add zero check before division.

**Location:** `cmd/category-analysis/main.go` line 74

---

## 3. Potential Panic on Empty Slice Access in Character Analysis

**Severity:** Medium  
**Impact:** Runtime panic  

**Description:** In `cmd/character-analysis/main.go`, slicing operations access the `problematicMedia` and `filenameCollisions` slices without checking if they have enough elements first.

```go
"problematic_samples": problematicMedia[:minInt(len(problematicMedia), 50)],
"collision_samples":   filenameCollisions[:minInt(len(filenameCollisions), 50)],
```

**Issue:** If slices are nil, this could still work, but the logic could be clearer.

**Location:** `cmd/character-analysis/main.go` lines 171-172

---

## 4. VideoManager Infinite Loop with No Exit Mechanism

**Severity:** Medium  
**Impact:** User experience issue  

**Description:** The `Run()` method in `internal/player/player.go` runs an infinite loop with no way to gracefully exit:

```go
for {
    if m.setRandomVideo() {
        ...
    } else {
        ...
        time.Sleep(10 * time.Second)
    }
}
```

**Fix Required:** Add signal handling or context cancellation support for graceful shutdown.

**Location:** `internal/player/player.go` lines 55-77 (Run function)

---

## 5. Integer Overflow Risk in Disk Cleanup

**Severity:** Low  
**Impact:** Incorrect behavior with very large file sizes  

**Description:** In `internal/downloader/downloader.go`, the calculation `needed := referenceMedia.Size + s.KeepFree` could potentially overflow on 32-bit systems when dealing with very large files, though this is mitigated by the overflow check on line 333-336.

```go
needed := referenceMedia.Size + s.KeepFree
if needed < 0 {
    // Overflow or negative values - skip the check
    break
}
```

**Note:** The overflow check is present but the comment could be clearer.

**Location:** `internal/downloader/downloader.go` lines 332-336

---

## 6. Race Condition in Download Progress Tracking (Books)

**Severity:** Low  
**Impact:** Incorrect progress reporting in concurrent scenarios  

**Description:** The `Downloader` struct in `internal/books/downloader.go` has a `progressMutex` but the download operations themselves don't update the progress counters during actual downloads, making the mutex somewhat unused.

```go
type Downloader struct {
    settings        *config.Settings
    progressMutex   sync.Mutex
    downloadedBytes int64
    totalBytes      int64
}
```

The `SetDownloadProgress` and `GetDownloadProgress` methods are thread-safe, but they're never called during actual file downloads.

**Fix Required:** Integrate progress tracking into the download process or remove unused fields.

**Location:** `internal/books/downloader.go` lines 17-24

---

## 7. Missing Error Handling for os.WriteFile in Analysis Tools

**Severity:** Low  
**Impact:** Silent failures in diagnostic tools  

**Description:** In `cmd/api-analysis/main.go`, after creating the temp file, there's an error check for the filename write but the temp file is created with `CreateTemp` and then written with `WriteFile`. If `WriteFile` succeeds but the file was already closed by the defer, this could cause issues.

```go
tmpFile, err := os.CreateTemp("", "api_analysis_*.json")
if err != nil {
    fmt.Printf("Warning: could not create temp file: %v\n", err)
    return
}
defer func() {
    if closeErr := tmpFile.Close(); closeErr != nil {
        fmt.Printf("Warning: failed to close temp file: %v\n", closeErr)
    }
}()

err = os.WriteFile(tmpFile.Name(), jsonData, 0o600)
```

**Note:** The temp file is closed after WriteFile returns, so this should work, but using `tmpFile.Write()` directly would be cleaner.

**Location:** `cmd/api-analysis/main.go` lines 201-214

---

## 8. Duplicate Type Definition

**Severity:** Low  
**Impact:** Code maintainability  

**Description:** In `cmd/api-analysis/main.go`, the `RootCategory` struct is defined twice - once inline inside the main function and once at package level at line 222.

```go
type RootCategory struct {
    Key         string   `json:"key"`
    Type        string   `json:"type"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Tags        []string `json:"tags"`
}
```

**Fix Required:** Remove the duplicate definition.

**Location:** `cmd/api-analysis/main.go` lines 63-69 and lines 222-228

---

## 9. Potential Empty Category Name in Output

**Severity:** Low  
**Impact:** Incorrect output generation  

**Description:** In `internal/output/writer.go`, when generating multi-category output, if a category has an empty name, the generated filename could be invalid or confusing.

```go
s.OutputFilename = fmt.Sprintf("%s.%s", category.Key, getDefaultExtension(s.Mode))
```

If `category.Key` is empty, this would create files like `.m3u` or `.txt`.

**Location:** `internal/output/writer.go` lines 108-114

---

## 10. Missing Context Propagation in HTTP Requests

**Severity:** Medium  
**Impact:** Resource management, cancellation support  

**Description:** HTTP requests in the codebase use `http.Get()` or `httpClient.Get()` without context, making it impossible to cancel long-running requests or implement proper timeout handling at the request level.

```go
resp, err := c.httpClient.Get(url)
```

**Fix Required:** Use `http.NewRequestWithContext()` for all HTTP requests.

**Locations:**
- `internal/api/client.go` multiple locations (GetLanguages, GetRootCategories, GetCategory)
- `internal/books/client.go` line 296 (getPublicationDataForLanguage)

---

## 11. Symlink Error Handling Inconsistency

**Severity:** Low  
**Impact:** Silent failures in filesystem mode  

**Description:** In `internal/output/writer.go`, symlink creation errors are logged as warnings but the function continues. This is intentional (as noted in comments), but the error handling is inconsistent - some errors return immediately while symlink errors are just logged.

```go
if err := os.Symlink(targetPath, linkPath); err != nil {
    // Log symlink error but continue - it's not critical
    fmt.Fprintf(os.Stderr, "Warning: Failed to create symlink %s -> %s: %v\n", linkPath, targetPath, err)
}
```

**Note:** This is working as intended but could confuse users when symlinks fail silently.

**Location:** `internal/output/writer.go` lines 195-198, 213-216, 227-230

---

## 12. TxtWriter Output Filename Validation Missing

**Severity:** Low  
**Impact:** Confusing error messages  

**Description:** In `internal/output/writer.go`, `NewTxtWriter` returns an error if the output filename is empty, but doesn't validate that the filename is a valid path.

```go
if filename == "" {
    return nil, fmt.Errorf("output filename is required for txt mode")
}
```

**Fix Required:** Add path validation or sanitization.

**Location:** `internal/output/writer.go` lines 278-281

---

## 13. Date Parsing Assumes UTC Without Timezone

**Severity:** Low  
**Impact:** Incorrect date handling for non-UTC content  

**Description:** In `internal/api/client.go`, the `parseDate` function strips the timezone indicator and parses without timezone, which could lead to incorrect date comparisons for content from different timezones.

```go
func parseDate(dateString string) (time.Time, error) {
    re := regexp.MustCompile(`\.\d+Z$`)
    dateString = re.ReplaceAllString(dateString, "")
    return time.Parse("2006-01-02T15:04:05", dateString)
}
```

**Note:** The returned time will have no timezone location (zero UTC offset assumed), which may cause issues with MinDate/MaxDate filtering.

**Location:** `internal/api/client.go` lines 302-306

---

## 14. Unchecked Error from Progress Bar

**Severity:** Very Low  
**Impact:** Minor logging noise  

**Description:** In `internal/downloader/downloader.go`, the progress bar `Add64` error is checked but only logs a message. While this is acceptable, the logging message could be confusing to users.

```go
if err := bar.Add64(start); err != nil {
    // Log error but continue - progress bar errors shouldn't stop download
    fmt.Fprintf(os.Stderr, "Progress bar error: %v\n", err)
}
```

**Note:** This is working as intended.

**Location:** `internal/downloader/downloader.go` lines 284-287

---

## 15. Windows Disk Space Function Uses Signed Int64

**Severity:** Low  
**Impact:** Incorrect disk space reporting on Windows for very large drives  

**Description:** In `internal/downloader/disk_free_windows.go`, the `freeBytes` variable is declared as `int64` instead of `uint64`, which could cause issues for drives larger than 8 exabytes (unlikely but technically incorrect).

```go
var freeBytes int64
// ...
return uint64(freeBytes), nil
```

**Fix Required:** Change to `uint64` for consistency with the return type.

**Location:** `internal/downloader/disk_free_windows.go` lines 28, 45

---

## 16. Unsafe Filename Sanitization Edge Cases

**Severity:** Low  
**Impact:** Potential filename issues on some filesystems  

**Description:** The filename sanitization in `internal/api/client.go` handles common problematic characters but may miss some edge cases like:
- Trailing dots and spaces (problematic on Windows)
- Reserved names on Windows (CON, PRN, AUX, NUL, etc.)
- Very long filenames

```go
func formatFilename(s string, safe bool) string {
    // ... current implementation
}
```

**Fix Required:** Add handling for Windows reserved names and trailing dots/spaces.

**Location:** `internal/api/client.go` lines 308-323

---

## 17. Command Writer Command Injection Risk (Design Decision)

**Severity:** Info (by design)  
**Impact:** Potential security concern if used with untrusted input  

**Description:** The `CommandWriter` in `internal/output/writer.go` executes user-provided commands with media URLs as arguments. This is by design but should be documented as a security consideration.

```go
// #nosec G204 - Command is user-configurable via CLI flags for external tool integration
cmd := exec.Command(w.s.Command[0], append(w.s.Command[1:], args...)...)
```

**Note:** The `#nosec` annotation indicates this is intentional, but users should be aware.

**Location:** `internal/output/writer.go` lines 411-415

---

## Summary Statistics

| Severity | Count |
|----------|-------|
| Medium   | 4     |
| Low      | 11    |
| Very Low | 1     |
| Info     | 1     |
| **Total**| **17**|

---

*Last updated: 2026-02-03*
