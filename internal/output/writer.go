// Package output provides various output writers for media content.
package output

import (
	"fmt"
	"html"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
)

// PlaylistEntry represents a single entry in a playlist.
type PlaylistEntry struct {
	Name     string
	Source   string
	Duration int
}

// Writer is the interface for all output writers.
type Writer interface {
	Add(entry PlaylistEntry)
	Dump() error
}

// CreateOutput creates the output based on the settings.
func CreateOutput(s *config.Settings, data []*api.Category) error {
	if s.Mode == "filesystem" {
		return outputFilesystem(s, data)
	}

	var writer Writer
	var err error

	switch {
	case strings.HasPrefix(s.Mode, "txt"):
		writer, err = NewTxtWriter(s)
	case strings.HasPrefix(s.Mode, "m3u"):
		writer, err = NewM3uWriter(s)
	case strings.HasPrefix(s.Mode, "html"):
		writer, err = NewHTMLWriter(s)
	case s.Mode == "stdout":
		writer = NewStdoutWriter(s)
	case s.Mode == "run":
		writer = NewCommandWriter(s)
	default:
		return fmt.Errorf("unknown mode: %s", s.Mode)
	}

	if err != nil {
		return err
	}

	if strings.HasSuffix(s.Mode, "multi") || strings.HasSuffix(s.Mode, "tree") {
		return outputMulti(s, data, writer)
	}
	return outputSingle(s, data, writer)
}

func outputSingle(s *config.Settings, data []*api.Category, writer Writer) error {
	var allMedia []*api.Media
	for _, category := range data {
		for _, item := range category.Contents {
			if media, ok := item.(*api.Media); ok {
				allMedia = append(allMedia, media)
			}
		}
	}
	sortMedia(allMedia, s.Sort)

	for _, media := range allMedia {
		source := media.URL
		if fileExists(filepath.Join(s.WorkDir, s.SubDir, media.Filename)) {
			source = filepath.Join(".", s.SubDir, media.Filename)
		}
		writer.Add(PlaylistEntry{
			Name:     media.Name,
			Source:   source,
			Duration: int(math.Round(media.Duration)),
		})
	}

	return writer.Dump()
}

func outputMulti(s *config.Settings, data []*api.Category, writer Writer) error {
	for _, category := range data {
		var categoryMedia []*api.Media
		for _, item := range category.Contents {
			if media, ok := item.(*api.Media); ok {
				categoryMedia = append(categoryMedia, media)
			}
		}

		if len(categoryMedia) == 0 {
			continue
		}

		sortMedia(categoryMedia, s.Sort)

		// Create separate output file for each category
		originalFilename := s.OutputFilename
		if originalFilename == "" {
			s.OutputFilename = fmt.Sprintf("%s.%s", category.Key, getDefaultExtension(s.Mode))
		} else {
			ext := filepath.Ext(originalFilename)
			base := strings.TrimSuffix(originalFilename, ext)
			s.OutputFilename = fmt.Sprintf("%s_%s%s", base, category.Key, ext)
		}

		// Create new writer for this category
		var categoryWriter Writer
		var err error

		switch {
		case strings.HasPrefix(s.Mode, "txt"):
			categoryWriter, err = NewTxtWriter(s)
		case strings.HasPrefix(s.Mode, "m3u"):
			categoryWriter, err = NewM3uWriter(s)
		case strings.HasPrefix(s.Mode, "html"):
			categoryWriter, err = NewHTMLWriter(s)
		default:
			// For stdout and run modes, we can't really do "multi" so just use single output
			s.OutputFilename = originalFilename
			return outputSingle(s, data, writer)
		}

		if err != nil {
			s.OutputFilename = originalFilename
			return err
		}

		for _, media := range categoryMedia {
			source := media.URL
			if fileExists(filepath.Join(s.WorkDir, s.SubDir, media.Filename)) {
				source = filepath.Join(".", s.SubDir, media.Filename)
			}
			categoryWriter.Add(PlaylistEntry{
				Name:     media.Name,
				Source:   source,
				Duration: int(math.Round(media.Duration)),
			})
		}

		if err := categoryWriter.Dump(); err != nil {
			s.OutputFilename = originalFilename
			return err
		}

		// Restore original filename
		s.OutputFilename = originalFilename
	}

	return nil
}

