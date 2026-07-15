package torrent

import (
	"path/filepath"
	"testing"

	"golang.org/x/time/rate"
)

func TestSetSpeedLimitsPersistsAndApplies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "speed_limits.json")
	client := &Client{
		speedLimitsFile: path,
		downloadLimiter: newRateLimiter(0),
		uploadLimiter:   newRateLimiter(0),
	}
	want := SpeedLimits{Download: 3 * 1024 * 1024, Upload: 512 * 1024}

	if err := client.SetSpeedLimits(want); err != nil {
		t.Fatalf("set speed limits: %v", err)
	}
	if got := client.SpeedLimits(); got != want {
		t.Fatalf("limits mismatch: got %+v, want %+v", got, want)
	}
	if got := client.downloadLimiter.Limit(); got != rate.Limit(want.Download) {
		t.Fatalf("download limiter mismatch: got %v", got)
	}
	if got := client.uploadLimiter.Limit(); got != rate.Limit(want.Upload) {
		t.Fatalf("upload limiter mismatch: got %v", got)
	}
	if got, err := loadSpeedLimits(path, SpeedLimits{}); err != nil || got != want {
		t.Fatalf("persisted limits mismatch: got %+v, err %v", got, err)
	}
}

func TestSetSpeedLimitsSupportsUnlimited(t *testing.T) {
	client := &Client{
		speedLimitsFile: filepath.Join(t.TempDir(), "speed_limits.json"),
		downloadLimiter: newRateLimiter(1024),
		uploadLimiter:   newRateLimiter(1024),
	}

	if err := client.SetSpeedLimits(SpeedLimits{}); err != nil {
		t.Fatalf("disable speed limits: %v", err)
	}
	if client.downloadLimiter.Limit() != rate.Inf || client.uploadLimiter.Limit() != rate.Inf {
		t.Fatal("zero limits must configure unlimited transfer")
	}
}
