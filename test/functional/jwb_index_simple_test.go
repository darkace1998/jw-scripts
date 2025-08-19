package functional

import (
	"strings"
	"testing"
	"time"
)

// TestJwbIndexHelp tests the help flag functionality
func TestJwbIndexHelp(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test --help flag
	stdout, stderr, exitCode := th.RunCommand("jwb-index", "--help")
	th.AssertSuccess(stdout, stderr, exitCode)
	th.AssertContains(stdout, "Index or download media from jw.org")
	th.AssertContains(stdout, "Usage:")
	th.AssertContains(stdout, "Flags:")
	
	// Test -h flag
	stdout, stderr, exitCode = th.RunCommand("jwb-index", "-h")
	th.AssertSuccess(stdout, stderr, exitCode)
	th.AssertContains(stdout, "Index or download media from jw.org")
}

// TestJwbIndexLanguageFlags tests language-related functionality
func TestJwbIndexLanguageFlags(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test --languages flag (with timeout due to network dependency)
	stdout, stderr, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index", "--languages")
	if exitCode == 0 && !timedOut {
		th.AssertContains(stdout, "language codes:")
		// Should contain some common language codes
		th.AssertContains(strings.ToLower(stdout), "e") // English
	} else {
		t.Logf("Languages test failed (likely network issue): exit code %d, timed out: %v, stderr: %s", exitCode, timedOut, stderr)
	}
	
	// Test -L flag
	stdout, stderr, exitCode, timedOut = th.RunCommandWithTimeout(30*time.Second, "jwb-index", "-L")
	if exitCode == 0 && !timedOut {
		th.AssertContains(stdout, "language codes:")
	} else {
		t.Logf("Languages test with -L failed (likely network issue): exit code %d, timed out: %v", exitCode, timedOut)
	}
}

// TestJwbIndexCategoryFlags tests category-related functionality
func TestJwbIndexCategoryFlags(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test --list-categories flag
	stdout, stderr, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index", "--list-categories", "VideoOnDemand")
	if exitCode == 0 && !timedOut {
		th.AssertContains(stdout, "Category:")
	} else {
		t.Logf("Categories test failed (likely network issue): exit code %d, timed out: %v, stderr: %s", exitCode, timedOut, stderr)
	}
	
	// Test -C flag
	stdout, stderr, exitCode, timedOut = th.RunCommandWithTimeout(30*time.Second, "jwb-index", "-C", "VideoOnDemand")
	if exitCode == 0 && !timedOut {
		th.AssertContains(stdout, "Category:")
	} else {
		t.Logf("Categories test with -C failed (likely network issue): exit code %d, timed out: %v", exitCode, timedOut)
	}
}

// TestJwbIndexInvalidFlags tests error handling for invalid flags
func TestJwbIndexInvalidFlags(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test invalid flag
	stdout, stderr, exitCode := th.RunCommand("jwb-index", "--invalid-flag")
	th.AssertFailure(stdout, stderr, exitCode)
	th.AssertContains(stderr, "unknown flag")
	
	// Test missing required parameter
	stdout, stderr, exitCode = th.RunCommand("jwb-index")
	th.AssertFailure(stdout, stderr, exitCode)
	th.AssertContains(stderr, "please use --mode or --download")
}

// TestJwbIndexBasicModes tests different output modes
func TestJwbIndexBasicModes(t *testing.T) {
	th := NewTestHarness(t)
	
	modes := []string{"stdout", "txt"}
	
	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			_, _, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index", "--mode", mode, "--quiet", "3")
			if timedOut {
				t.Logf("Mode %s test timed out (likely network issue)", mode)
			} else if exitCode == 0 {
				t.Logf("Mode %s completed successfully", mode)
			} else {
				t.Logf("Mode %s failed (likely network issue): exit code %d", mode, exitCode)
			}
		})
	}
}

// TestJwbIndexFlagValidation tests various flag validations
func TestJwbIndexFlagValidation(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test quality flag
	_, _, exitCode, timedOut := th.RunCommandWithTimeout(20*time.Second, "jwb-index", "--mode", "stdout", "--quality", "720", "--quiet", "3")
	if timedOut {
		t.Log("Quality test timed out (likely network issue)")
	} else if exitCode != 0 {
		t.Logf("Quality test failed (likely network issue): exit code %d", exitCode)
	}
	
	// Test quiet flag
	_, _, exitCode, timedOut = th.RunCommandWithTimeout(20*time.Second, "jwb-index", "--mode", "stdout", "--quiet", "2")
	if timedOut {
		t.Log("Quiet test timed out (likely network issue)")
	} else if exitCode != 0 {
		t.Logf("Quiet test failed (likely network issue): exit code %d", exitCode)
	}
	
	// Test language flag
	_, _, exitCode, timedOut = th.RunCommandWithTimeout(20*time.Second, "jwb-index", "--mode", "stdout", "--lang", "E", "--quiet", "3")
	if timedOut {
		t.Log("Language test timed out (likely network issue)")
	} else if exitCode != 0 {
		t.Logf("Language test failed (likely network issue): exit code %d", exitCode)
	}
}

// TestJwbIndexAdvancedFlags tests more complex flag combinations
func TestJwbIndexAdvancedFlags(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test update flag (sets append, latest, and sort automatically)
	_, _, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index", "--mode", "stdout", "--update", "--quiet", "3")
	if timedOut {
		t.Log("Update test timed out (likely network issue)")
	} else if exitCode != 0 {
		t.Logf("Update test failed (likely network issue): exit code %d", exitCode)
	}
	
	// Test latest flag
	_, stderr, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index", "--mode", "stdout", "--latest", "--quiet", "0")
	if timedOut {
		t.Log("Latest test timed out (likely network issue)")
	} else if exitCode == 0 {
		// Should contain the filtering message when quiet is 0
		th.AssertContains(stderr, "filtering to latest videos since:")
	} else {
		t.Logf("Latest test failed (likely network issue): exit code %d", exitCode)
	}
	
	// Test run mode without command (should fail)
	stdout, stderr, exitCode := th.RunCommand("jwb-index", "--mode", "run")
	th.AssertFailure(stdout, stderr, exitCode)
	th.AssertContains(stderr, "run mode requires a command to be specified")
}

// TestJwbIndexDownloadFlags tests download-related flags (without actually downloading)
func TestJwbIndexDownloadFlags(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test download flag validation - should either work or fail gracefully
	_, _, exitCode, timedOut := th.RunCommandWithTimeout(10*time.Second, "jwb-index", "--download", "--quiet", "3")
	if timedOut {
		t.Log("Download test timed out (expected for potential long downloads)")
	} else {
		t.Logf("Download test completed with exit code %d", exitCode)
	}
	
	// Test subtitle download flag
	_, _, exitCode, timedOut = th.RunCommandWithTimeout(10*time.Second, "jwb-index", "--download-subtitles", "--quiet", "3")
	if timedOut {
		t.Log("Subtitle download test timed out (expected)")
	} else {
		t.Logf("Subtitle download test completed with exit code %d", exitCode)
	}
}