func getDefaultExtension(mode string) string {
	switch {
	case strings.HasPrefix(mode, "txt"):
		return "txt"
	case strings.HasPrefix(mode, "m3u"):
		return "m3u"
	case strings.HasPrefix(mode, "html"):
		return "html"
	default:
		return "txt"
	}
}

func outputFilesystem(s *config.Settings, data []*api.Category) error {
	dataDir := filepath.Join(s.WorkDir, s.SubDir)
	if s.Quiet < 1 {
		fmt.Fprintln(os.Stderr, "creating directory structure")
	}

	for _, category := range data {
		catDir := filepath.Join(dataDir, category.Key)
		if err := os.MkdirAll(catDir, 0o750); err != nil {
			return err
		}

		if category.Home {
			// Create symlink for home categories
			linkPath := filepath.Join(s.WorkDir, category.Name)
			targetPath, err := filepath.Rel(s.WorkDir, catDir)
			if err != nil {
				return err
			}
			if err := os.Symlink(targetPath, linkPath); err != nil {
				// Log symlink error but continue - it's not critical
				fmt.Fprintf(os.Stderr, "Warning: Failed to create symlink %s -> %s: %v\n", linkPath, targetPath, err)
			}
		}

		for _, item := range category.Contents {
			switch v := item.(type) {
			case *api.Category:
				linkDest := filepath.Join(dataDir, v.Key)
				if err := os.MkdirAll(linkDest, 0o750); err != nil {
					return err
				}
				linkFile := filepath.Join(catDir, v.Name)
				targetPath, err := filepath.Rel(catDir, linkDest)
				if err != nil {
					return err
				}
				if err := os.Symlink(targetPath, linkFile); err != nil {
					// Log symlink error but continue - it's not critical
					fmt.Fprintf(os.Stderr, "Warning: Failed to create symlink %s -> %s: %v\n", linkFile, targetPath, err)
				}
			case *api.Media:
				linkDest := filepath.Join(dataDir, v.Filename)
				if !fileExists(linkDest) {
					continue
				}
				linkFile := filepath.Join(catDir, v.FriendlyName)
				targetPath, err := filepath.Rel(catDir, linkDest)
				if err != nil {
					return err
				}
				if err := os.Symlink(targetPath, linkFile); err != nil {
					// Log symlink error but continue - it's not critical
					fmt.Fprintf(os.Stderr, "Warning: Failed to create symlink %s -> %s: %v\n", linkFile, targetPath, err)
				}
			}
		}
	}
	return nil
}

