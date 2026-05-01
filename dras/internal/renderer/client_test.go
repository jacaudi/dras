package renderer

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jacaudi/dras/internal/image"
)

func TestFetchHappyPath(t *testing.T) {
	pngBytes := []byte{0x89, 'P', 'N', 'G', 'X'}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/render/KATX" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope{
			Image: base64.StdEncoding.EncodeToString(pngBytes),
			Metadata: metadata{
				Station:         "KATX",
				Product:         "base_reflectivity",
				ScanTime:        "2026-04-26T15:32:00Z",
				ElevationDeg:    0.5,
				VCP:             215,
				RendererVersion: "v3.0.0-test",
			},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	img, err := c.Fetch("KATX")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(img.Data) != string(pngBytes) {
		t.Errorf("data = %q, want %q", img.Data, pngBytes)
	}
	if img.ContentType != "image/png" {
		t.Errorf("content type = %q", img.ContentType)
	}
	if img.StationID != "KATX" {
		t.Errorf("station = %q", img.StationID)
	}
	if img.Filename == "" {
		t.Error("filename empty")
	}
}

func TestFetchServerErrorReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"no_recent_scan","detail":"none today"}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	_, err := c.Fetch("KATX")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no_recent_scan") {
		t.Errorf("err = %v, want to contain 'no_recent_scan'", err)
	}
}

func TestFetchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Timeout: 100 * time.Millisecond})
	_, err := c.Fetch("KATX")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestFetchMalformedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not json"))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	_, err := c.Fetch("KATX")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchBadBase64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"image":"!!!not-base64!!!","metadata":{"station":"KATX","product":"base_reflectivity","scan_time":"2026-04-26T15:32:00Z","elevation_deg":0.5,"vcp":215,"renderer_version":"v"}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	_, err := c.Fetch("KATX")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLatestAlwaysReturnsFalse(t *testing.T) {
	c := New(Config{BaseURL: "http://example.invalid", Timeout: time.Second})
	if _, ok := c.Latest("KATX"); ok {
		t.Error("Latest should always return false in renderer mode")
	}
}

// Compile-time check that *Client satisfies image.Source.
var _ image.Source = (*Client)(nil)
