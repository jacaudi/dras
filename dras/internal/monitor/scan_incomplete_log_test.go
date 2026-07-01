package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/image"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
)

// Integration coverage for issue #122: fetchRadarImage must log the benign
// "scan not yet complete" (image.ErrScanIncomplete) case at INFO and every
// other fetch failure at WARN, in both cases returning a nil image so the
// notification proceeds text-only.

// fakeSource is a minimal image.Source that returns a preconfigured error.
type fakeSource struct{ err error }

func (f fakeSource) Fetch(_ context.Context, _ string) (*image.Image, error) { return nil, f.err }
func (f fakeSource) Latest(_ string) (*image.Image, bool)                    { return nil, false }

// recordingHandler captures the level and message of each emitted record.
type recordingHandler struct{ records *[]slog.Record }

func (h recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h recordingHandler) Handle(_ context.Context, r slog.Record) error {
	*h.records = append(*h.records, r)
	return nil
}
func (h recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h recordingHandler) WithGroup(string) slog.Handler      { return h }

func newMonitorForLogTest() *Monitor {
	return New(radar.NewMockDataFetcher(), notify.NewMockNotifier(), nil, &config.Config{})
}

func TestFetchRadarImage_ScanIncomplete_LogsInfo(t *testing.T) {
	var records []slog.Record
	logger := slog.New(recordingHandler{records: &records})

	m := newMonitorForLogTest()
	// Real production shape: the renderer client wraps the sentinel with
	// context around it (fmt.Errorf("...: %w", ..., image.ErrScanIncomplete)).
	m.imageService = fakeSource{err: fmt.Errorf("renderer returned 502: decode_failed (No MSG31 records): %w", image.ErrScanIncomplete)}

	img := m.fetchRadarImage(context.Background(), "KATX", logger)
	if img != nil {
		t.Errorf("expected nil image, got %+v", img)
	}

	if len(records) != 1 {
		t.Fatalf("expected exactly 1 log record, got %d", len(records))
	}
	if records[0].Level != slog.LevelInfo {
		t.Errorf("level = %v, want INFO for benign scan-incomplete skip", records[0].Level)
	}
	if !strings.Contains(records[0].Message, "not yet complete") {
		t.Errorf("message = %q, want to mention 'not yet complete'", records[0].Message)
	}
}

func TestFetchRadarImage_GenericError_LogsWarn(t *testing.T) {
	var records []slog.Record
	logger := slog.New(recordingHandler{records: &records})

	m := newMonitorForLogTest()
	m.imageService = fakeSource{err: fmt.Errorf("renderer returned 500: internal (S3 download failed)")}

	img := m.fetchRadarImage(context.Background(), "KATX", logger)
	if img != nil {
		t.Errorf("expected nil image, got %+v", img)
	}

	if len(records) != 1 {
		t.Fatalf("expected exactly 1 log record, got %d", len(records))
	}
	if records[0].Level != slog.LevelWarn {
		t.Errorf("level = %v, want WARN for a genuine fetch failure", records[0].Level)
	}
	if !strings.Contains(records[0].Message, "Failed to fetch radar image") {
		t.Errorf("message = %q, want the WARN failure text", records[0].Message)
	}
}
