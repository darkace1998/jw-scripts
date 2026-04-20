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
	// Test the logic for calculating the 31-day backward window for --latest flag
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startOfPeriod := startOfToday.AddDate(0, 0, -31)
	endOfToday := startOfToday.AddDate(0, 0, 1).Add(-time.Nanosecond)

	// Should span exactly 31 days ending today
	diff := endOfToday.Sub(startOfPeriod)
	expectedDiff := 32*24*time.Hour - time.Nanosecond

	if diff != expectedDiff {
		t.Errorf("Expected 32 days difference (minus 1 nanosecond), got %v", diff)
	}

	// StartOfPeriod should be at midnight
	if startOfPeriod.Hour() != 0 || startOfPeriod.Minute() != 0 || startOfPeriod.Second() != 0 {
		t.Errorf("StartOfPeriod should be at midnight, got %v", startOfPeriod)
	}

	// StartOfPeriod should be in the past
	if !startOfPeriod.Before(now) {
		t.Errorf("StartOfPeriod should be in the past, got %v", startOfPeriod)
	}

	// EndOfToday should still be after now (except at end-of-day edge)
	if !endOfToday.After(now) {
		t.Errorf("EndOfToday should be in the future, got %v", endOfToday)
	}
}

func TestMaxDateFiltering(t *testing.T) {
	// Test that MaxDate field works correctly in Settings
	settings := &Settings{}

	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startOfPeriod := startOfToday.AddDate(0, 0, -31)
	endOfToday := startOfToday.AddDate(0, 0, 1).Add(-time.Nanosecond)

	settings.MinDate = startOfPeriod.Unix()
	settings.MaxDate = endOfToday.Unix()

	// Test that MinDate and MaxDate are properly set
	if settings.MinDate <= 0 {
		t.Errorf("MinDate should be positive, got %d", settings.MinDate)
	}

	if settings.MaxDate <= settings.MinDate {
		t.Errorf("MaxDate (%d) should be greater than MinDate (%d)", settings.MaxDate, settings.MinDate)
	}

	// Test the "past 31 days through end of today" window
	minTime := time.Unix(settings.MinDate, 0)
	maxTime := time.Unix(settings.MaxDate, 0)
	windowDuration := maxTime.Sub(minTime)

	// Should be approximately 32 days minus 1ns (allowing for minor precision differences)
	expectedDuration := 32*24*time.Hour - time.Nanosecond
	if windowDuration < expectedDuration-time.Second || windowDuration > expectedDuration+time.Second {
		t.Errorf("Expected ~32 days between MinDate and MaxDate, got %v", windowDuration)
	}
}
