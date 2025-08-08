package api

import (
	"reflect"
	"testing"
	"time"
)

func TestGetBestVideo(t *testing.T) {
	// The anonymous struct is defined here to match the one in getBestVideo
	type file struct {
		ProgressiveDownloadURL string `json:"progressiveDownloadURL"`
		Checksum               string `json:"checksum"`
		Filesize               int64  `json:"filesize"`
		Duration               int    `json:"duration"`
		Label                  string `json:"label"`
		Subtitled              bool   `json:"subtitled"`
		Subtitles              struct {
			URL string `json:"url"`
		} `json:"subtitles"`
	}

	testCases := []struct {
		name      string
		files     []file
		quality   int
		subtitles bool
		want      *file
	}{
		{
			name: "select highest quality",
			files: []file{
				{ProgressiveDownloadURL: "720p.mp4", Label: "720p"},
				{ProgressiveDownloadURL: "480p.mp4", Label: "480p"},
				{ProgressiveDownloadURL: "360p.mp4", Label: "360p"},
			},
			quality:   720,
			subtitles: false,
			want:      &file{ProgressiveDownloadURL: "720p.mp4", Label: "720p"},
		},
		{
			name: "quality limit",
			files: []file{
				{ProgressiveDownloadURL: "720p.mp4", Label: "720p"},
				{ProgressiveDownloadURL: "480p.mp4", Label: "480p"},
			},
			quality:   480,
			subtitles: false,
			want:      &file{ProgressiveDownloadURL: "480p.mp4", Label: "480p"},
		},
		{
			name: "prefer subtitles",
			files: []file{
				{ProgressiveDownloadURL: "720p.mp4", Label: "720p", Subtitled: false},
				{ProgressiveDownloadURL: "720p_sub.mp4", Label: "720p", Subtitled: true},
			},
			quality:   720,
			subtitles: true,
			want:      &file{ProgressiveDownloadURL: "720p_sub.mp4", Label: "720p", Subtitled: true},
		},
		{
			name: "no matching subtitles",
			files: []file{
				{ProgressiveDownloadURL: "720p.mp4", Label: "720p", Subtitled: false},
				{ProgressiveDownloadURL: "480p.mp4", Label: "480p", Subtitled: false},
			},
			quality:   720,
			subtitles: true, // prefer subs, but none available
			want:      &file{ProgressiveDownloadURL: "720p.mp4", Label: "720p", Subtitled: false}, // should still pick best quality
		},
		{
			name:      "empty file list",
			files:     []file{},
			quality:   720,
			subtitles: false,
			want:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Because the function expects a slice of a specific anonymous struct,
			// we can't directly pass our named struct `file`.
			// We need to convert it. This is a bit of a hack, but it's the only way
			// without changing the original function's signature.
			// A better approach would be to use reflection to create the slice, but that's too complex.
			// Let's try to pass it directly. Go might be smart enough.
			// No, it's not. The type is different.
			// The easiest way is to copy the struct definition inside the test cases.

			// Re-running the thought process. The previous thought of defining the struct inside the test was correct.
			// But it's cleaner to define it once. Let's see if I can make it work by defining the type.
			// The problem is that `getBestVideo` uses an anonymous struct.
			// The most pragmatic way is to just copy the struct definition into the test.
			// It makes the test verbose, but it will work.

			// Let's rewrite the testCases with the anonymous struct.
			// This is getting too complicated. I will simplify the test creation.
			// I'll define the struct inside the test function.

			// The anonymous struct definition is identical to the one in getBestVideo
			files := make([]struct {
				ProgressiveDownloadURL string `json:"progressiveDownloadURL"`
				Checksum               string `json:"checksum"`
				Filesize               int64  `json:"filesize"`
				Duration               int    `json:"duration"`
				Label                  string `json:"label"`
				Subtitled              bool   `json:"subtitled"`
				Subtitles              struct {
					URL string `json:"url"`
				} `json:"subtitles"`
			}, len(tc.files))

			for i, f := range tc.files {
				files[i].ProgressiveDownloadURL = f.ProgressiveDownloadURL
				files[i].Label = f.Label
				files[i].Subtitled = f.Subtitled
			}

			var want *struct {
				ProgressiveDownloadURL string `json:"progressiveDownloadURL"`
				Checksum               string `json:"checksum"`
				Filesize               int64  `json:"filesize"`
				Duration               int    `json:"duration"`
				Label                  string `json:"label"`
				Subtitled              bool   `json:"subtitled"`
				Subtitles              struct {
					URL string `json:"url"`
				} `json:"subtitles"`
			}

			if tc.want != nil {
				want = &struct {
					ProgressiveDownloadURL string `json:"progressiveDownloadURL"`
					Checksum               string `json:"checksum"`
					Filesize               int64  `json:"filesize"`
					Duration               int    `json:"duration"`
					Label                  string `json:"label"`
					Subtitled              bool   `json:"subtitled"`
					Subtitles              struct {
						URL string `json:"url"`
					} `json:"subtitles"`
				}{
					ProgressiveDownloadURL: tc.want.ProgressiveDownloadURL,
					Label:                  tc.want.Label,
					Subtitled:              tc.want.Subtitled,
				}
			}

			got := getBestVideo(files, tc.quality, tc.subtitles)

			// We need to compare the pointers' values, not the pointers themselves.
			if !reflect.DeepEqual(got, want) {
				t.Errorf("getBestVideo() got = %v, want %v", got, want)
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
			want: "My.AwesomeVideo.mp4",
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
			want: "a'b.c.txt",
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
			name:      "valid date without milliseconds",
			dateStr:   "2021-06-25T10:00:00",
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
