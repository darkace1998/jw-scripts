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
	Date             int64
	Duration         int
	MD5              string
	Name             string
	Size             int64
	SubtitleURL      string
	URL              string
	Filename         string
	FriendlyName     string
	SubtitleFilename string
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
			Files           []struct {
				ProgressiveDownloadURL string `json:"progressiveDownloadURL"`
				Checksum               string `json:"checksum"`
				Filesize               int64  `json:"filesize"`
				Duration               int    `json:"duration"`
				Label                  string `json:"label"`
				Subtitled              bool   `json:"subtitled"`
				Subtitles              struct {
					URL string `json:"url"`
				} `json:"subtitles"`
			} `json:"files"`
		} `json:"media"`
	} `json:"category"`
}
