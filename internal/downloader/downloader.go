// Package downloader provides file downloading functionality with rate limiting.
package downloader

import (
	"context"
	"crypto/md5" // #nosec G501 - MD5 used for file integrity verification, not cryptographic security
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darkace1998/jw-scripts/internal/api"
	"github.com/darkace1998/jw-scripts/internal/config"
	"github.com/darkace1998/jw-scripts/internal/httpx"
	"github.com/darkace1998/jw-scripts/internal/metadata"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

var (
	// ErrDiskLimitReached is returned when the disk space limit has been reached
	ErrDiskLimitReached = errors.New("disk limit reached")
	// ErrMissingTimestamp is returned when a required timestamp is missing
	ErrMissingTimestamp = errors.New("missing timestamp")
	// ErrCannotFreeDiskSpace is returned when disk space cannot be freed
	ErrCannotFreeDiskSpace = errors.New("cannot free more disk space")
)

// dirPerm is used for download directories. Media libraries are usually read
// by a media server running as a different user, so directories must be
// traversable by everyone.
const dirPerm = 0o755

// idleTimeout aborts a download when no data arrives for this long. It is a
// variable so tests can shorten it.
var idleTimeout = 2 * time.Minute

// downloadClient is used for all file downloads. It has no overall timeout
// (large videos can take a long time), but connection setup and response
// headers are bounded and stalled transfers are caught by idleTimeout.
var downloadClient = httpx.NewClient(0)

// MkdirAll creates a download directory that media servers can read.
func MkdirAll(dir string) error {
	// #nosec G301 - media library directories must be readable by media servers
	return os.MkdirAll(dir, dirPerm)
}

// ShowProgress reports whether download progress bars should be drawn: only
// when not in quiet mode and stderr is an interactive terminal, so logs from
// cron jobs and containers are not flooded with progress updates.
func ShowProgress(quiet int) bool {
	return quiet < 1 && term.IsTerminal(int(os.Stderr.Fd())) // #nosec G115 - file descriptors fit in int
}

// DownloadAll downloads all media files. Individual failures do not stop the
// run; they are summarized in the returned error once everything else has
// been processed.
func DownloadAll(s *config.Settings, data []*api.Category) error {
	wd := filepath.Join(s.WorkDir, s.SubDir)
	if err := MkdirAll(wd); err != nil {
		return err
	}

	var mediaList []*api.Media
	categoryOf := make(map[*api.Media]*api.Category)
	for _, cat := range data {
		for _, item := range cat.Contents {
			if media, ok := item.(*api.Media); ok {
				mediaList = append(mediaList, media)
				categoryOf[media] = cat
			}
		}
	}

	sort.SliceStable(mediaList, func(i, j int) bool {
		return mediaList[i].Date > mediaList[j].Date
	})

	var errs []error

	if s.DownloadSubtitles {
		if failed := downloadAllSubtitles(s, mediaList, wd); failed > 0 {
			errs = append(errs, fmt.Errorf("%d subtitle downloads failed", failed))
		}
	}

	if s.Download {
		if s.KeepFree > 0 && s.Warning && s.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "warning: disk space limit is set: old MP4 files in %s will be DELETED when free space drops below %d MiB (disable this warning with --no-warning)\n",
				wd, s.KeepFree/(1024*1024))
			// #nosec G115 - KeepFree is guaranteed positive by the enclosing condition
			if free, err := getFreeDiskSpace(wd); err == nil && free < uint64(s.KeepFree) {
				fmt.Fprintf(os.Stderr, "warning: free disk space (%d MiB) is already below the limit (%d MiB); the space limit seems wrong\n",
					free/(1024*1024), s.KeepFree/(1024*1024))
			}
		}

		if s.Quiet < 1 {
			fmt.Fprintln(os.Stderr, "scanning local files")
		}

		var downloadList []*api.Media
		checkedFiles := make(map[string]bool)
		for _, media := range mediaList {
			if media.Filename == "" || checkedFiles[media.Filename] {
				continue
			}
			checkedFiles[media.Filename] = true
			if !checkMedia(s, media, wd) {
				downloadList = append(downloadList, media)
			}
		}

		failed := 0
		for i, media := range downloadList {
			if s.KeepFree > 0 {
				if err := diskCleanup(s, wd, media); err != nil {
					if reason := diskLimitReason(err); reason != "" {
						if s.Quiet < 2 {
							fmt.Fprintf(os.Stderr, "not enough disk space (%s), skipping: %s\n", reason, media.Name)
						}
						continue
					}
					return errors.Join(append(errs, err)...)
				}
			}

			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "[%d/%d] ", i+1, len(downloadList))
			}
			if err := downloadMedia(s, media, wd); err != nil {
				failed++
				if s.Quiet < 2 {
					fmt.Fprintf(os.Stderr, "download failed for %s: %v\n", media.Name, err)
				}
			}
		}
		if failed > 0 {
			errs = append(errs, fmt.Errorf("%d of %d downloads failed", failed, len(downloadList)))
		}
	}

	if s.WriteMetadata {
		if failed := writeAllMetadata(s, mediaList, categoryOf, wd); failed > 0 {
			errs = append(errs, fmt.Errorf("failed to write metadata for %d files", failed))
		}
	}

	return errors.Join(errs...)
}

