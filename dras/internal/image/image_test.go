package image

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestURLForUsesTemplate(t *testing.T) {
	svc := New(Config{URLTemplate: "https://example.com/{station}/img.gif"})
	got := svc.URLFor("KRAX")
	want := "https://example.com/KRAX/img.gif"
	if got != want {
		t.Errorf("URLFor() = %q, want %q", got, want)
	}
}

func TestURLForDefaultTemplate(t *testing.T) {
	svc := New(Config{})
	got := svc.URLFor("KRAX")
	if !strings.Contains(got, "KRAX") {
		t.Errorf("URLFor() = %q, expected to contain KRAX", got)
	}
	if !strings.Contains(got, "radar.weather.gov") {
		t.Errorf("URLFor() = %q, expected default NWS host", got)
	}
}

func TestRetentionDefault(t *testing.T) {
	svc := New(Config{})
	if svc.Retention() != DefaultRetention {
		t.Errorf("Retention() = %v, want %v", svc.Retention(), DefaultRetention)
	}
}

func TestFetchSendsUserAgentAndStores(t *testing.T) {
	body := []byte("GIF89a-fake-image-bytes")
	const ua = "dras/test (+https://github.com/jacaudi/dras)"

	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "image/gif")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	svc := New(Config{
		URLTemplate: server.URL + "/{station}.gif",
		UserAgent:   ua,
	})

	img, err := svc.Fetch("KATX")
	if err != nil {
		t.Fatalf("Fetch() returned error: %v", err)
	}
	if img == nil {
		t.Fatal("Fetch() returned nil image")
	}
	if receivedUA != ua {
		t.Errorf("server saw User-Agent %q, want %q", receivedUA, ua)
	}
	if string(img.Data) != string(body) {
		t.Errorf("Fetch() data = %q, want %q", img.Data, body)
	}
	if img.ContentType != "image/gif" {
		t.Errorf("Fetch() content type = %q, want image/gif", img.ContentType)
	}
	if !strings.HasPrefix(img.Filename, "KATX-") || !strings.HasSuffix(img.Filename, ".gif") {
		t.Errorf("Fetch() filename = %q, want KATX-<ts>.gif", img.Filename)
	}

	cached, ok := svc.Latest("KATX")
	if !ok {
		t.Fatal("Latest() returned no cached image after Fetch()")
	}
	if string(cached.Data) != string(body) {
		t.Errorf("Latest() data = %q, want %q", cached.Data, body)
	}

	history := svc.History("KATX")
	if len(history) != 1 {
		t.Errorf("History() length = %d, want 1", len(history))
	}
}

func TestFetchAppendsHistoryAndPrunes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/gif")
		_, _ = w.Write([]byte("frame"))
	}))
	defer server.Close()

	svc := New(Config{
		URLTemplate: server.URL + "/{station}.gif",
		Retention:   2 * time.Hour,
	})

	// Seed three images, two of which fall outside the retention window.
	now := time.Now()
	svc.history["KATX"] = []*Image{
		{StationID: "KATX", FetchedAt: now.Add(-3 * time.Hour), Data: []byte("old1")},
		{StationID: "KATX", FetchedAt: now.Add(-150 * time.Minute), Data: []byte("old2")},
		{StationID: "KATX", FetchedAt: now.Add(-30 * time.Minute), Data: []byte("recent")},
	}

	if _, err := svc.Fetch("KATX"); err != nil {
		t.Fatalf("Fetch() returned error: %v", err)
	}

	history := svc.History("KATX")
	if len(history) != 2 {
		t.Fatalf("History() length = %d, want 2 (recent + just-fetched)", len(history))
	}
	if string(history[0].Data) != "recent" {
		t.Errorf("history[0] data = %q, want %q", history[0].Data, "recent")
	}
	if string(history[1].Data) != "frame" {
		t.Errorf("history[1] data = %q, want %q", history[1].Data, "frame")
	}

	latest, ok := svc.Latest("KATX")
	if !ok {
		t.Fatal("Latest() returned no image")
	}
	if string(latest.Data) != "frame" {
		t.Errorf("Latest() data = %q, want %q", latest.Data, "frame")
	}
}

func TestFetchReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	svc := New(Config{URLTemplate: server.URL + "/{station}.gif"})
	if _, err := svc.Fetch("KATX"); err == nil {
		t.Error("Fetch() expected error for 404 response, got nil")
	}
	if _, ok := svc.Latest("KATX"); ok {
		t.Error("Latest() expected no cached image on failed fetch")
	}
}

func TestFetchEmptyStationID(t *testing.T) {
	svc := New(Config{})
	if _, err := svc.Fetch(""); err == nil {
		t.Error("Fetch(\"\") expected error, got nil")
	}
}

// Compile-time check that *Service satisfies Source.
var _ Source = (*Service)(nil)

func TestFilenameForExtension(t *testing.T) {
	ts := time.Date(2026, 4, 25, 12, 30, 45, 0, time.UTC)
	tests := []struct {
		contentType string
		want        string
	}{
		{"image/gif", "K-20260425T123045Z.gif"},
		{"image/png", "K-20260425T123045Z.png"},
		{"image/jpeg", "K-20260425T123045Z.jpg"},
		{"image/jpg", "K-20260425T123045Z.jpg"},
		{"application/octet-stream", "K-20260425T123045Z.gif"},
	}
	for _, tt := range tests {
		got := filenameFor("K", tt.contentType, ts)
		if got != tt.want {
			t.Errorf("filenameFor(_, %q, _) = %q, want %q", tt.contentType, got, tt.want)
		}
	}
}
