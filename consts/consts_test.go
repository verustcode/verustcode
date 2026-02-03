package consts

import (
	"sync"
	"testing"
	"time"
)

func TestServiceName(t *testing.T) {
	if ServiceName != "verustcode" {
		t.Errorf("ServiceName = %q, want %q", ServiceName, "verustcode")
	}
}

func TestOutputFormats(t *testing.T) {
	if OutputFormatMarkdown != "markdown" {
		t.Errorf("OutputFormatMarkdown = %q, want %q", OutputFormatMarkdown, "markdown")
	}
	if OutputFormatJSON != "json" {
		t.Errorf("OutputFormatJSON = %q, want %q", OutputFormatJSON, "json")
	}
}

func TestProjectInfo(t *testing.T) {
	if ProjectName != "VerustCode" {
		t.Errorf("ProjectName = %q, want %q", ProjectName, "VerustCode")
	}
	if ProjectURL != "https://github.com/verustcode/verustcode" {
		t.Errorf("ProjectURL = %q, want %q", ProjectURL, "https://github.com/verustcode/verustcode")
	}
}

func TestSetStartedAt(t *testing.T) {
	// Reset state for testing
	startedAt = time.Time{}
	startedOnce = sync.Once{}

	now := time.Now()
	SetStartedAt(now)

	got := GetStartedAt()
	if !got.Equal(now) {
		t.Errorf("GetStartedAt() = %v, want %v", got, now)
	}

	// Test that SetStartedAt can only be called once
	anotherTime := now.Add(time.Hour)
	SetStartedAt(anotherTime)
	got = GetStartedAt()
	if !got.Equal(now) {
		t.Errorf("GetStartedAt() after second call = %v, want %v (should not change)", got, now)
	}
}

func TestGetUptime(t *testing.T) {
	// Reset state
	startedAt = time.Time{}
	startedOnce = sync.Once{}

	// Test zero time
	uptime := GetUptime()
	if uptime != 0 {
		t.Errorf("GetUptime() with zero time = %v, want 0", uptime)
	}

	// Test with set time
	now := time.Now()
	SetStartedAt(now)
	uptime = GetUptime()
	if uptime < 0 {
		t.Errorf("GetUptime() = %v, want non-negative", uptime)
	}
	if uptime > time.Second {
		t.Errorf("GetUptime() = %v, want less than 1 second", uptime)
	}
}
