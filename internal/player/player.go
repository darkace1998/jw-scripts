// Package player provides media playback functionality.
package player

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/darkace1998/jw-scripts/internal/config"
)

// VideoManager manages the video playback.
type VideoManager struct {
	wd        string
	replay    int
	cmd       []string
	verbose   bool
	dumpFile  string
	history   []string
	video     string
	pos       int
	startTime time.Time
	errors    int
}

// NewVideoManager creates a new VideoManager.
func NewVideoManager(s *config.Settings) *VideoManager {
	return &VideoManager{
		wd:       s.WorkDir,
		replay:   30, // default replay time
		cmd:      []string{"omxplayer", "--pos", "{}", "--no-osd"},
		verbose:  s.Quiet < 1,
		dumpFile: filepath.Join(s.WorkDir, "dump.json"),
	}
}

// SetCmd sets the video player command.
func (m *VideoManager) SetCmd(cmd []string) {
	if len(cmd) > 0 {
		m.cmd = cmd
	}
}

// SetReplay sets the replay time in seconds.
func (m *VideoManager) SetReplay(replay int) {
	m.replay = replay
}

// Run starts the video player loop.
func (m *VideoManager) Run() error {
	if err := m.readDump(); err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "could not read dump file: %v\n", err)
		}
	}

	showMsg := true
	for {
		if m.setRandomVideo() {
			if err := m.playVideo(); err != nil {
				return err
			}
			showMsg = true
		} else {
			if showMsg {
				fmt.Fprintln(os.Stderr, "no videos to play yet")
				showMsg = false
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func (m *VideoManager) readDump() error {
	data, err := os.ReadFile(m.dumpFile)
	if err != nil {
		return err
	}

	var dump struct {
		Video   string   `json:"video"`
		Pos     int      `json:"pos"`
		History []string `json:"history"`
	}
	if err := json.Unmarshal(data, &dump); err != nil {
		return err
	}

	m.video = dump.Video
	m.pos = dump.Pos
	m.history = dump.History
	return nil
}

func (m *VideoManager) writeDump() error {
	dump := struct {
		Video   string   `json:"video"`
		Pos     int      `json:"pos"`
		History []string `json:"history"`
	}{
		Video:   m.video,
		Pos:     m.calculatePos(),
		History: m.history,
	}

	data, err := json.Marshal(dump)
	if err != nil {
		return err
	}

	return os.WriteFile(m.dumpFile, data, 0o600)
}

func (m *VideoManager) setRandomVideo() bool {
	if m.video != "" {
		m.startTime = time.Now()
		return true
	}

	files, err := m.listVideos()
	if err != nil || len(files) == 0 {
		return false
	}

	rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })

	for _, vid := range files {
		if !contains(m.history, vid) {
			m.video = vid
			m.pos = 0
			return true
		}
	}
	return false
}

func (m *VideoManager) calculatePos() int {
	if !m.startTime.IsZero() {
		p := int(time.Since(m.startTime).Seconds()) + m.pos - m.replay
		if p < 0 {
			p = 0
		}
		return p
	}
	return 0
}

func (m *VideoManager) playVideo() error {
	if err := m.writeDump(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "playing: %s\n", filepath.Base(m.video))

	cmdArgs := make([]string, len(m.cmd))
	for i, arg := range m.cmd {
		cmdArgs[i] = strings.Replace(arg, "{}", fmt.Sprintf("%d", m.pos), 1)
	}
	cmdArgs = append(cmdArgs, m.video)

	// #nosec G204 - Command is user-configurable via CLI flags for media player functionality
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	if m.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	m.startTime = time.Now()
	if err := cmd.Run(); err != nil {
		// Don't treat player exit as a fatal error
		fmt.Fprintf(os.Stderr, "video player error: %v\n", err)
	}

	if m.calculatePos() == 0 {
		m.errors++
	} else {
		m.errors = 0
	}
	if m.errors > 10 {
		return fmt.Errorf("video player restarting too quickly")
	}

	m.addToHistory(m.video)
	m.video = ""
	return nil
}

func (m *VideoManager) addToHistory(video string) {
	files, _ := m.listVideos()
	maxLen := len(files) / 2
	m.history = append(m.history, video)
	if len(m.history) > maxLen {
		m.history = m.history[len(m.history)-maxLen:]
	}
}

func (m *VideoManager) listVideos() ([]string, error) {
	var videos []string
	files, err := os.ReadDir(m.wd)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".mp4") {
			videos = append(videos, filepath.Join(m.wd, file.Name()))
		}
	}
	return videos, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