// diskLimitReason explains a diskCleanup error that means "skip this file",
// or returns "" for unexpected errors.
func diskLimitReason(err error) string {
	switch {
	case errors.Is(err, ErrDiskLimitReached):
		return "no older videos left to remove"
	case errors.Is(err, ErrMissingTimestamp):
		return "media has no date to compare with old videos"
	case errors.Is(err, ErrCannotFreeDiskSpace):
		return "no MP4 files left to remove"
	default:
		return ""
	}
}

// writeAllMetadata writes an NFO metadata file (read by Jellyfin, Emby, Kodi
// and Plex NFO agents) next to every media file that exists locally.
// Unchanged NFO files are not rewritten. It returns the number of failures.
func writeAllMetadata(s *config.Settings, mediaList []*api.Media, categoryOf map[*api.Media]*api.Category, directory string) int {
	if s.Quiet < 1 {
		fmt.Fprintln(os.Stderr, "writing metadata")
	}

	written := make(map[string]bool)
	count, failed := 0, 0
	for _, media := range mediaList {
		if media.Filename == "" || written[media.Filename] {
			continue
		}
		written[media.Filename] = true

		if !fileExists(filepath.Join(directory, media.Filename)) {
			continue
		}

		meta := metadata.FromMedia(s.Lang, categoryOf[media], media)
		if err := metadata.Write(directory, media.Filename, meta); err != nil {
			failed++
			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "failed to write metadata for %s: %v\n", media.Filename, err)
			}
			continue
		}
		count++
	}

	if s.Quiet < 1 {
		fmt.Fprintf(os.Stderr, "wrote metadata for %d files\n", count)
	}
	return failed
}

// downloadAllSubtitles downloads missing subtitle files and returns the
// number of failures.
func downloadAllSubtitles(s *config.Settings, mediaList []*api.Media, directory string) int {
	var queue []*api.Media
	queued := make(map[string]bool)
	for _, media := range mediaList {
		if media.SubtitleURL == "" || media.SubtitleFilename == "" || queued[media.SubtitleFilename] {
			continue
		}
		queued[media.SubtitleFilename] = true
		subtitlePath := filepath.Join(directory, media.SubtitleFilename)
		if s.OverwriteBad || !fileExists(subtitlePath) {
			queue = append(queue, media)
		}
	}

	failed := 0
	for i, media := range queue {
		if s.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "[%d/%d] downloading: %s\n", i+1, len(queue), media.SubtitleFilename)
		}
		// Download to a temporary file and rename on success so a failed
		// download never leaves a truncated subtitle file behind that would
		// be treated as complete on the next run.
		subtitlePath := filepath.Join(directory, media.SubtitleFilename)
		tmpPath := subtitlePath + ".part"
		if err := DownloadFile(media.SubtitleURL, tmpPath, false, 0, false); err != nil {
			failed++
			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "failed to download subtitle %s: %v\n", media.SubtitleFilename, err)
			}
			if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) && s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "failed to clean up partial file %s: %v\n", tmpPath, removeErr)
			}
			continue
		}
		if err := os.Rename(tmpPath, subtitlePath); err != nil {
			failed++
			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "failed to finalize subtitle %s: %v\n", media.SubtitleFilename, err)
			}
		}
	}

	return failed
}

