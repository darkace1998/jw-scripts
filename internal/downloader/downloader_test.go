package downloader

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestThrottledReader(t *testing.T) {
	tests := []struct {
		name      string
		dataSize  int
		rateLimit float64 // MB/s
		tolerance float64 // acceptable rate variance as percentage
	}{
		{
			name:      "No rate limit",
			dataSize:  64 * 1024, // 64KB
			rateLimit: 0,          // No limit
			tolerance: 0,          // Not applicable for unlimited
		},
		{
			name:      "Small rate limit",
			dataSize:  32 * 1024, // 32KB
			rateLimit: 0.1,       // 0.1 MB/s
			tolerance: 20,        // 20% tolerance
		},
		{
			name:      "Medium rate limit",
			dataSize:  64 * 1024, // 64KB
			rateLimit: 0.5,       // 0.5 MB/s
			tolerance: 20,        // 20% tolerance
		},
		{
			name:      "Standard rate limit",
			dataSize:  128 * 1024, // 128KB
			rateLimit: 1.0,        // 1.0 MB/s
			tolerance: 20,         // 20% tolerance
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := strings.Repeat("A", tt.dataSize)
			
			var reader io.Reader = strings.NewReader(data)
			if tt.rateLimit > 0 {
				reader = newThrottledReader(reader, tt.rateLimit)
			}

			start := time.Now()
			buf := make([]byte, 4096)
			totalRead := 0

			for {
				n, err := reader.Read(buf)
				totalRead += n
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			elapsed := time.Since(start)
			
			if totalRead != tt.dataSize {
				t.Errorf("Expected to read %d bytes, got %d", tt.dataSize, totalRead)
			}

			if tt.rateLimit > 0 {
				actualRate := float64(totalRead) / elapsed.Seconds() / 1024 / 1024 // MB/s
				efficiency := (actualRate / tt.rateLimit) * 100
				
				if efficiency > 100+tt.tolerance {
					t.Errorf("Rate too high: expected ~%.2f MB/s, got %.2f MB/s (%.1f%% efficiency)", 
						tt.rateLimit, actualRate, efficiency)
				}
				if efficiency < 100-tt.tolerance {
					t.Errorf("Rate too low: expected ~%.2f MB/s, got %.2f MB/s (%.1f%% efficiency)", 
						tt.rateLimit, actualRate, efficiency)
				}
				
				t.Logf("Rate limit test passed: %.2f MB/s target, %.2f MB/s actual (%.1f%% efficiency)", 
					tt.rateLimit, actualRate, efficiency)
			} else {
				// For unlimited, just check it completed quickly
				if elapsed > unlimitedRateMaxDuration {
					t.Errorf("Unlimited rate took too long: %v", elapsed)
				}
				t.Logf("Unlimited rate test passed: completed in %v", elapsed)
			}
		})
	}
}

func TestThrottledReaderZeroRateLimit(t *testing.T) {
	data := "Hello, World!"
	reader := newThrottledReader(strings.NewReader(data), 0) // No rate limit

	buf := make([]byte, len(data))
	start := time.Now()
	n, err := reader.Read(buf)
	elapsed := time.Since(start)

	if err != nil && err != io.EOF {
		t.Fatalf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to read %d bytes, got %d", len(data), n)
	}

	if string(buf[:n]) != data {
		t.Errorf("Expected data %q, got %q", data, string(buf[:n]))
	}

	// Should be very fast with no rate limit
	if elapsed > zeroRateLimitMaxDuration {
		t.Errorf("Read with no rate limit took too long: %v", elapsed)
	}
}

func TestThrottledReaderSmallReads(t *testing.T) {
	data := strings.Repeat("B", 1024) // 1KB
	reader := newThrottledReader(strings.NewReader(data), 0.1) // Very slow: 0.1 MB/s

	start := time.Now()
	buf := make([]byte, 256) // Read in small 256-byte chunks
	totalRead := 0

	for {
		n, err := reader.Read(buf)
		totalRead += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	elapsed := time.Since(start)
	actualRate := float64(totalRead) / elapsed.Seconds() / 1024 / 1024 // MB/s

	if totalRead != len(data) {
		t.Errorf("Expected to read %d bytes, got %d", totalRead, len(data))
	}

	// Should be close to 0.1 MB/s
	if actualRate > 0.15 || actualRate < 0.05 {
		t.Errorf("Rate should be ~0.1 MB/s, got %.3f MB/s", actualRate)
	}
	
	t.Logf("Small reads test: %.3f MB/s (target 0.1 MB/s)", actualRate)
}

func TestThrottledReaderRestart(t *testing.T) {
	// Test that a new throttledReader starts fresh each time
	data := strings.Repeat("C", 512) // 512 bytes
	
	// First read
	reader1 := newThrottledReader(strings.NewReader(data), 0.1) // 0.1 MB/s
	buf := make([]byte, 256)
	
	start := time.Now()
	n1, err := reader1.Read(buf)
	if err != nil {
		t.Fatalf("First read error: %v", err)
	}
	
	// Second read from the same reader (should continue throttling from where it left off)
	n2, err := reader1.Read(buf[n1:])
	elapsed := time.Since(start)
	
	if err != nil && err != io.EOF {
		t.Fatalf("Second read error: %v", err)
	}
	
	totalRead := n1 + n2
	actualRate := float64(totalRead) / elapsed.Seconds() / 1024 / 1024
	
	// Should maintain the rate limit across multiple reads
	if actualRate > 0.15 || actualRate < 0.05 {
		t.Errorf("Rate across multiple reads should be ~0.1 MB/s, got %.3f MB/s", actualRate)
	}
	
	t.Logf("Multi-read test: %.3f MB/s (target 0.1 MB/s)", actualRate)
}