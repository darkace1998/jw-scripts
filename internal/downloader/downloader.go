package downloader

import (
	"crypto/md5" // #nosec G501 - MD5 used for file integrity verification, not cryptographic security
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/allejok96/jwb-go/internal/api"
	"github.com/allejok96/jwb-go/internal/config"
	"github.com/schollz/progressbar/v3"
)

var (
	// ErrDiskLimitReached is returned when the disk space limit has been reached
	ErrDiskLimitReached = errors.New("disk limit reached")
	// ErrMissingTimestamp is returned when a required timestamp is missing
	ErrMissingTimestamp = errors.New("missing timestamp")
	// ErrCannotFreeDiskSpace is returned when disk space cannot be freed
	ErrCannotFreeDiskSpace = errors.New("cannot free more disk space")
)

// DownloadAll downloads all media files.
func DownloadAll(s *config.Settings, data []*api.Category) error {
	wd := filepath.Join(s.WorkDir, s.SubDir)
	if err := os.MkdirAll(wd, 0o750); err != nil {
		return err
	}

	var mediaList []*api.Media
	for _, cat := range data {
		for _, item := range cat.Contents {
			if media, ok := item.(*api.Media); ok {
				mediaList = append(mediaList, media)
			}
		}
	}

	sort.Slice(mediaList, func(i, j int) bool {
		return mediaList[i].Date > mediaList[j].Date
	})

	if s.DownloadSubtitles {
		if err := downloadAllSubtitles(s, mediaList, wd); err != nil {
			return err
		}
	}

	if !s.Download {
		return nil
	}

	if s.Quiet < 1 {
		fmt.Fprintln(os.Stderr, "scanning local files")
	}

	var downloadList []*api.Media
	checkedFiles := make(map[string]bool)
	for _, media := range mediaList {
		if !checkedFiles[media.Filename] {
			checkedFiles[media.Filename] = true
			if !checkMedia(s, media, wd) {
				downloadList = append(downloadList, media)
			}
		}
	}

	for i, media := range downloadList {
		if s.KeepFree > 0 {
			if err := diskCleanup(s, wd, media); err != nil {
				if err == ErrDiskLimitReached || err == ErrMissingTimestamp {
					if s.Quiet < 2 {
						fmt.Fprintf(os.Stderr, "low disk space and missing metadata, skipping: %s\n", media.Name)
					}
					continue
				}
				return err
			}
		}

		if s.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "[%d/%d] ", i+1, len(downloadList))
		}
		if err := downloadMedia(s, media, wd); err != nil {
			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "download failed for %s: %v\n", media.Name, err)
			}
		}
	}

	return nil
}

func downloadAllSubtitles(s *config.Settings, mediaList []*api.Media, directory string) error {
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return err
	}

	var queue []*api.Media
	for _, media := range mediaList {
		if media.SubtitleURL != "" {
			subtitlePath := filepath.Join(directory, media.SubtitleFilename)
			if s.OverwriteBad || !fileExists(subtitlePath) {
				queue = append(queue, media)
			}
		}
	}

	for i, media := range queue {
		if s.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "[%d/%d] downloading: %s\n", i+1, len(queue), media.SubtitleFilename)
		}
		subtitlePath := filepath.Join(directory, media.SubtitleFilename)
		if err := DownloadFile(s, media.SubtitleURL, subtitlePath, false, 0); err != nil {
			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "failed to download subtitle %s: %v\n", media.SubtitleFilename, err)
			}
			// Clean up any empty or partial files created during failed download
			if fi, statErr := os.Stat(subtitlePath); statErr == nil && fi.Size() == 0 {
				if removeErr := os.Remove(subtitlePath); removeErr != nil && s.Quiet < 2 {
					fmt.Fprintf(os.Stderr, "failed to clean up empty file %s: %v\n", media.SubtitleFilename, removeErr)
				}
			}
		}
	}

	return nil
}

