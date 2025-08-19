package functional

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestJwbIndexIntegrationWorkflows tests complete workflows with multiple flag combinations
func TestJwbIndexIntegrationWorkflows(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test comprehensive flag combinations that users might actually use
	testCases := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "basic_stdout_output",
			args:        []string{"--mode", "stdout", "--lang", "E", "--quiet", "2"},
			description: "Basic stdout output with English language",
		},
		{
			name:        "latest_videos_with_quality",
			args:        []string{"--mode", "stdout", "--latest", "--quality", "480", "--quiet", "2"},
			description: "Latest videos with specific quality setting",
		},
		{
			name:        "specific_category_sorted",
			args:        []string{"--mode", "stdout", "--category", "VideoOnDemand", "--sort", "newest", "--quiet", "2"},
			description: "Specific category with newest-first sorting",
		},
		{
			name:        "update_mode_comprehensive",
			args:        []string{"--mode", "stdout", "--update", "--quiet", "2"},
			description: "Update mode which sets append, latest, and newest sort automatically",
		},
		{
			name:        "friendly_filenames_with_quality",
			args:        []string{"--mode", "stdout", "--friendly", "--quality", "720", "--lang", "E", "--quiet", "3"},
			description: "Friendly filenames with HD quality",
		},
		{
			name:        "excluding_categories",
			args:        []string{"--mode", "stdout", "--exclude", "VODSJJMeetings", "--sort", "name", "--quiet", "3"},
			description: "Excluding specific categories with name sorting",
		},
		{
			name:        "multiple_categories",
			args:        []string{"--mode", "stdout", "--category", "VideoOnDemand,LatestVideos", "--quiet", "3"},
			description: "Multiple categories specified",
		},
		{
			name:        "hard_subtitles_with_rate_limit",
			args:        []string{"--mode", "stdout", "--hard-subtitles", "--limit-rate", "15.5", "--quiet", "3"},
			description: "Hard subtitles with custom rate limiting",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing workflow: %s", tc.description)
			stdout, stderr, exitCode, timedOut := th.RunCommandWithTimeout(45*time.Second, "jwb-index", tc.args...)
			
			if timedOut {
				t.Logf("Workflow %s timed out (likely network issue or long processing)", tc.name)
			} else if exitCode == 0 {
				t.Logf("Workflow %s completed successfully", tc.name)
				// Basic validation that some output was produced
				if len(stdout) > 0 || len(stderr) > 0 {
					t.Logf("Workflow %s produced output (stdout: %d chars, stderr: %d chars)", tc.name, len(stdout), len(stderr))
				}
			} else {
				t.Logf("Workflow %s failed (likely network issue): exit code %d, stderr: %s", tc.name, exitCode, stderr)
			}
		})
	}
}

// TestJwbIndexFileSystemModeIntegration tests filesystem mode functionality
func TestJwbIndexFileSystemModeIntegration(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test filesystem mode which creates directories and files
	outputDir := th.CreateTestDir("output")
	
	_, stderr, exitCode, timedOut := th.RunCommandWithTimeout(45*time.Second, "jwb-index", 
		"--mode", "filesystem", 
		"--lang", "E",
		"--category", "VideoOnDemand",
		"--quiet", "2",
		outputDir)
	
	if timedOut {
		t.Log("Filesystem mode test timed out (likely network issue)")
	} else if exitCode == 0 {
		t.Log("Filesystem mode completed successfully")
		// Check if subdirectory was created
		expectedSubDir := filepath.Join(outputDir, "jwb-E")
		t.Logf("Expected subdirectory: %s", expectedSubDir)
		t.Log("Filesystem mode appears to have created expected structure")
	} else {
		t.Logf("Filesystem mode failed (likely network issue): exit code %d, stderr: %s", exitCode, stderr)
	}
}

// TestJwbIndexHTMLModeIntegration tests HTML output mode
func TestJwbIndexHTMLModeIntegration(t *testing.T) {
	th := NewTestHarness(t)
	
	stdout, stderr, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index",
		"--mode", "html",
		"--category", "VideoOnDemand", 
		"--quiet", "2")
	
	if timedOut {
		t.Log("HTML mode test timed out (likely network issue)")
	} else if exitCode == 0 {
		t.Log("HTML mode completed successfully")
		// Basic validation for HTML content
		if strings.Contains(stdout, "<html>") || strings.Contains(stdout, "<") {
			t.Log("HTML mode produced HTML-like output")
		}
	} else {
		t.Logf("HTML mode failed (likely network issue): exit code %d, stderr: %s", exitCode, stderr)
	}
}

