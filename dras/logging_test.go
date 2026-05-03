package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"fatal", slog.LevelError}, // FATAL maps to ERROR
		{"FATAL", slog.LevelError},
		{"", slog.LevelInfo},        // default
		{"garbage", slog.LevelInfo}, // default
		{"  info  ", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := parseLevel(tt.in)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestNewLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	lg := newLogger("debug", "text", &buf)
	lg.Info("hello world", "k", "v")

	out := buf.String()
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("expected level=INFO in text output, got %q", out)
	}
	if !strings.Contains(out, `msg="hello world"`) {
		t.Errorf("expected msg=\"hello world\" in text output, got %q", out)
	}
	if !strings.Contains(out, "k=v") {
		t.Errorf("expected k=v in text output, got %q", out)
	}
}

func TestNewLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	lg := newLogger("debug", "json", &buf)
	lg.Info("hello world", "k", "v")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("expected valid JSON output, got %q (err: %v)", buf.String(), err)
	}
	if rec["msg"] != "hello world" {
		t.Errorf("expected msg=\"hello world\", got %v", rec["msg"])
	}
	if rec["level"] != "INFO" {
		t.Errorf("expected level=INFO, got %v", rec["level"])
	}
	if rec["k"] != "v" {
		t.Errorf("expected k=v in JSON output, got %v", rec["k"])
	}
}

func TestNewLogger_DefaultsToText(t *testing.T) {
	var buf bytes.Buffer
	lg := newLogger("info", "", &buf)
	lg.Info("msg")

	out := buf.String()
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("expected text format by default, got %q", out)
	}
	// JSON output would start with '{', text would not.
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("expected text output (no JSON), got %q", out)
	}
}

func TestNewLogger_LevelFilters(t *testing.T) {
	var buf bytes.Buffer
	lg := newLogger("warn", "text", &buf)
	lg.Debug("debug-msg")
	lg.Info("info-msg")
	if buf.Len() != 0 {
		t.Errorf("expected debug/info to be filtered at level=warn, got %q", buf.String())
	}
	lg.Warn("warn-msg")
	if !strings.Contains(buf.String(), "warn-msg") {
		t.Errorf("expected warn-msg to be emitted, got %q", buf.String())
	}
}
