package functional

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHarness provides utilities for functional testing of CLI applications
type TestHarness struct {
	t       *testing.T
	binDir  string
	workDir string
}

// NewTestHarness creates a new test harness
func NewTestHarness(t *testing.T) *TestHarness {
	// Get the project root directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	
	// Find project root by looking for go.mod
	projectRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find project root with go.mod")
		}
		projectRoot = parent
	}
	
	binDir := filepath.Join(projectRoot, "bin")
	
	// Create a temporary work directory for tests
	workDir, err := os.MkdirTemp("", "jwb-test-*")
	if err != nil {
		t.Fatal(err)
	}
	
	t.Cleanup(func() {
		os.RemoveAll(workDir)
	})
	
	return &TestHarness{
		t:       t,
		binDir:  binDir,
		workDir: workDir,
	}
}

// RunCommand executes a command and returns stdout, stderr, and exit code
func (th *TestHarness) RunCommand(cmd string, args ...string) (string, string, int) {
	binPath := filepath.Join(th.binDir, cmd)
	
	// Check if binary exists
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		th.t.Fatalf("Binary %s does not exist. Make sure to build first: go build -o bin/ ./...", binPath)
	}
	
	command := exec.Command(binPath, args...)
	command.Dir = th.workDir
	
	// Set timeout to prevent hanging
	// ctx := command  // Reserved for future timeout implementation
	
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	
	err := command.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			th.t.Fatalf("Failed to run command %s: %v", cmd, err)
		}
	}
	
	return stdout.String(), stderr.String(), exitCode
}

// RunCommandWithTimeout executes a command with a timeout
func (th *TestHarness) RunCommandWithTimeout(timeout time.Duration, cmd string, args ...string) (string, string, int, bool) {
	binPath := filepath.Join(th.binDir, cmd)
	
	command := exec.Command(binPath, args...)
	command.Dir = th.workDir
	
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	
	done := make(chan error, 1)
	go func() {
		done <- command.Run()
	}()
	
	select {
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			}
		}
		return stdout.String(), stderr.String(), exitCode, false
	case <-time.After(timeout):
		command.Process.Kill()
		return stdout.String(), stderr.String(), -1, true
	}
}

// AssertSuccess asserts that a command was successful (exit code 0)
func (th *TestHarness) AssertSuccess(stdout, stderr string, exitCode int) {
	if exitCode != 0 {
		th.t.Errorf("Command failed with exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
	}
}

// AssertFailure asserts that a command failed (exit code != 0)
func (th *TestHarness) AssertFailure(stdout, stderr string, exitCode int) {
	if exitCode == 0 {
		th.t.Errorf("Command succeeded but was expected to fail\nStdout: %s\nStderr: %s", stdout, stderr)
	}
}

// AssertContains asserts that output contains expected text
func (th *TestHarness) AssertContains(output, expected string) {
	if !strings.Contains(output, expected) {
		th.t.Errorf("Output does not contain expected text\nExpected: %s\nActual: %s", expected, output)
	}
}

// AssertNotContains asserts that output does not contain text
func (th *TestHarness) AssertNotContains(output, notExpected string) {
	if strings.Contains(output, notExpected) {
		th.t.Errorf("Output contains unexpected text\nNot expected: %s\nActual: %s", notExpected, output)
	}
}

// CreateTestFile creates a file in the test work directory
func (th *TestHarness) CreateTestFile(name, content string) string {
	path := filepath.Join(th.workDir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		th.t.Fatal(err)
	}
	return path
}

// CreateTestDir creates a directory in the test work directory
func (th *TestHarness) CreateTestDir(name string) string {
	path := filepath.Join(th.workDir, name)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		th.t.Fatal(err)
	}
	return path
}

// GetWorkDir returns the temporary work directory
func (th *TestHarness) GetWorkDir() string {
	return th.workDir
}