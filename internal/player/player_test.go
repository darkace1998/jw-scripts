package player

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkace1998/jw-scripts/internal/config"
)

func TestNewVideoManager(t *testing.T) {
	s := &config.Settings{WorkDir: t.TempDir(), Quiet: 2}
	vm := NewVideoManager(s)

	if vm == nil {
		t.Fatal("NewVideoManager returned nil")
	}
	if vm.wd != s.WorkDir {
		t.Errorf("expected wd %q, got %q", s.WorkDir, vm.wd)
	}
	if vm.replay != 30 {
		t.Errorf("expected default replay 30, got %d", vm.replay)
	}
	if vm.cmd[0] != "mpv" {
		t.Errorf("expected default cmd[0] to be mpv, got %q", vm.cmd[0])
	}
}

func TestSetCmd(t *testing.T) {
	s := &config.Settings{WorkDir: t.TempDir(), Quiet: 2}
	vm := NewVideoManager(s)

	vm.SetCmd([]string{"vlc", "--start-time", "{}"})
	if vm.cmd[0] != "vlc" {
		t.Errorf("expected cmd[0] vlc, got %q", vm.cmd[0])
	}

	// Setting empty should not change
	vm.SetCmd([]string{})
	if vm.cmd[0] != "vlc" {
		t.Errorf("empty SetCmd should not change cmd, got %q", vm.cmd[0])
	}
}

func TestSetReplay(t *testing.T) {
	s := &config.Settings{WorkDir: t.TempDir(), Quiet: 2}
	vm := NewVideoManager(s)

	vm.SetReplay(60)
	if vm.replay != 60 {
		t.Errorf("expected replay 60, got %d", vm.replay)
	}
}

func TestListVideos(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	// Create some test files
	for _, name := range []string{"a.mp4", "b.MP4", "c.txt", "d.mp3"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	videos, err := vm.listVideos()
	if err != nil {
		t.Fatalf("listVideos error: %v", err)
	}

	// Should find a.mp4 and b.MP4 (case-insensitive .mp4 match)
	if len(videos) != 2 {
		t.Errorf("expected 2 videos, got %d: %v", len(videos), videos)
	}
}

func TestListVideosEmptyDir(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	videos, err := vm.listVideos()
	if err != nil {
		t.Fatalf("listVideos error: %v", err)
	}
	if len(videos) != 0 {
		t.Errorf("expected 0 videos, got %d", len(videos))
	}
}

func TestListVideosBadDir(t *testing.T) {
	s := &config.Settings{WorkDir: "/nonexistent/path", Quiet: 2}
	vm := NewVideoManager(s)

	_, err := vm.listVideos()
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestDumpRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	vm.video = filepath.Join(dir, "test.mp4")
	vm.history = []string{filepath.Join(dir, "old.mp4")}

	if err := vm.writeDump(); err != nil {
		t.Fatalf("writeDump error: %v", err)
	}

	// Read back and verify
	vm2 := NewVideoManager(s)
	if err := vm2.readDump(); err != nil {
		t.Fatalf("readDump error: %v", err)
	}

	if vm2.video != vm.video {
		t.Errorf("video mismatch: got %q, want %q", vm2.video, vm.video)
	}
	if len(vm2.history) != 1 || vm2.history[0] != vm.history[0] {
		t.Errorf("history mismatch: got %v, want %v", vm2.history, vm.history)
	}
}

func TestReadDumpNotFound(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	err := vm.readDump()
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist error, got %v", err)
	}
}

func TestReadDumpInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	if err := os.WriteFile(vm.dumpFile, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := vm.readDump()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWriteDumpFormat(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)
	vm.video = "test.mp4"

	if err := vm.writeDump(); err != nil {
		t.Fatalf("writeDump error: %v", err)
	}

	data, err := os.ReadFile(vm.dumpFile)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("dump file is not valid JSON: %v", err)
	}
	if _, ok := parsed["video"]; !ok {
		t.Error("dump file missing 'video' key")
	}
}

func TestSetRandomVideoEmpty(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	if vm.setRandomVideo() {
		t.Error("setRandomVideo should return false for empty dir")
	}
}

func TestSetRandomVideoPicksFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("v"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	if !vm.setRandomVideo() {
		t.Error("setRandomVideo should return true when videos exist")
	}
	if vm.video == "" {
		t.Error("video should be set after setRandomVideo")
	}
}

func TestSetRandomVideoSkipsHistory(t *testing.T) {
	dir := t.TempDir()
	vPath := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(vPath, []byte("v"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)
	vm.history = []string{vPath}

	if vm.setRandomVideo() {
		t.Error("setRandomVideo should return false when all videos are in history")
	}
}

func TestSetRandomVideoResumesExisting(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)
	vm.video = "existing.mp4"

	if !vm.setRandomVideo() {
		t.Error("setRandomVideo should return true when video already set")
	}
}

func TestCalculatePos(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	// Zero start time
	if vm.calculatePos() != 0 {
		t.Error("expected 0 for zero start time")
	}
}

func TestAddToHistory(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 10; i++ {
		name := filepath.Join(dir, "v"+string(rune('0'+i))+".mp4")
		if err := os.WriteFile(name, []byte("v"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	// Add videos to history
	for i := 0; i < 10; i++ {
		vm.addToHistory("video" + string(rune('0'+i)))
	}

	// History should be trimmed to at most len(files)/2 = 5
	if len(vm.history) > 5 {
		t.Errorf("expected history <= 5, got %d", len(vm.history))
	}
}

func TestStop(t *testing.T) {
	dir := t.TempDir()
	s := &config.Settings{WorkDir: dir, Quiet: 2}
	vm := NewVideoManager(s)

	vm.Stop()

	select {
	case <-vm.ctx.Done():
		// expected
	default:
		t.Error("expected context to be canceled after Stop")
	}
}
