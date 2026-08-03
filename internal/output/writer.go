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
	"regexp"
	"sort"
	"strings"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
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
	if s.OutputFilename == "" && requiresOutputFilename(s.Mode) {
		s.OutputFilename = fmt.Sprintf("playlist.%s", getDefaultExtension(s.Mode))
	}

	if strings.HasSuffix(s.Mode, "multi") || strings.HasSuffix(s.Mode, "tree") {
		return outputMulti(s, data)
	}

	writer, err := newWriter(s)
	if err != nil {
		return err
	}
	return outputSingle(s, data, writer)
}

// newWriter constructs the writer for the configured mode. When --append is
// requested, existing file content is loaded so old entries are preserved and
// deduplicated against new ones.
func newWriter(s *config.Settings) (Writer, error) {
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
		return nil, fmt.Errorf("unknown mode: %s", s.Mode)
	}
	if err != nil {
		return nil, err
	}

	if s.Append {
		if a, ok := writer.(interface{ LoadExisting() error }); ok {
			if err := a.LoadExisting(); err != nil {
				return nil, err
			}
		}
	}

	return writer, nil
}

func requiresOutputFilename(mode string) bool {
	return strings.HasPrefix(mode, "txt") ||
		strings.HasPrefix(mode, "m3u") ||
		strings.HasPrefix(mode, "html")
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
		if media.Filename != "" && fileExists(filepath.Join(s.WorkDir, s.SubDir, media.Filename)) {
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

func outputMulti(s *config.Settings, data []*api.Category) error {
	originalFilename := s.OutputFilename
	defer func() { s.OutputFilename = originalFilename }()

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

		// Skip categories with empty keys to avoid invalid filenames
		if category.Key == "" {
			continue
		}

		sortMedia(categoryMedia, s.Sort)

		// Create separate output file for each category
		if originalFilename == "" {
			s.OutputFilename = fmt.Sprintf("%s.%s", category.Key, getDefaultExtension(s.Mode))
		} else {
			ext := filepath.Ext(originalFilename)
			base := strings.TrimSuffix(originalFilename, ext)
			s.OutputFilename = fmt.Sprintf("%s_%s%s", base, category.Key, ext)
		}

		// Create new writer for this category
		categoryWriter, err := newWriter(s)
		if err != nil {
			return err
		}

		for _, media := range categoryMedia {
			source := media.URL
			if media.Filename != "" && fileExists(filepath.Join(s.WorkDir, s.SubDir, media.Filename)) {
				source = filepath.Join(".", s.SubDir, media.Filename)
			}
			categoryWriter.Add(PlaylistEntry{
				Name:     media.Name,
				Source:   source,
				Duration: int(math.Round(media.Duration)),
			})
		}

		if err := categoryWriter.Dump(); err != nil {
			return err
		}
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

	if s.CleanAllSymlinks {
		if err := cleanSymlinks(s, dataDir); err != nil {
			return err
		}
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
			if err := os.Symlink(targetPath, linkPath); err != nil && !os.IsExist(err) {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", linkPath, targetPath, err)
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
				if err := os.Symlink(targetPath, linkFile); err != nil && !os.IsExist(err) {
					return fmt.Errorf("failed to create symlink %s -> %s: %w", linkFile, targetPath, err)
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
				if err := os.Symlink(targetPath, linkFile); err != nil && !os.IsExist(err) {
					return fmt.Errorf("failed to create symlink %s -> %s: %w", linkFile, targetPath, err)
				}
			}
		}
	}
	return nil
}

// cleanSymlinks removes all symlinks below the data directory as well as
// top-level symlinks in the work directory that point into the data
// directory. This implements the --clean-symlinks flag so stale links from
// previous runs (renamed categories, removed media) do not accumulate.
func cleanSymlinks(s *config.Settings, dataDir string) error {
	if s.Quiet < 1 {
		fmt.Fprintln(os.Stderr, "removing old symlinks")
	}

	if fileExists(dataDir) {
		err := filepath.WalkDir(dataDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				// #nosec G122 - removing symlinks we created under our own data directory
				return os.Remove(path)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	// Home-category symlinks are created at the top level of the work
	// directory; only remove the ones that point into our data directory.
	entries, err := os.ReadDir(s.WorkDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		linkPath := filepath.Join(s.WorkDir, entry.Name())
		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(s.WorkDir, target)
		}
		absTarget, err := filepath.Abs(target)
		if err != nil {
			continue
		}
		absDataDir, err := filepath.Abs(dataDir)
		if err != nil {
			continue
		}
		if absTarget == absDataDir || strings.HasPrefix(absTarget, absDataDir+string(os.PathSeparator)) {
			if err := os.Remove(linkPath); err != nil {
				return err
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
		// #nosec G404 - math/rand is fine for shuffling playlist order; not security-sensitive
		rand.Shuffle(len(mediaList), func(i, j int) {
			mediaList[i], mediaList[j] = mediaList[j], mediaList[i]
		})
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// --- TxtWriter ---

// hrefRegex extracts the href attribute from HTML anchor lines when loading
// an existing HTML playlist in append mode.
var hrefRegex = regexp.MustCompile(`href="([^"]*)"`)

// TxtWriter handles writing playlist entries to a text file
type TxtWriter struct {
	s            *config.Settings
	path         string
	queue        []PlaylistEntry
	history      map[string]bool
	existingBody string
	start        string
	end          string
	formatter    func(PlaylistEntry) string
}

// NewTxtWriter creates a new TxtWriter instance for writing playlist entries to a text file
func NewTxtWriter(s *config.Settings) (*TxtWriter, error) {
	filename := s.OutputFilename
	if filename == "" {
		return nil, fmt.Errorf("output filename is required for txt mode")
	}
	// Validate that the resulting path stays within the work directory to prevent path traversal
	fullPath := filepath.Join(s.WorkDir, filename)
	cleanPath := filepath.Clean(fullPath)
	workDirAbs, err := filepath.Abs(s.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("invalid work directory: %w", err)
	}
	cleanPathAbs, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid output filename %s: %w", filename, err)
	}
	// Use filepath.Rel for cross-platform path validation
	rel, err := filepath.Rel(workDirAbs, cleanPathAbs)
	if err != nil {
		return nil, fmt.Errorf("invalid output filename %s: %w", filename, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("invalid output filename: path traversal detected in %s", filename)
	}

	return &TxtWriter{
		s:       s,
		path:    cleanPath,
		history: make(map[string]bool),
		formatter: func(e PlaylistEntry) string {
			return e.Source
		},
	}, nil
}

// LoadExisting reads an already existing output file so that its entries are
// kept and not duplicated when new entries are appended (--append).
func (w *TxtWriter) LoadExisting() error {
	// #nosec G304 - Path is user-configured output file for legitimate file operations
	data, err := os.ReadFile(w.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	body := strings.TrimPrefix(string(data), w.start)
	body = strings.TrimSuffix(body, w.end)
	w.existingBody = body

	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := hrefRegex.FindStringSubmatch(line); m != nil {
			w.history[html.UnescapeString(m[1])] = true
			continue
		}
		w.history[line] = true
	}
	return nil
}

// Add adds a playlist entry to the writer's queue
func (w *TxtWriter) Add(entry PlaylistEntry) {
	if !w.history[entry.Source] {
		w.queue = append(w.queue, entry)
		w.history[entry.Source] = true
	}
}

// Dump writes the existing (appended) content plus all queued playlist
// entries to the output file. The file is only created/replaced here, so a
// failed indexing run never truncates a previously written playlist.
func (w *TxtWriter) Dump() error {
	// #nosec G304 - Path is user-configured output file for legitimate file operations
	file, err := os.Create(w.path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if _, err := file.WriteString(w.start); err != nil {
		return err
	}
	if _, err := file.WriteString(w.existingBody); err != nil {
		return err
	}

	for _, entry := range w.queue {
		if _, err := file.WriteString(w.formatter(entry) + "\n"); err != nil {
			return err
		}
	}

	if _, err := file.WriteString(w.end); err != nil {
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
