package monitor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/image"
	"github.com/jacaudi/dras/internal/logger"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
	"github.com/jacaudi/dras/internal/renderer"
)

func TestFetchRadarImageNilService(t *testing.T) {
	m := New(radar.NewMockDataFetcher(), notify.NewMockNotifier(), nil, &config.Config{})

	got := m.fetchRadarImage("KATX", logger.WithField("station", "KATX"))
	if got != nil {
		t.Errorf("fetchRadarImage() = %v, want nil when image service is nil", got)
	}
}

func TestFetchRadarImageReturnsNilOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	imgSvc := image.New(image.Config{URLTemplate: server.URL + "/{station}.gif"})
	m := New(radar.NewMockDataFetcher(), notify.NewMockNotifier(), imgSvc, &config.Config{})

	got := m.fetchRadarImage("KATX", logger.WithField("station", "KATX"))
	if got != nil {
		t.Errorf("fetchRadarImage() = %v, want nil on HTTP failure", got)
	}
}

func TestAttachmentForChange(t *testing.T) {
	body := []byte("GIF89a-fake")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/gif")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	imgSvc := image.New(image.Config{URLTemplate: server.URL + "/{station}.gif"})
	m := New(radar.NewMockDataFetcher(), notify.NewMockNotifier(), imgSvc, &config.Config{})
	stationLogger := logger.WithField("station", "KATX")

	t.Run("returns nil when VCP did not change", func(t *testing.T) {
		fresh, err := imgSvc.Fetch("KATX")
		if err != nil {
			t.Fatalf("Fetch() error: %v", err)
		}
		got := m.attachmentForChange("KATX", false, fresh, stationLogger)
		if got != nil {
			t.Errorf("attachmentForChange() = %v, want nil when VCP unchanged", got)
		}
	})

	t.Run("returns attachment with fresh image when VCP changed", func(t *testing.T) {
		fresh, err := imgSvc.Fetch("KATX")
		if err != nil {
			t.Fatalf("Fetch() error: %v", err)
		}
		got := m.attachmentForChange("KATX", true, fresh, stationLogger)
		if got == nil {
			t.Fatal("attachmentForChange() = nil, want attachment")
		}
		if string(got.Data) != string(body) {
			t.Errorf("attachment data = %q, want %q", got.Data, body)
		}
		if got.ContentType != "image/gif" {
			t.Errorf("attachment content type = %q, want image/gif", got.ContentType)
		}
	})

	t.Run("falls back to cached image when fresh fetch failed", func(t *testing.T) {
		// imgSvc still has a cached image from the prior subtest. Pass nil as
		// the just-fetched image to simulate a failed download.
		got := m.attachmentForChange("KATX", true, nil, stationLogger)
		if got == nil {
			t.Fatal("attachmentForChange() = nil, want fallback to cached image")
		}
		if string(got.Data) != string(body) {
			t.Errorf("attachment data = %q, want cached %q", got.Data, body)
		}
	})

	t.Run("returns nil when no image is available", func(t *testing.T) {
		got := m.attachmentForChange("KUNKNOWN", true, nil, stationLogger)
		if got != nil {
			t.Errorf("attachmentForChange() = %v, want nil for unknown station with no cache", got)
		}
	})

	t.Run("returns nil when image service is disabled", func(t *testing.T) {
		mNoImg := New(radar.NewMockDataFetcher(), notify.NewMockNotifier(), nil, &config.Config{})
		fresh, err := imgSvc.Fetch("KATX")
		if err != nil {
			t.Fatalf("Fetch() error: %v", err)
		}
		got := mNoImg.attachmentForChange("KATX", true, fresh, stationLogger)
		if got != nil {
			t.Errorf("attachmentForChange() = %v, want nil when image service is disabled", got)
		}
	})
}

