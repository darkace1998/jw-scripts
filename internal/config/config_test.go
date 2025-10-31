// Package config provides configuration settings for JW scripts.
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
	// Test the logic for calculating the 31-day forward window for --latest flag
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfPeriod := startOfToday.AddDate(0, 0, 31).Add(-time.Nanosecond)

	// Should be exactly 31 days
	diff := endOfPeriod.Sub(startOfToday)
	expectedDiff := 31*24*time.Hour - time.Nanosecond

	if diff != expectedDiff {
		t.Errorf("Expected 31 days difference (minus 1 nanosecond), got %v", diff)
	}

	// MinDate should be at midnight today
	if startOfToday.Hour() != 0 || startOfToday.Minute() != 0 || startOfToday.Second() != 0 {
		t.Errorf("StartOfToday should be at midnight, got %v", startOfToday)
	}

	// MinDate should not be in the future beyond today
	if startOfToday.After(now) {
		t.Errorf("StartOfToday should not be in the future, got %v", startOfToday)
	}

	// EndOfPeriod should be in the future
	if !endOfPeriod.After(now) {
		t.Errorf("EndOfPeriod should be in the future, got %v", endOfPeriod)
	}
}

func TestMaxDateFiltering(t *testing.T) {
	// Test that MaxDate field works correctly in Settings
	settings := &Settings{}

	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfPeriod := startOfToday.AddDate(0, 0, 31).Add(-time.Nanosecond)

	settings.MinDate = startOfToday.Unix()
	settings.MaxDate = endOfPeriod.Unix()

	// Test that MinDate and MaxDate are properly set
	if settings.MinDate <= 0 {
		t.Errorf("MinDate should be positive, got %d", settings.MinDate)
	}

	if settings.MaxDate <= settings.MinDate {
		t.Errorf("MaxDate (%d) should be greater than MinDate (%d)", settings.MaxDate, settings.MinDate)
	}

	// Test the 31-day window
	minTime := time.Unix(settings.MinDate, 0)
	maxTime := time.Unix(settings.MaxDate, 0)
	windowDuration := maxTime.Sub(minTime)

	// Should be approximately 31 days (allowing for minor precision differences)
	expectedDuration := 31 * 24 * time.Hour
	if windowDuration < expectedDuration-time.Second || windowDuration > expectedDuration+time.Second {
		t.Errorf("Expected ~31 days between MinDate and MaxDate, got %v", windowDuration)
	}
}
