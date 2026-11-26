package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestContains(t *testing.T) {
	values := []string{"today", "7d", "30d", "year", "all"}
	for _, v := range values {
		if !contains(values, v) {
			t.Fatalf("expected contains(%q) to be true", v)
		}
	}
	if contains(values, "nope") {
		t.Fatal("expected unknown value to be false")
	}
	if contains(nil, "today") {
		t.Fatal("expected nil slice to be false")
	}
}

func TestNumber(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  float64
	}{
		{"float64", float64(1.5), 1.5},
		{"float32", float32(2.5), 2.5},
		{"int", int(3), 3},
		{"int64", int64(4), 4},
		{"json.Number", json.Number("12.75"), 12.75},
		{"string", "nope", 0},
		{"nil", nil, 0},
	}
	for _, tt := range tests {
		if got := number(tt.value); got != tt.want {
			t.Fatalf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}

func TestFirstString(t *testing.T) {
	values := map[string]any{"name": "alice", "userName": "alice_u", "empty": ""}
	got, ok := firstString(values, "missing", "empty", "userName", "name")
	if !ok || got != "alice_u" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if _, ok := firstString(values, "missing"); ok {
		t.Fatal("expected missing key to return false")
	}
}

func TestUsageTokens(t *testing.T) {
	var tokens UsageTokens
	tokens.add(UsageTokens{Input: 1, Output: 2, CacheRead: 3, CacheWrite: 4})
	tokens.add(UsageTokens{Input: 10, Output: 20, CacheRead: 30, CacheWrite: 40})
	if tokens.total() != 110 {
		t.Fatalf("total got %d want 110", tokens.total())
	}
}

func TestPercent(t *testing.T) {
	if got := percent(50, 200); got != 25 {
		t.Fatalf("got %v want 25", got)
	}
	if got := percent(10, 0); got != 0 {
		t.Fatalf("got %v want 0 for zero total", got)
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		value string
		want  uint64
	}{
		{"1024", 1024},
		{"1K", 1024},
		{"1M", 1024 * 1024},
		{"2G", 2 * 1024 * 1024 * 1024},
		{"1.5G", uint64(1.5 * 1024 * 1024 * 1024)},
	}
	for _, tt := range tests {
		if got := parseSize(tt.value); got != tt.want {
			t.Fatalf("%s: got %d want %d", tt.value, got, tt.want)
		}
	}
}

func TestRangeCutoff(t *testing.T) {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	cutoff, ok := rangeCutoff("today")
	if !ok || !cutoff.Equal(startOfToday) {
		t.Fatalf("today cutoff got %v ok=%v", cutoff, ok)
	}

	cutoff, ok = rangeCutoff("7d")
	want7d := startOfToday.AddDate(0, 0, -6)
	if !ok || !cutoff.Equal(want7d) {
		t.Fatalf("7d cutoff got %v ok=%v", cutoff, ok)
	}

	cutoff, ok = rangeCutoff("30d")
	want30d := startOfToday.AddDate(0, 0, -29)
	if !ok || !cutoff.Equal(want30d) {
		t.Fatalf("30d cutoff got %v ok=%v", cutoff, ok)
	}

	cutoff, ok = rangeCutoff("year")
	wantYear := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
	if !ok || !cutoff.Equal(wantYear) {
		t.Fatalf("year cutoff got %v ok=%v", cutoff, ok)
	}

	if _, ok := rangeCutoff("all"); ok {
		t.Fatal("all should not be bounded")
	}
}

func TestUsageWindow(t *testing.T) {
	window := usageWindow(map[string]any{
		"used":      12.5,
		"cap":       50.0,
		"remaining": 37.5,
		"percent":   25.0,
		"resetsAt":  "2026-11-26T12:00:00Z",
	})
	if !window.Available || window.Used != 12.5 || window.Cap != 50 || window.Remaining != 37.5 || window.Percent != 25 || window.ResetsAt != "2026-11-26T12:00:00Z" {
		t.Fatalf("unexpected window: %+v", window)
	}

	if got := usageWindow(nil); got.Available {
		t.Fatalf("nil window should be unavailable: %+v", got)
	}
}

func TestParseOpenCodeUsageLimits(t *testing.T) {
	input := []byte(`{"sessions":1,"tokens":{"input":10,"output":20,"reasoning":0,"cache":{"read":0,"write":0}},"limits":{"requests":{"used":5,"cap":100}}}`)
	usage, err := parseOpenCodeUsage(input)
	if err != nil {
		t.Fatalf("parseOpenCodeUsage: %v", err)
	}
	if usage.Sessions != 1 {
		t.Fatalf("sessions got %d", usage.Sessions)
	}
	if usage.Limits == nil || usage.Limits["requests"] == nil {
		t.Fatalf("expected limits to be preserved: %+v", usage.Limits)
	}
}

func TestAuth(t *testing.T) {
	called := false
	handler := auth("secret", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/?token=wrong", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token status got %d", rec.Code)
	}
	if called {
		t.Fatal("handler should not be called for wrong token")
	}

	called = false
	req = httptest.NewRequest(http.MethodGet, "/?token=secret", nil)
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNoContent || !called {
		t.Fatalf("right token status=%d called=%v", rec.Code, called)
	}
}

func TestFaviconHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	faviconHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("content-type got %q", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected favicon body")
	}
}

func TestDashboardHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	dashboardHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("content-type got %q", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected dashboard body")
	}
}
