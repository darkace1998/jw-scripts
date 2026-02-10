package downloader

import (
	"io"
	"strings"
	"testing"
)

func BenchmarkThrottledReaderUnlimited(b *testing.B) {
	data := strings.Repeat("A", 64*1024)
	buf := make([]byte, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := newThrottledReader(strings.NewReader(data), 0)
		for {
			_, err := reader.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