// checkMedia reports whether a media file already exists locally and, with
// --fix-broken, whether it has the expected size (and MD5 with --checksum).
func checkMedia(s *config.Settings, media *api.Media, directory string) bool {
	file := filepath.Join(directory, media.Filename)
	if !fileExists(file) {
		return false
	}

	if s.OverwriteBad {
		if err := verifyFile(s, media, file); err != nil {
			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "%v: %s\n", err, file)
			}
			return false
		}
	}

	return true
}

// verifyFile checks path against the size reported by the API and, when
// --checksum is enabled, against the MD5 checksum.
func verifyFile(s *config.Settings, media *api.Media, path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if media.Size > 0 && fi.Size() != media.Size {
		return fmt.Errorf("size mismatch (got %d bytes, expected %d)", fi.Size(), media.Size)
	}
	if s.Checksums && media.MD5 != "" {
		ok, err := CheckMD5(path, media.MD5)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("checksum mismatch")
		}
	}
	return nil
}

// downloadMedia downloads (or, for imported media, copies) a media file to a
// ".part" file, verifies it and moves it into place.
func downloadMedia(s *config.Settings, media *api.Media, directory string) error {
	file := filepath.Join(directory, media.Filename)
	tmpFile := file + ".part"

	if media.LocalPath != "" {
		if s.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "importing: %s\n", media.Filename)
		}
		if err := copyFile(media.LocalPath, tmpFile); err != nil {
			_ = os.Remove(tmpFile)
			return err
		}
	} else {
		resume := fileExists(tmpFile)
		if s.Quiet < 2 {
			verb := "downloading"
			if resume {
				verb = "resuming"
			}
			fmt.Fprintf(os.Stderr, "%s: %s (%s)\n", verb, media.Filename, media.Name)
		}
		if err := DownloadFile(media.URL, tmpFile, resume, s.RateLimit, ShowProgress(s.Quiet)); err != nil {
			return err
		}

		if err := verifyFile(s, media, tmpFile); err != nil {
			if !resume {
				_ = os.Remove(tmpFile)
				return err
			}
			// The partial file may belong to an older version of the media;
			// start over once.
			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "resumed download is broken (%v), restarting: %s\n", err, media.Filename)
			}
			if err := os.Remove(tmpFile); err != nil {
				return err
			}
			if err := DownloadFile(media.URL, tmpFile, false, s.RateLimit, ShowProgress(s.Quiet)); err != nil {
				return err
			}
			if err := verifyFile(s, media, tmpFile); err != nil {
				_ = os.Remove(tmpFile)
				return err
			}
		}
	}

	if media.Date > 0 {
		t := time.Unix(media.Date, 0)
		if err := os.Chtimes(tmpFile, t, t); err != nil {
			return err
		}
	}

	return os.Rename(tmpFile, file)
}

