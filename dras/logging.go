package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// parseLevel maps a string log level to slog.Level.
// Recognized: DEBUG, INFO, WARN/WARNING, ERROR, FATAL (mapped to ERROR).
// Unknown or empty values default to INFO.
func parseLevel(level string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	case "FATAL":
		// slog has no FATAL level; map to ERROR. Callers that want
		// fatal-style behavior call fatal() which logs at ERROR + os.Exit(1).
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// newLogger constructs a *slog.Logger writing to w. The format argument
// (case-insensitive) selects the handler: "json" -> JSONHandler, anything
// else (including empty) -> TextHandler.
func newLogger(levelStr, format string, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(levelStr)}
	var h slog.Handler
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h)
}

// fatal logs at ERROR level (slog has no FATAL) and exits with code 1.
// Mirrors the printf-style ergonomics of the old logger.Fatal helper.
func fatal(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