func sortMedia(mediaList []*api.Media, sortKey string) {
	switch sortKey {
	case "name":
		sort.Slice(mediaList, func(i, j int) bool {
			return mediaList[i].Name < mediaList[j].Name
		})
	case "newest", "oldest":
		sort.Slice(mediaList, func(i, j int) bool {
			if sortKey == "newest" {
				return mediaList[i].Date > mediaList[j].Date
			}
			return mediaList[i].Date < mediaList[j].Date
		})
	case "random":
		// Use the global random number generator (automatically seeded in Go 1.20+)
		rand.Shuffle(len(mediaList), func(i, j int) {
			mediaList[i], mediaList[j] = mediaList[j], mediaList[i]
		})
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// --- TxtWriter ---

// TxtWriter handles writing playlist entries to a text file
type TxtWriter struct {
	s         *config.Settings
	file      *os.File
	queue     []PlaylistEntry
	history   map[string]bool
	start     string
	end       string
	formatter func(PlaylistEntry) string
}

// NewTxtWriter creates a new TxtWriter instance for writing playlist entries to a text file
func NewTxtWriter(s *config.Settings) (*TxtWriter, error) {
	filename := s.OutputFilename
	if filename == "" {
		return nil, fmt.Errorf("output filename is required for txt mode")
	}
	// #nosec G304 - Path is user-configured output file for legitimate file operations
	file, err := os.Create(filepath.Join(s.WorkDir, filename))
	if err != nil {
		return nil, err
	}

	return &TxtWriter{
		s:       s,
		file:    file,
		history: make(map[string]bool),
		formatter: func(e PlaylistEntry) string {
			return e.Source
		},
	}, nil
}

// Add adds a playlist entry to the writer's queue
func (w *TxtWriter) Add(entry PlaylistEntry) {
	if !w.history[entry.Source] {
		w.queue = append(w.queue, entry)
		w.history[entry.Source] = true
	}
}

// Dump writes all queued playlist entries to the output file
func (w *TxtWriter) Dump() error {
	defer func() { _ = w.file.Close() }()
	if _, err := w.file.WriteString(w.start); err != nil {
		return err
	}

	for _, entry := range w.queue {
		if _, err := w.file.WriteString(w.formatter(entry) + "\n"); err != nil {
			return err
		}
	}

	if _, err := w.file.WriteString(w.end); err != nil {
		return err
	}
	return nil
}

// --- M3uWriter ---

// NewM3uWriter creates a new TxtWriter configured for M3U playlist format
func NewM3uWriter(s *config.Settings) (*TxtWriter, error) {
	w, err := NewTxtWriter(s)
	if err != nil {
		return nil, err
	}
	w.start = "#EXTM3U\n"
	w.formatter = func(e PlaylistEntry) string {
		return fmt.Sprintf("#EXTINF:%d, %s\n%s", e.Duration, e.Name, e.Source)
	}
	return w, nil
}

// --- HtmlWriter ---

// NewHTMLWriter creates a new TxtWriter configured for HTML format
func NewHTMLWriter(s *config.Settings) (*TxtWriter, error) {
	w, err := NewTxtWriter(s)
	if err != nil {
		return nil, err
	}
	w.start = "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"/></head><body>\n"
	w.end = "</body></html>\n"
	w.formatter = func(e PlaylistEntry) string {
		return fmt.Sprintf("<a href=%q>%s</a><br>", html.EscapeString(e.Source), html.EscapeString(e.Name))
	}
	return w, nil
}

// --- StdoutWriter ---

// StdoutWriter handles writing playlist entries to standard output
type StdoutWriter struct {
	s     *config.Settings
	queue []PlaylistEntry
}

// NewStdoutWriter creates a new StdoutWriter instance for writing playlist entries to stdout
func NewStdoutWriter(s *config.Settings) *StdoutWriter {
	return &StdoutWriter{s: s}
}

// Add adds a playlist entry to the writer's queue
func (w *StdoutWriter) Add(entry PlaylistEntry) {
	w.queue = append(w.queue, entry)
}

// Dump writes all queued playlist entries to standard output
func (w *StdoutWriter) Dump() error {
	for _, entry := range w.queue {
		fmt.Println(entry.Source)
	}
	return nil
}

// --- CommandWriter ---

// CommandWriter handles executing commands for playlist entries
type CommandWriter struct {
	s     *config.Settings
	queue []PlaylistEntry
}

// NewCommandWriter creates a new CommandWriter instance for executing commands on playlist entries
func NewCommandWriter(s *config.Settings) *CommandWriter {
	return &CommandWriter{s: s}
}

// Add adds a playlist entry to the writer's queue
func (w *CommandWriter) Add(entry PlaylistEntry) {
	w.queue = append(w.queue, entry)
}

// Dump executes the configured command for all queued playlist entries
func (w *CommandWriter) Dump() error {
	if len(w.queue) == 0 {
		return nil
	}

	var args []string
	for _, entry := range w.queue {
		args = append(args, entry.Source)
	}

	// #nosec G204 - Command is user-configurable via CLI flags for external tool integration
	cmd := exec.Command(w.s.Command[0], append(w.s.Command[1:], args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
