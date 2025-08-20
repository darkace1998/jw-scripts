package config

// Settings holds all the application settings, primarily from command-line flags.
type Settings struct {
	Quiet             int
	ListLanguages     bool
	PositionalArgs    []string
	WorkDir           string
	SubDir            string
	OutputFilename    string
	Command           []string
	Lang              string
	Quality           int
	HardSubtitles     bool
	MinDate           int64
	IncludeCategories []string
	ExcludeCategories []string
	FilterCategories  []string
	PrintCategory     string
	ListCategories    bool // flag to indicate --category with no args should list categories
	Latest            bool
	KeepFree          int64
	Warning           bool
	ImportDir         string
	Download          bool
	DownloadSubtitles bool
	FriendlyFilenames bool
	RateLimit         float64
	Checksums         bool
	OverwriteBad      bool
	Append            bool
	CleanAllSymlinks  bool
	Update            bool
	Mode              string
	SafeFilenames     bool
	Sort              string
}