// TestVCPChangeDeliversAttachment is an end-to-end test that wires together a
// fake radar source, a real image service backed by httptest, and a mock
// notifier to verify that a VCP change between two polls triggers a
// notification with the just-downloaded radar image attached.
func TestVCPChangeDeliversAttachment(t *testing.T) {
	// Image server returns a unique body per request so we can tell which
	// image was attached.
	var imageRequests int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&imageRequests, 1)
		w.Header().Set("Content-Type", "image/gif")
		w.Write([]byte{'G', 'I', 'F', byte('0' + n)})
	}))
	defer server.Close()

	imgSvc := image.New(image.Config{
		URLTemplate: server.URL + "/{station}.gif",
		Retention:   time.Hour,
		UserAgent:   "dras/test",
	})

	radarMock := radar.NewMockDataFetcher()
	radarMock.SetResponse("KATX", &radar.Data{
		Name: "Seattle", VCP: "R31", Mode: "Clear Air",
		Status: "Online", OperabilityStatus: "Normal",
		PowerSource: "Utility", GenState: "Off",
	})

	notifyMock := notify.NewMockNotifier()

	cfg := &config.Config{
		DryRun:        false,
		CheckInterval: time.Minute,
		AlertConfig:   radar.AlertConfig{VCP: true},
	}

	m := New(radarMock, notifyMock, imgSvc, cfg)
	ctx := context.Background()

	// First poll seeds the cache and triggers the startup notification.
	if err := m.processStation(ctx, "KATX"); err != nil {
		t.Fatalf("first processStation() error: %v", err)
	}

	startupNotifs := notifyMock.GetNotifications()
	if len(startupNotifs) != 1 || startupNotifs[0].Title != "DRAS Startup" {
		t.Fatalf("expected startup notification, got %+v", startupNotifs)
	}
	// Startup notification carries the freshly-fetched radar image so the
	// user has visual context the moment the monitor comes online.
	if startupNotifs[0].Attachment == nil {
		t.Fatal("startup notification has no attachment")
	}
	if string(startupNotifs[0].Attachment.Data) != string([]byte{'G', 'I', 'F', '1'}) {
		t.Errorf("startup attachment data = %q, want %q (first image)", startupNotifs[0].Attachment.Data, []byte{'G', 'I', 'F', '1'})
	}
	if startupNotifs[0].Attachment.ContentType != "image/gif" {
		t.Errorf("startup attachment content type = %q, want image/gif", startupNotifs[0].Attachment.ContentType)
	}
	if got := atomic.LoadInt64(&imageRequests); got != 1 {
		t.Errorf("expected 1 image request after first poll, got %d", got)
	}

	// Second poll: VCP changes from R31 (Clear Air) to R12 (Precipitation).
	radarMock.SetResponse("KATX", &radar.Data{
		Name: "Seattle", VCP: "R12", Mode: "Precipitation",
		Status: "Online", OperabilityStatus: "Normal",
		PowerSource: "Utility", GenState: "Off",
	})

	if err := m.processStation(ctx, "KATX"); err != nil {
		t.Fatalf("second processStation() error: %v", err)
	}

	notifs := notifyMock.GetNotifications()
	if len(notifs) != 2 {
		t.Fatalf("expected 2 notifications, got %d (%+v)", len(notifs), notifs)
	}

	change := notifs[1]
	if change.Title != "KATX Update" {
		t.Errorf("change notification title = %q, want %q", change.Title, "KATX Update")
	}
	if change.Attachment == nil {
		t.Fatal("change notification has no attachment")
	}
	// Second image request returns "GIF2" — the freshest poll.
	want := []byte{'G', 'I', 'F', '2'}
	if string(change.Attachment.Data) != string(want) {
		t.Errorf("attachment data = %q, want %q (freshest image)", change.Attachment.Data, want)
	}
	if change.Attachment.ContentType != "image/gif" {
		t.Errorf("attachment content type = %q, want image/gif", change.Attachment.ContentType)
	}
	if got := atomic.LoadInt64(&imageRequests); got != 2 {
		t.Errorf("expected 2 image requests total, got %d", got)
	}

	// History should contain both polls' images.
	history := imgSvc.History("KATX")
	if len(history) != 2 {
		t.Errorf("expected 2 images in history, got %d", len(history))
	}
}