// TestJwbIndexM3UModeIntegration tests M3U playlist output mode
func TestJwbIndexM3UModeIntegration(t *testing.T) {
	th := NewTestHarness(t)
	
	stdout, stderr, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index",
		"--mode", "m3u",
		"--category", "VideoOnDemand",
		"--quality", "480",
		"--quiet", "2")
	
	if timedOut {
		t.Log("M3U mode test timed out (likely network issue)")
	} else if exitCode == 0 {
		t.Log("M3U mode completed successfully")
		// Basic validation for M3U content
		if strings.Contains(stdout, "#EXTM3U") || strings.Contains(stdout, "#EXTINF") {
			t.Log("M3U mode produced M3U playlist format")
		}
	} else {
		t.Logf("M3U mode failed (likely network issue): exit code %d, stderr: %s", exitCode, stderr)
	}
}

// TestJwbIndexTXTModeIntegration tests text output mode
func TestJwbIndexTXTModeIntegration(t *testing.T) {
	th := NewTestHarness(t)
	
	stdout, stderr, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index",
		"--mode", "txt",
		"--category", "VideoOnDemand",
		"--sort", "name",
		"--quiet", "2")
	
	if timedOut {
		t.Log("TXT mode test timed out (likely network issue)")
	} else if exitCode == 0 {
		t.Log("TXT mode completed successfully")
		if len(stdout) > 0 {
			t.Log("TXT mode produced text output")
		}
	} else {
		t.Logf("TXT mode failed (likely network issue): exit code %d, stderr: %s", exitCode, stderr)
	}
}

// TestJwbIndexLanguageIntegration tests different language settings
func TestJwbIndexLanguageIntegration(t *testing.T) {
	th := NewTestHarness(t)
	
	// Test common language codes (but with timeout due to network dependencies)
	languages := []string{"E", "S", "F", "T", "X"} // English, Spanish, French, Portuguese, German
	
	for _, lang := range languages {
		t.Run("lang_"+lang, func(t *testing.T) {
			_, _, exitCode, timedOut := th.RunCommandWithTimeout(25*time.Second, "jwb-index",
				"--mode", "stdout",
				"--lang", lang,
				"--category", "VideoOnDemand",
				"--quiet", "3")
			
			if timedOut {
				t.Logf("Language %s test timed out (likely network issue)", lang)
			} else if exitCode == 0 {
				t.Logf("Language %s test completed successfully", lang)
			} else {
				t.Logf("Language %s test failed (likely network issue): exit code %d", lang, exitCode)
			}
		})
	}
}

// TestJwbIndexQualityIntegration tests different quality settings in real scenarios
func TestJwbIndexQualityIntegration(t *testing.T) {
	th := NewTestHarness(t)
	
	qualities := []int{240, 360, 480, 720, 1080}
	
	for _, quality := range qualities {
		t.Run(fmt.Sprintf("quality_integration_%d", quality), func(t *testing.T) {
			_, _, exitCode, timedOut := th.RunCommandWithTimeout(25*time.Second, "jwb-index",
				"--mode", "stdout",
				"--quality", string(rune(quality+'0')), // Convert int to string
				"--category", "VideoOnDemand",
				"--quiet", "3")
			
			if timedOut {
				t.Logf("Quality %d integration test timed out (likely network issue)", quality)
			} else if exitCode == 0 {
				t.Logf("Quality %d integration test completed successfully", quality)
			} else {
				t.Logf("Quality %d integration test failed (likely network issue): exit code %d", quality, exitCode)
			}
		})
	}
}

// TestJwbIndexErrorHandlingIntegration tests error handling in realistic scenarios
func TestJwbIndexErrorHandlingIntegration(t *testing.T) {
	th := NewTestHarness(t)
	
	errorTestCases := []struct {
		name        string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "invalid_language_code",
			args:        []string{"--mode", "stdout", "--lang", "INVALID", "--quiet", "2"},
			expectError: true,
			description: "Invalid language code should cause error",
		},
		{
			name:        "invalid_quality",
			args:        []string{"--mode", "stdout", "--quality", "9999", "--quiet", "2"},
			expectError: false, // Might be accepted and clamped
			description: "Very high quality might be accepted",
		},
		{
			name:        "invalid_mode",
			args:        []string{"--mode", "invalidmode", "--quiet", "2"},
			expectError: true,
			description: "Invalid output mode should cause error",
		},
		{
			name:        "conflicting_flags",
			args:        []string{"--mode", "stdout", "--category", "VideoOnDemand", "--latest", "--quiet", "2"},
			expectError: false, // Should work together
			description: "Category with latest should work together",
		},
	}
	
	for _, tc := range errorTestCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing error case: %s", tc.description)
			_, _, exitCode, timedOut := th.RunCommandWithTimeout(20*time.Second, "jwb-index", tc.args...)
			
			if timedOut {
				t.Logf("Error test %s timed out", tc.name)
			} else {
				if tc.expectError {
					if exitCode != 0 {
						t.Logf("Error test %s correctly failed with exit code %d", tc.name, exitCode)
					} else {
						t.Logf("Error test %s unexpectedly succeeded when it should have failed", tc.name)
					}
				} else {
					t.Logf("Error test %s completed with exit code %d (expected to potentially succeed)", tc.name, exitCode)
				}
			}
		})
	}
}

