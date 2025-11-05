// Package api provides client and data structures for interacting with the JW.org API.
package api

// Category represents a category of media on JW Broadcasting.
type Category struct {
	Key      string
	Name     string
	Home     bool
	Contents []interface{} // Can contain either *Category or *Media
}

// Media represents a single media item, like a video or audio file.
type Media struct {
	Date                     int64
	Duration                 float64
	MD5                      string
	Name                     string
	Size                     int64
	SubtitleURL              string
	URL                      string
	Filename                 string
	FriendlyName             string
	SubtitleFilename         string
	FriendlySubtitleFilename string
}

// File represents a media file, like a video or audio file.
type File struct {
	ProgressiveDownloadURL string    `json:"progressiveDownloadURL"`
	Checksum               string    `json:"checksum"`
	Filesize               int64     `json:"filesize"`
	Duration               float64   `json:"duration"`
	Label                  string    `json:"label"`
	Subtitled              bool      `json:"subtitled"`
	Subtitles              Subtitles `json:"subtitles"`
}

// Subtitles represents the subtitles for a media file.
type Subtitles struct {
	URL string `json:"url"`
}

// Language represents a single language available on JW Broadcasting.
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// LanguagesResponse is the response from the languages API endpoint.
type LanguagesResponse struct {
	Languages []Language `json:"languages"`
}

// CategoryResponse is the response from the category API endpoint.
type CategoryResponse struct {
	Category struct {
		Key           string `json:"key"`
		Name          string `json:"name"`
		Subcategories []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"subcategories"`
		Media []struct {
			Title           string `json:"title"`
			Type            string `json:"type"`
			PrimaryCategory string `json:"primaryCategory"`
			FirstPublished  string `json:"firstPublished"`
			Files           []File `json:"files"`
		} `json:"media"`
	} `json:"category"`
}

// RootCategoriesResponse is the response from the root categories API endpoint.
type RootCategoriesResponse struct {
	Categories []struct {
		Key         string   `json:"key"`
		Type        string   `json:"type"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	} `json:"categories"`
}
