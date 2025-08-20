package api

import (
	"reflect"
	"testing"
	"time"
)

func TestGetBestVideo(t *testing.T) {
	testCases := []struct {
		name      string
		files     []File
		quality   int
		subtitles bool
		want      *File
	}{
		{
			name: "select highest quality",
			files: []File{
				{ProgressiveDownloadURL: "720p.mp4", Label: "720p"},
				{ProgressiveDownloadURL: "480p.mp4", Label: "480p"},
				{ProgressiveDownloadURL: "360p.mp4", Label: "360p"},
			},
			quality:   720,
			subtitles: false,
			want:      &File{ProgressiveDownloadURL: "720p.mp4", Label: "720p"},
		},
		{
			name: "quality limit",
			files: []File{
				{ProgressiveDownloadURL: "720p.mp4", Label: "720p"},
				{ProgressiveDownloadURL: "480p.mp4", Label: "480p"},
			},
			quality:   480,
			subtitles: false,
			want:      &File{ProgressiveDownloadURL: "480p.mp4", Label: "480p"},
		},
		{
			name: "prefer subtitles",
			files: []File{
				{ProgressiveDownloadURL: "720p.mp4", Label: "720p", Subtitled: false},
				{ProgressiveDownloadURL: "720p_sub.mp4", Label: "720p", Subtitled: true},
			},
			quality:   720,
			subtitles: true,
			want:      &File{ProgressiveDownloadURL: "720p_sub.mp4", Label: "720p", Subtitled: true},
		},
		{
			name: "no matching subtitles",
			files: []File{
				{ProgressiveDownloadURL: "720p.mp4", Label: "720p", Subtitled: false},
				{ProgressiveDownloadURL: "480p.mp4", Label: "480p", Subtitled: false},
			},
			quality:   720,
			subtitles: true,                                                                       // prefer subs, but none available
			want:      &File{ProgressiveDownloadURL: "720p.mp4", Label: "720p", Subtitled: false}, // should still pick best quality
		},
		{
			name:      "empty file list",
			files:     []File{},
			quality:   720,
			subtitles: false,
			want:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getBestVideo(tc.files, tc.quality, tc.subtitles)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("getBestVideo() got = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetFriendlyFilename(t *testing.T) {
	testCases := []struct {
		name string
		n    string
		url  string
		safe bool
		want string
	}{
		{
			name: "valid name and url",
			n:    "My Awesome Video",
			url:  "http://example.com/video.mp4",
			safe: true,
			want: "My Awesome Video.mp4",
		},
		{
			name: "empty url",
			n:    "My Awesome Video",
			url:  "",
			safe: true,
			want: "",
		},
		{
			name: "name with special chars",
			n:    "My:Awesome/Video",
			url:  "http://example.com/video.mp4",
			safe: true,
			want: "My-AwesomeVideo.mp4",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getFriendlyFilename(tc.n, tc.url, tc.safe)
			if got != tc.want {
				t.Errorf("getFriendlyFilename() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetFriendlySubtitleFilename(t *testing.T) {
	testCases := []struct {
		name        string
		n           string
		subtitleURL string
		safe        bool
		want        string
	}{
		{
			name:        "valid name and subtitle url",
			n:           "My Awesome Video",
			subtitleURL: "http://example.com/subtitle.vtt",
			safe:        true,
			want:        "My Awesome Video.vtt",
		},
		{
			name:        "empty subtitle url",
			n:           "My Awesome Video",
			subtitleURL: "",
			safe:        true,
			want:        "",
		},
		{
			name:        "name with special chars",
			n:           "My:Awesome/Video",
			subtitleURL: "http://example.com/subtitle.vtt",
			safe:        true,
			want:        "My-AwesomeVideo.vtt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getFriendlySubtitleFilename(tc.n, tc.subtitleURL, tc.safe)
			if got != tc.want {
				t.Errorf("getFriendlySubtitleFilename() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetFilename(t *testing.T) {
	testCases := []struct {
		name string
		url  string
		safe bool
		want string
	}{
		{
			name: "valid url",
			url:  "http://example.com/path/to/file.txt",
			safe: true,
			want: "file.txt",
		},
		{
			name: "empty url",
			url:  "",
			safe: true,
			want: "",
		},
		{
			name: "url with special chars",
			url:  "http://example.com/a<b>c.txt",
			safe: true,
			want: "abc.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getFilename(tc.url, tc.safe)
			if got != tc.want {
				t.Errorf("getFilename() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatFilename(t *testing.T) {
	testCases := []struct {
		name string
		s    string
		safe bool
		want string
	}{
		{
			name: "no forbidden characters",
			s:    "hello_world.txt",
			safe: true,
			want: "hello_world.txt",
		},
		{
			name: "safe mode with forbidden characters",
			s:    "a<b>c|d?e*f.txt",
			safe: true,
			want: "abcdef.txt",
		},
		{
			name: "not safe mode with forbidden characters",
			s:    "a/b.txt",
			safe: false,
			want: "ab.txt",
		},
		{
			name: "safe mode with quotes and colons",
			s:    `a"b:c.txt`,
			safe: true,
			want: "a'b-c.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatFilename(tc.s, tc.safe)
			if got != tc.want {
				t.Errorf("formatFilename() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	testCases := []struct {
		name      string
		dateStr   string
		want      time.Time
		expectErr bool
	}{
		{
			name:      "valid date with milliseconds",
			dateStr:   "2021-06-25T10:00:00.123Z",
			want:      time.Date(2021, 6, 25, 10, 0, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:    "valid date without milliseconds",
			dateStr: "2021-06-25T10:00:00",
			// The original function strips the Z, so this will be parsed as local time.
			// Let's adjust the test to handle this.
			// No, the original function's regex only strips `.123Z`. It doesn't handle a missing `Z`.
			// Let's re-read the function.
			// `re := regexp.MustCompile(`\.[0-9]+Z$`)`
			// `dateString = re.ReplaceAllString(dateString, "")`
			// `return time.Parse("2006-01-02T15:04:05", dateString)`
			// If the input is "2021-06-25T10:00:00", the regex doesn't match, and it's passed to time.Parse.
			// time.Parse without a zone will assume UTC. So my original assumption was correct.
			want:      time.Date(2021, 6, 25, 10, 0, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:      "invalid date",
			dateStr:   "not a date",
			want:      time.Time{},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDate(tc.dateStr)
			if (err != nil) != tc.expectErr {
				t.Errorf("parseDate() error = %v, expectErr %v", err, tc.expectErr)
				return
			}
			if !tc.expectErr && !got.Equal(tc.want) {
				t.Errorf("parseDate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMakeUniqueFilename(t *testing.T) {
	testCases := []struct {
		name          string
		filename      string
		usedFilenames map[string]bool
		want          string
	}{
		{
			name:          "unique filename",
			filename:      "video.mp4",
			usedFilenames: make(map[string]bool),
			want:          "video.mp4",
		},
		{
			name:     "duplicate filename",
			filename: "video.mp4",
			usedFilenames: map[string]bool{
				"video.mp4": true,
			},
			want: "video (1).mp4",
		},
		{
			name:     "multiple duplicates",
			filename: "video.mp4",
			usedFilenames: map[string]bool{
				"video.mp4":     true,
				"video (1).mp4": true,
			},
			want: "video (2).mp4",
		},
		{
			name:          "empty filename",
			filename:      "",
			usedFilenames: make(map[string]bool),
			want:          "",
		},
		{
			name:     "filename without extension",
			filename: "video",
			usedFilenames: map[string]bool{
				"video": true,
			},
			want: "video (1)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy of the map to avoid modifying the test case
			usedFilenames := make(map[string]bool)
			for k, v := range tc.usedFilenames {
				usedFilenames[k] = v
			}
			
			got := makeUniqueFilename(tc.filename, usedFilenames)
			if got != tc.want {
				t.Errorf("makeUniqueFilename() = %q, want %q", got, tc.want)
			}
			
			// Verify the filename was added to the used map
			if tc.filename != "" && !usedFilenames[got] {
				t.Errorf("makeUniqueFilename() did not add %q to usedFilenames map", got)
			}
		})
	}
}