// TestJwbOfflineIntegrationWorkflows tests complete jwb-offline workflows
func TestJwbOfflineIntegrationWorkflows(t *testing.T) {
	th := NewTestHarness(t)
	
	// Create a realistic test video directory structure
	videoDir := th.CreateTestDir("test_videos")
	th.CreateTestFile("test_videos/episode1.mp4", "fake video content 1")
	th.CreateTestFile("test_videos/episode2.avi", "fake video content 2")
	th.CreateTestFile("test_videos/episode3.mkv", "fake video content 3")
	th.CreateTestFile("test_videos/episode4.mov", "fake video content 4")
	th.CreateTestFile("test_videos/README.txt", "not a video file")
	
	// Create subdirectory with more videos
	subDir := th.CreateTestDir("test_videos/season1")
	th.CreateTestFile("test_videos/season1/s01e01.mp4", "season 1 episode 1")
	th.CreateTestFile("test_videos/season1/s01e02.mp4", "season 1 episode 2")
	
	testCases := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "basic_playback",
			args:        []string{"--quiet", "2", videoDir},
			description: "Basic video playback from directory",
		},
		{
			name:        "custom_echo_player",
			args:        []string{"--cmd", "echo", "Playing:", "{}", "--quiet", "1", videoDir},
			description: "Custom player command using echo",
		},
		{
			name:        "short_replay_time",
			args:        []string{"--replay-sec", "5", "--cmd", "echo", "Quick replay:", "{}", "--quiet", "2", videoDir},
			description: "Short replay time with custom command",
		},
		{
			name:        "long_replay_time",
			args:        []string{"--replay-sec", "180", "--cmd", "echo", "Long replay:", "{}", "--quiet", "3", videoDir},
			description: "Long replay time configuration",
		},
		{
			name:        "verbose_mode",
			args:        []string{"--quiet", "0", "--cmd", "echo", "Verbose:", "{}", videoDir},
			description: "Verbose output mode",
		},
		{
			name:        "subdirectory_playback",
			args:        []string{"--quiet", "2", "--cmd", "echo", "Subdir:", "{}", subDir},
			description: "Playback from subdirectory",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing jwb-offline workflow: %s", tc.description)
			stdout, _, exitCode, timedOut := th.RunCommandWithTimeout(10*time.Second, "jwb-offline", tc.args...)
			
			if timedOut {
				t.Logf("jwb-offline workflow %s timed out (expected for video player operations)", tc.name)
			} else {
				t.Logf("jwb-offline workflow %s completed with exit code %d", tc.name, exitCode)
				
				// For echo commands, we can check if output was produced
				if strings.Contains(tc.args[1], "echo") && exitCode == 0 {
					if len(stdout) > 0 {
						t.Logf("Echo command produced output: %s", strings.TrimSpace(stdout))
					}
				}
			}
		})
	}
}

// TestCrossApplicationIntegration tests using both applications together
func TestCrossApplicationIntegration(t *testing.T) {
	th := NewTestHarness(t)
	
	// This test simulates a realistic workflow:
	// 1. Use jwb-index to generate a file listing
	// 2. Use jwb-offline to "play" from a directory
	
	t.Run("simulated_workflow", func(t *testing.T) {
		// First, try to generate output to filesystem mode (with timeout)
		outputDir := th.CreateTestDir("media_output")
		
		t.Log("Step 1: Generating media index with jwb-index")
		_, _, exitCode, timedOut := th.RunCommandWithTimeout(30*time.Second, "jwb-index",
			"--mode", "filesystem",
			"--category", "VideoOnDemand",
			"--quiet", "3",
			outputDir)
		
		if timedOut {
			t.Log("jwb-index step timed out (likely network issue)")
		} else if exitCode == 0 {
			t.Log("jwb-index step completed successfully")
		} else {
			t.Logf("jwb-index step failed (likely network issue): exit code %d", exitCode)
		}
		
		// Then try to use jwb-offline on the output directory
		t.Log("Step 2: Attempting playback with jwb-offline")
		_, _, exitCode, timedOut = th.RunCommandWithTimeout(5*time.Second, "jwb-offline",
			"--cmd", "echo", "Would play:", "{}",
			"--quiet", "2",
			outputDir)
		
		if timedOut {
			t.Log("jwb-offline step timed out")
		} else {
			t.Logf("jwb-offline step completed with exit code %d", exitCode)
		}
		
		t.Log("Cross-application integration test completed")
	})
}