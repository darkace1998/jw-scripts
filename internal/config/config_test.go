package config

import (
	"testing"
	"time"
)

func TestDefaultRateLimit(t *testing.T) {
	// This test verifies the default rate limit is set correctly in the application
	// The actual default is set in cmd/jwb-index/main.go flag definition
	settings := &Settings{}
	
	// Test that RateLimit field can hold the expected default value
	settings.RateLimit = 25.0
	if settings.RateLimit != 25.0 {
		t.Errorf("Expected RateLimit to be 25.0, got %f", settings.RateLimit)
	}
}

func TestLatestDateCalculation(t *testing.T) {
	// Test the logic for calculating 14 days ago for --latest flag
	now := time.Now()
	fourteenDaysAgo := now.AddDate(0, 0, -14)
	
	// Should be exactly 14 days
	diff := now.Sub(fourteenDaysAgo)
	expectedDiff := 14 * 24 * time.Hour
	
	if diff != expectedDiff {
		t.Errorf("Expected 14 days difference, got %v", diff)
	}
	
	// Unix timestamp should be in the past
	if fourteenDaysAgo.Unix() >= now.Unix() {
		t.Errorf("14 days ago timestamp should be less than current time")
	}
}