// TestNonVCPChangeOmitsAttachment verifies that changes which are not VCP
// changes (e.g. power source) do not attach the radar image, matching the
// user-facing scope of the feature.
func TestNonVCPChangeOmitsAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/gif")
		w.Write([]byte("img"))
	}))
	defer server.Close()

	imgSvc := image.New(image.Config{URLTemplate: server.URL + "/{station}.gif"})
	radarMock := radar.NewMockDataFetcher()
	radarMock.SetResponse("KATX", &radar.Data{
		Name: "Seattle", VCP: "R31", Mode: "Clear Air",
		Status: "Online", OperabilityStatus: "Normal",
		PowerSource: "Utility", GenState: "Off",
	})

	notifyMock := notify.NewMockNotifier()
	cfg := &config.Config{
		DryRun:        false,
		CheckInterval: time.Minute,
		AlertConfig:   radar.AlertConfig{VCP: true, PowerSource: true},
	}

	m := New(radarMock, notifyMock, imgSvc, cfg)
	ctx := context.Background()

	if err := m.processStation(ctx, "KATX"); err != nil {
		t.Fatalf("first processStation() error: %v", err)
	}

	// Change only the power source, not the VCP.
	radarMock.SetResponse("KATX", &radar.Data{
		Name: "Seattle", VCP: "R31", Mode: "Clear Air",
		Status: "Online", OperabilityStatus: "Normal",
		PowerSource: "Generator", GenState: "On",
	})

	if err := m.processStation(ctx, "KATX"); err != nil {
		t.Fatalf("second processStation() error: %v", err)
	}

	notifs := notifyMock.GetNotifications()
	if len(notifs) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifs))
	}
	change := notifs[1]
	if change.Attachment != nil {
		t.Errorf("non-VCP change should not have attachment, got %+v", change.Attachment)
	}
}

// TestRendererModeDeliversAttachment exercises the full happy path with the
// renderer.Client image source. A renderer stub returns a unique PNG per
// request so we can assert which body was attached.
func TestRendererModeDeliversAttachment(t *testing.T) {
	pngs := []string{"GIF1", "GIF2"} // intentionally not real PNGs; the contract is "bytes"
	var imageRequests int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&imageRequests, 1)
		body := pngs[n-1]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"image": base64.StdEncoding.EncodeToString([]byte(body)),
			"metadata": map[string]any{
				"station":          "KATX",
				"product":          "base_reflectivity",
				"scan_time":        "2026-04-26T15:32:00Z",
				"elevation_deg":    0.5,
				"vcp":              215,
				"renderer_version": "v0.0.0-test",
			},
		})
	}))
	defer srv.Close()

	rendererClient := renderer.New(renderer.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})

	radarMock := radar.NewMockDataFetcher()
	radarMock.SetResponse("KATX", &radar.Data{
		Name: "Seattle", VCP: "R31", Mode: "Clear Air",
		Status: "Online", OperabilityStatus: "Normal",
		PowerSource: "Utility", GenState: "Off",
	})

	notifyMock := notify.NewMockNotifier()
	cfg := &config.Config{
		DryRun:        false,
		CheckInterval: time.Minute,
		AlertConfig:   radar.AlertConfig{VCP: true},
	}

	m := New(radarMock, notifyMock, rendererClient, cfg)
	ctx := context.Background()

	// First poll: startup notification with the freshly-rendered image.
	if err := m.processStation(ctx, "KATX"); err != nil {
		t.Fatalf("first processStation: %v", err)
	}
	startup := notifyMock.GetNotifications()
	if len(startup) != 1 || startup[0].Title != "DRAS Startup" {
		t.Fatalf("expected startup notification, got %+v", startup)
	}
	if startup[0].Attachment == nil || string(startup[0].Attachment.Data) != "GIF1" {
		t.Fatalf("startup attachment = %+v, want body 'GIF1'", startup[0].Attachment)
	}

	// VCP change.
	radarMock.SetResponse("KATX", &radar.Data{
		Name: "Seattle", VCP: "R12", Mode: "Precipitation",
		Status: "Online", OperabilityStatus: "Normal",
		PowerSource: "Utility", GenState: "Off",
	})
	if err := m.processStation(ctx, "KATX"); err != nil {
		t.Fatalf("second processStation: %v", err)
	}
	all := notifyMock.GetNotifications()
	if len(all) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(all))
	}
	change := all[1]
	if change.Attachment == nil || string(change.Attachment.Data) != "GIF2" {
		t.Fatalf("change attachment = %+v, want body 'GIF2'", change.Attachment)
	}
	if got := atomic.LoadInt64(&imageRequests); got != 2 {
		t.Errorf("expected 2 renderer requests, got %d", got)
	}
}