// copyFile copies src to dst (used for --import).
func copyFile(src, dst string) error {
	// #nosec G304 - src is a file from the user-supplied import directory
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	// #nosec G304 - dst is inside the managed download directory
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// DownloadFile downloads a file from a URL to a specified path. With resume,
// an existing file at path is continued with a range request; a file that is
// already complete is left as is, and one that cannot be continued is
// downloaded again from the start.
func DownloadFile(rawURL, path string, resume bool, rateLimit float64, showProgress bool) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %q", parsedURL.Scheme)
	}

	var start int64
	if resume {
		if fi, err := os.Stat(path); err == nil {
			start = fi.Size()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), http.NoBody)
	if err != nil {
		return err
	}
	if start > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
	}

	// #nosec G704 - URL scheme is validated above to only allow http/https; this is a legitimate file download
	resp, err := downloadClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	appendData := false
	switch {
	case start > 0 && resp.StatusCode == http.StatusRequestedRangeNotSatisfiable:
		// Nothing left to fetch if the partial file already has the full
		// size; otherwise it is stale (e.g. the file changed on the server).
		if total, ok := contentRangeTotal(resp.Header.Get("Content-Range")); ok && total == start {
			return nil
		}
		_ = resp.Body.Close()
		return DownloadFile(rawURL, path, false, rateLimit, showProgress)
	case start > 0 && resp.StatusCode == http.StatusPartialContent:
		if first, ok := contentRangeStart(resp.Header.Get("Content-Range")); !ok || first != start {
			// The server sent a different range than requested
			_ = resp.Body.Close()
			return DownloadFile(rawURL, path, false, rateLimit, showProgress)
		}
		appendData = true
	case resp.StatusCode == http.StatusOK:
		// Full content (fresh download, or the server ignored the range)
		start = 0
	default:
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	var out *os.File
	if appendData {
		// #nosec G304 - Path is from download logic for legitimate file operations
		out, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	} else {
		// #nosec G304 - Path is from download logic for legitimate file operations
		out, err = os.Create(path)
	}
	if err != nil {
		return err
	}

	var body io.Reader = newIdleTimeoutReader(resp.Body, idleTimeout, cancel)
	if rateLimit > 0 {
		body = newThrottledReader(body, rateLimit)
	}

	var dst io.Writer = out
	if showProgress {
		size := int64(-1) // unknown length: indeterminate progress
		if resp.ContentLength >= 0 {
			size = resp.ContentLength + start
		}
		bar := progressbar.NewOptions64(
			size,
			progressbar.OptionSetDescription("downloading"),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionThrottle(100*time.Millisecond),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprint(os.Stderr, "\n")
			}),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
		)
		_ = bar.Add64(start) // progress bar errors must not stop the download
		dst = io.MultiWriter(out, bar)
	}

	if _, err := io.Copy(dst, body); err != nil {
		_ = out.Close()
		if ctx.Err() != nil {
			return fmt.Errorf("download stalled: no data received for %s", idleTimeout)
		}
		return err
	}
	return out.Close()
}

// contentRangeTotal returns the total size from a Content-Range header such
// as "bytes */1234" or "bytes 0-99/1234".
func contentRangeTotal(header string) (int64, bool) {
	i := strings.LastIndexByte(header, '/')
	if i < 0 {
		return 0, false
	}
	total, err := strconv.ParseInt(header[i+1:], 10, 64)
	return total, err == nil
}

// contentRangeStart returns the first byte position from a Content-Range
// header such as "bytes 100-199/1234".
func contentRangeStart(header string) (int64, bool) {
	rest, ok := strings.CutPrefix(header, "bytes ")
	if !ok {
		return 0, false
	}
	first, _, ok := strings.Cut(rest, "-")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(first, 10, 64)
	return n, err == nil
}

// idleTimeoutReader calls cancel when no Read returns data for longer than
// the timeout, which aborts the underlying request.
type idleTimeoutReader struct {
	r       io.Reader
	timeout time.Duration
	timer   *time.Timer
	once    sync.Once
}

func newIdleTimeoutReader(r io.Reader, timeout time.Duration, cancel context.CancelFunc) *idleTimeoutReader {
	return &idleTimeoutReader{r: r, timeout: timeout, timer: time.AfterFunc(timeout, cancel)}
}

func (r *idleTimeoutReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.timer.Reset(r.timeout)
	}
	if err != nil {
		r.once.Do(func() { r.timer.Stop() })
	}
	return n, err
}

