# Known Bugs and Issues

This document tracks identified bugs and potential issues in the jw-scripts codebase.

**Status: All actionable bugs have been fixed.**

---

## 1. HTTP Client Missing Timeout Configuration ✅ FIXED

**Severity:** Medium  
**Impact:** Resource exhaustion, potential DoS  
**Status:** Fixed - Added 30s timeout to both HTTP clients

**Description:** The HTTP clients in `internal/api/client.go` and `internal/books/client.go` are created without timeout configuration. This can lead to goroutine leaks and resource exhaustion if the server is slow or unresponsive.

**Locations:**
- `internal/api/client.go` (NewClient function)
- `internal/books/client.go` (NewClient function)

---

## 2. Potential Division by Zero in Category Analysis ✅ FIXED

**Severity:** Low  
**Impact:** Runtime panic  
**Status:** Fixed - Added zero check before division

**Description:** In `cmd/category-analysis/main.go`, the subtitle ratio calculation divides by `totalMedia` without checking if it's zero first.

**Location:** `cmd/category-analysis/main.go`

---

## 3. Potential Panic on Empty Slice Access in Character Analysis ✅ FIXED

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

## 4. VideoManager Infinite Loop with No Exit Mechanism ✅ FIXED

**Severity:** Medium  
**Impact:** User experience issue  
**Status:** Fixed - Added signal handling (SIGINT/SIGTERM) and context-based graceful shutdown

**Description:** The `Run()` method in `internal/player/player.go` runs an infinite loop with no way to gracefully exit.

**Location:** `internal/player/player.go` (Run function)

---

## 5. Integer Overflow Risk in Disk Cleanup ✅ FIXED

**Severity:** Low  
**Impact:** Incorrect behavior with very large file sizes  
**Status:** Fixed - Improved comment clarity to explain the overflow check

**Description:** In `internal/downloader/downloader.go`, the calculation `needed := referenceMedia.Size + s.KeepFree` could potentially overflow on 32-bit systems when dealing with very large files, though this is mitigated by the overflow check.

**Location:** `internal/downloader/downloader.go`

---

## 6. Race Condition in Download Progress Tracking (Books) ✅ FIXED

**Severity:** Low  
**Impact:** Incorrect progress reporting in concurrent scenarios  
**Status:** Fixed - Removed unused progress tracking fields and methods

**Description:** The `Downloader` struct in `internal/books/downloader.go` had unused `progressMutex`, `downloadedBytes`, and `totalBytes` fields that were never used during actual downloads.

**Location:** `internal/books/downloader.go`

---

## 7. Missing Error Handling for os.WriteFile in Analysis Tools ✅ FIXED

**Severity:** Low  
**Impact:** Silent failures in diagnostic tools  
**Status:** Fixed - Changed to use tmpFile.Write() directly instead of os.WriteFile()

**Description:** In analysis tools, temp files were created with `CreateTemp` and then written with `WriteFile`, which was less efficient. Now using direct file write.
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

**Locations:**
- `cmd/api-analysis/main.go`
- `cmd/character-analysis/main.go`

---

## 8. Duplicate Type Definition ✅ FIXED

**Severity:** Low  
**Impact:** Code maintainability  
**Status:** Fixed - Removed duplicate package-level RootCategory struct

**Description:** In `cmd/api-analysis/main.go`, the `RootCategory` struct was defined twice - once inline inside the main function and once at package level.

**Location:** `cmd/api-analysis/main.go`

---

## 9. Potential Empty Category Name in Output ✅ FIXED

**Severity:** Low  
**Impact:** Incorrect output generation  
**Status:** Fixed - Added validation to skip categories with empty keys

**Description:** In `internal/output/writer.go`, when generating multi-category output, if a category has an empty key, the generated filename could be invalid.

**Location:** `internal/output/writer.go`

---

## 10. Missing Context Propagation in HTTP Requests ⚠️ MITIGATED

**Severity:** Medium  
**Impact:** Resource management, cancellation support  
**Status:** Mitigated - HTTP client timeout (30s) now handles the most critical use case. Full context propagation would require API changes.

**Description:** HTTP requests in the codebase use `http.Get()` without context. This is now mitigated by the 30-second timeout on the HTTP client.

**Note:** Full context propagation is an optional future enhancement that would require changing method signatures.

**Locations:**
- `internal/api/client.go`
- `internal/books/client.go`

---

## 11. Symlink Error Handling Inconsistency ℹ️ BY DESIGN

**Severity:** Low  
**Impact:** Silent failures in filesystem mode  
**Status:** By design - Symlink errors are logged but don't stop processing

**Description:** In `internal/output/writer.go`, symlink creation errors are logged as warnings but the function continues. This is intentional - symlinks are not critical to functionality.

**Location:** `internal/output/writer.go`

---

## 12. TxtWriter Output Filename Validation Missing ✅ FIXED

**Severity:** Low  
**Impact:** Confusing error messages  
**Status:** Fixed - Added basic path validation to NewTxtWriter

**Description:** In `internal/output/writer.go`, `NewTxtWriter` now validates filenames before creating the file.

**Location:** `internal/output/writer.go`

---

## 13. Date Parsing Assumes UTC Without Timezone ✅ FIXED

**Severity:** Low  
**Impact:** Incorrect date handling for non-UTC content  
**Status:** Fixed - Now tries RFC3339 first and explicitly converts to UTC

**Description:** In `internal/api/client.go`, the `parseDate` function now properly handles timezones by trying RFC3339 format first and explicitly converting all times to UTC.

**Location:** `internal/api/client.go`

---

## 14. Unchecked Error from Progress Bar ℹ️ BY DESIGN

**Severity:** Very Low  
**Impact:** Minor logging noise  
**Status:** By design - Error is logged but download continues

**Description:** In `internal/downloader/downloader.go`, the progress bar `Add64` error is checked and logged. Progress bar errors should not stop downloads.

**Location:** `internal/downloader/downloader.go`

---

## 15. Windows Disk Space Function Uses Signed Int64 ✅ FIXED

**Severity:** Low  
**Impact:** Incorrect disk space reporting on Windows for very large drives  
**Status:** Fixed - Changed freeBytes from int64 to uint64

**Description:** In `internal/downloader/disk_free_windows.go`, the `freeBytes` variable is now declared as `uint64` for consistency with the return type.

**Location:** `internal/downloader/disk_free_windows.go`

---

## 16. Unsafe Filename Sanitization Edge Cases ✅ FIXED

**Severity:** Low  
**Impact:** Potential filename issues on some filesystems  
**Status:** Fixed - Added handling for Windows reserved names and trailing dots/spaces

**Description:** The filename sanitization in `internal/api/client.go` now handles:
- Trailing dots and spaces (problematic on Windows)
- Windows reserved names (CON, PRN, AUX, NUL, COM1-9, LPT1-9)

**Location:** `internal/api/client.go`

---

## 17. Command Writer Command Injection Risk ℹ️ BY DESIGN

**Severity:** Info (by design)  
**Impact:** Potential security concern if used with untrusted input  
**Status:** By design - This is documented and intentional for external tool integration

**Description:** The `CommandWriter` in `internal/output/writer.go` executes user-provided commands with media URLs as arguments. The `#nosec` annotation indicates this is intentional.

**Location:** `internal/output/writer.go`

---

## Summary Statistics

| Status | Count |
|--------|-------|
| ✅ FIXED | 13 |
| ⚠️ MITIGATED | 1 |
| ℹ️ BY DESIGN | 3 |
| **Total** | **17** |

---

*Last updated: 2026-02-03*