func checkMedia(s *config.Settings, media *api.Media, directory string) bool {
	file := filepath.Join(directory, media.Filename)
	if !fileExists(file) {
		return false
	}

	if s.OverwriteBad {
		fi, err := os.Stat(file)
		if err != nil {
			return false
		}

		if media.Size > 0 && fi.Size() != media.Size {
			if s.Quiet < 2 {
				fmt.Fprintf(os.Stderr, "size mismatch: %s\n", file)
			}
			return false
		}

		if s.Checksums && media.MD5 != "" {
			ok, err := CheckMD5(file, media.MD5)
			if err != nil || !ok {
				if s.Quiet < 2 {
					fmt.Fprintf(os.Stderr, "checksum mismatch: %s\n", file)
				}
				return false
			}
		}
	}

	return true
}

func downloadMedia(s *config.Settings, media *api.Media, directory string) error {
	file := filepath.Join(directory, media.Filename)
	tmpFile := file + ".part"

	if fileExists(tmpFile) {
		if s.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "resuming: %s (%s)\n", media.Filename, media.Name)
		}
		if err := DownloadFile(s, media.URL, tmpFile, true, s.RateLimit); err != nil {
			return err
		}

		// Validate resumed download if we have expected file size or checksum
		if media.Size > 0 || (s.Checksums && media.MD5 != "") {
			fi, err := os.Stat(tmpFile)
			if err == nil {
				// Check file size if available
				if media.Size > 0 && fi.Size() != media.Size {
					if s.Quiet < 2 {
						fmt.Fprintf(os.Stderr, "resumed download size mismatch, restarting: %s\n", media.Filename)
					}
					if err := os.Remove(tmpFile); err != nil {
						return err
					}
					if err := DownloadFile(s, media.URL, tmpFile, false, s.RateLimit); err != nil {
						return err
					}
				} else if s.Checksums && media.MD5 != "" {
					// Verify checksum if enabled and available
					if ok, err := CheckMD5(tmpFile, media.MD5); err == nil && !ok {
						if s.Quiet < 2 {
							fmt.Fprintf(os.Stderr, "resumed download checksum mismatch, restarting: %s\n", media.Filename)
						}
						if err := os.Remove(tmpFile); err != nil {
							return err
						}
						if err := DownloadFile(s, media.URL, tmpFile, false, s.RateLimit); err != nil {
							return err
						}
					}
				}
			}
		}
	} else {
		if s.Quiet < 2 {
			fmt.Fprintf(os.Stderr, "downloading: %s (%s)\n", media.Filename, media.Name)
		}
		if err := DownloadFile(s, media.URL, tmpFile, false, s.RateLimit); err != nil {
			return err
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

// DownloadFile downloads a file from a URL to a specified path.
func DownloadFile(s *config.Settings, url, path string, resume bool, rateLimit float64) error {
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return err
	}

	var start int64
	if resume {
		fi, err := os.Stat(path)
		if err == nil {
			start = fi.Size()
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	var out *os.File
	if resume {
		// #nosec G304 - Path is from download logic for legitimate file operations
		out, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	} else {
		// #nosec G304 - Path is from download logic for legitimate file operations
		out, err = os.Create(path)
	}
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	size := resp.ContentLength + start

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
	if err := bar.Add64(start); err != nil {
		// Log error but continue - progress bar errors shouldn't stop download
		fmt.Fprintf(os.Stderr, "Progress bar error: %v\n", err)
	}

	var body io.Reader = resp.Body
	if rateLimit > 0 {
		body = newThrottledReader(body, rateLimit)
	}

	_, err = io.Copy(io.MultiWriter(out, bar), body)
	return err
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

	return fmt.Sprintf("%x", h.Sum(nil)) == expectedMD5, nil
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
			// Overflow or negative values - skip the check
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
		if err := os.Remove(filepath.Join(directory, oldest.Name())); err != nil {
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
		if !file.IsDir() && filepath.Ext(file.Name()) == ".mp4" {
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
	return !os.IsNotExist(err)
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
