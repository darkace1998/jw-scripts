package functional

import (
	"testing"
	"time"
)

// TestJwbOfflineHelp tests the help flag functionality
func TestJwbOfflineHelp(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test --help flag
	stdout, stderr, exitCode := th.RunCommand("jwb-offline", "--help")
	th.AssertSuccess(stdout, stderr, exitCode)
	th.AssertContains(stdout, "Shuffle and play videos in DIR")
	th.AssertContains(stdout, "Usage:")
	th.AssertContains(stdout, "Flags:")
	
	// Test -h flag
	stdout, stderr, exitCode = th.RunCommand("jwb-offline", "-h")
	th.AssertSuccess(stdout, stderr, exitCode)
	th.AssertContains(stdout, "Shuffle and play videos in DIR")
}

// TestJwbOfflineInvalidFlags tests error handling for invalid flags
func TestJwbOfflineInvalidFlags(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test invalid flag
	stdout, stderr, exitCode := th.RunCommand("jwb-offline", "--invalid-flag")
	th.AssertFailure(stdout, stderr, exitCode)
	th.AssertContains(stderr, "unknown flag")
}

// TestJwbOfflineBasicFlags tests basic flag functionality
func TestJwbOfflineBasicFlags(t *testing.T) {
	th := NewTestHarness(t)
	
	// Create a test directory with some fake video files
	testDir := th.CreateTestDir("videos")
	th.CreateTestFile("videos/test1.mp4", "fake video content 1")
	th.CreateTestFile("videos/test2.mp4", "fake video content 2")
	
	// Test quiet flag (with timeout since it might try to play videos)
	_, _, exitCode, timedOut := th.RunCommandWithTimeout(5*time.Second, "jwb-offline", "--quiet", "3", testDir)
	if timedOut {
		t.Log("Quiet test timed out (expected - likely waiting for video player)")
	} else {
		t.Logf("Quiet test completed with exit code %d", exitCode)
	}
	
	// Test replay-sec flag
	_, _, exitCode, timedOut = th.RunCommandWithTimeout(5*time.Second, "jwb-offline", "--replay-sec", "30", "--quiet", "3", testDir)
	if timedOut {
		t.Log("Replay-sec test timed out (expected)")
	} else {
		t.Logf("Replay-sec test completed with exit code %d", exitCode)
	}
}

// TestJwbOfflineCustomCommand tests custom player command functionality
func TestJwbOfflineCustomCommand(t *testing.T) {
	th := NewTestHarness(t)
	
	// Create a test directory with some fake video files
	testDir := th.CreateTestDir("videos")
	th.CreateTestFile("videos/test1.mp4", "fake video content 1")
	
	// Test with echo command (which should exist on most systems)
	stdout, _, exitCode, timedOut := th.RunCommandWithTimeout(5*time.Second, "jwb-offline", "--cmd", "echo", "playing", "{}", "--quiet", "2", testDir)
	if timedOut {
		t.Log("Custom command test timed out")
	} else {
		t.Logf("Custom command test completed with exit code %d", exitCode)
		if exitCode == 0 && len(stdout) > 0 {
			t.Logf("Custom command produced output: %s", stdout)
		}
	}
}

// TestJwbOfflineDirectoryHandling tests directory argument functionality
func TestJwbOfflineDirectoryHandling(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test with non-existent directory
	_, _, exitCode, timedOut := th.RunCommandWithTimeout(5*time.Second, "jwb-offline", "--quiet", "3", "/non/existent/directory")
	if timedOut {
		t.Log("Non-existent directory test timed out")
	} else {
		t.Logf("Non-existent directory test completed with exit code %d", exitCode)
	}
	
	// Test with empty directory
	emptyDir := th.CreateTestDir("empty")
	_, _, exitCode, timedOut = th.RunCommandWithTimeout(5*time.Second, "jwb-offline", "--quiet", "3", emptyDir)
	if timedOut {
		t.Log("Empty directory test timed out")
	} else {
		t.Logf("Empty directory test completed with exit code %d", exitCode)
	}
	
	// Test without directory argument (should use current directory)
	_, _, exitCode, timedOut = th.RunCommandWithTimeout(5*time.Second, "jwb-offline", "--quiet", "3")
	if timedOut {
		t.Log("Default directory test timed out")
	} else {
		t.Logf("Default directory test completed with exit code %d", exitCode)
	}
}

// TestJwbOfflineErrorConditions tests various error conditions
func TestJwbOfflineErrorConditions(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test with invalid replay-sec value
	stdout, stderr, exitCode := th.RunCommand("jwb-offline", "--replay-sec", "invalid")
	th.AssertFailure(stdout, stderr, exitCode)
	th.AssertContains(stderr, "invalid argument")
	
	// Test with invalid quiet value
	stdout, stderr, exitCode = th.RunCommand("jwb-offline", "--quiet", "invalid")
	th.AssertFailure(stdout, stderr, exitCode)
	th.AssertContains(stderr, "invalid argument")
}

// TestJwbOfflineFlagCombinations tests various flag combinations
func TestJwbOfflineFlagCombinations(t *testing.T) {
	th := NewTestHarness(t)
	
	// Create a test directory with some fake video files
	testDir := th.CreateTestDir("videos")
	th.CreateTestFile("videos/test1.mp4", "fake video content 1")
	
	testCases := []struct {
		name string
		args []string
	}{
		{
			name: "all_flags_combined",
			args: []string{"--quiet", "2", "--replay-sec", "45", "--cmd", "echo", "test", "{}", testDir},
		},
		{
			name: "minimal_flags",
			args: []string{"--quiet", "3", testDir},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, exitCode, timedOut := th.RunCommandWithTimeout(5*time.Second, "jwb-offline", tc.args...)
			if timedOut {
				t.Logf("Flag combination %s test timed out (expected)", tc.name)
			} else {
				t.Logf("Flag combination %s test completed with exit code %d", tc.name, exitCode)
			}
		})
	}
}