// CheckMD5 calculates the MD5 checksum of a file and compares it to the expected checksum.
// Note: MD5 is used here for file integrity verification (not cryptographic security)
// as it matches the checksum format provided by the external API.
func CheckMD5(path, expectedMD5 string) (bool, error) {
	// #nosec G304 - Path is for file checksum verification in download process
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	h := md5.New() // #nosec G401 - MD5 used for file integrity verification, not cryptographic security
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}

	return strings.EqualFold(fmt.Sprintf("%x", h.Sum(nil)), expectedMD5), nil
}

func diskCleanup(s *config.Settings, directory string, referenceMedia *api.Media) error {
	if s.KeepFree == 0 || referenceMedia.Size == 0 {
		return nil
	}

	if !fileExists(directory) {
		return nil
	}

	for {
		free, err := getFreeDiskSpace(directory)
		if err != nil {
			return err
		}

		needed := referenceMedia.Size + s.KeepFree
		if needed < 0 {
			// Integer overflow detected: referenceMedia.Size + s.KeepFree exceeded int64 max value
			// This can happen with very large file sizes on 32-bit systems
			// Skip the disk space check to avoid incorrect behavior
			break
		}
		if free > uint64(needed) {
			break
		}

		if s.Quiet < 1 {
			fmt.Fprintf(os.Stderr, "free space: %d MiB, needed: %d MiB\n", free/(1024*1024), needed/(1024*1024))
		}

		if referenceMedia.Date == 0 {
			return ErrMissingTimestamp
		}

		oldest, err := getOldestMP4(directory)
		if err != nil {
			return err
		}

		if referenceMedia.Date <= oldest.ModTime().Unix() {
			return ErrDiskLimitReached
		}

		if s.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "removing old video: %s\n", oldest.Name())
		}
		if err := removeMediaFile(directory, oldest.Name()); err != nil {
			return err
		}
	}
	return nil
}

// removeMediaFile deletes a media file together with its subtitle and
// metadata sidecar files.
func removeMediaFile(directory, name string) error {
	if err := os.Remove(filepath.Join(directory, name)); err != nil {
		return err
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	for _, companion := range []string{
		base + ".vtt",
		filepath.Base(metadata.NFOPath(directory, name)),
		filepath.Base(metadata.LegacySidecarPath(directory, name)),
	} {
		if err := os.Remove(filepath.Join(directory, companion)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func getOldestMP4(directory string) (os.FileInfo, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	var oldest os.FileInfo
	var oldestModTime time.Time
	for _, file := range files {
		if !file.IsDir() && strings.EqualFold(filepath.Ext(file.Name()), ".mp4") {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if oldest == nil || info.ModTime().Before(oldestModTime) {
				oldest = info
				oldestModTime = info.ModTime()
			}
		}
	}

	if oldest == nil {
		return nil, ErrCannotFreeDiskSpace
	}

	return oldest, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// throttledReader is a reader that is throttled to a certain rate.
type throttledReader struct {
	r         io.Reader
	rateLimit float64 // bytes per second
	startTime time.Time
	totalRead int64
}

func newThrottledReader(r io.Reader, rateLimit float64) *throttledReader {
	return &throttledReader{
		r:         r,
		rateLimit: rateLimit * 1024 * 1024, // Convert MB/s to bytes/s
		startTime: time.Now(),
	}
}

func (r *throttledReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if err != nil || n == 0 || r.rateLimit <= 0 {
		return n, err
	}

	r.totalRead += int64(n)
	elapsed := time.Since(r.startTime).Seconds()

	// Calculate expected time for the data read so far
	expectedTime := float64(r.totalRead) / r.rateLimit

	// If we're reading too fast, sleep
	if elapsed < expectedTime {
		sleepTime := time.Duration((expectedTime - elapsed) * float64(time.Second))
		time.Sleep(sleepTime)
	}

	return n, err
}
