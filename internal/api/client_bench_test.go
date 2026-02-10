package api

import "testing"

func BenchmarkGetBestVideo(b *testing.B) {
	files := []File{
		{ProgressiveDownloadURL: "1080p.mp4", Label: "1080p"},
		{ProgressiveDownloadURL: "720p.mp4", Label: "720p"},
		{ProgressiveDownloadURL: "480p.mp4", Label: "480p"},
		{ProgressiveDownloadURL: "360p.mp4", Label: "360p"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getBestVideo(files, 720, false)
	}
}

func BenchmarkFormatFilename(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatFilename("My:Awesome/Video<Name>.mp4", true)
	}
}

func BenchmarkMakeUniqueFilename(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usedFilenames := map[string]bool{"video.mp4": true}
		makeUniqueFilename("video.mp4", usedFilenames)
	}
}
