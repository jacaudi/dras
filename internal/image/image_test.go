package image

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestURLForUsesTemplate(t *testing.T) {
	svc := New("https://example.com/{station}/img.gif")
	got := svc.URLFor("KRAX")
	want := "https://example.com/KRAX/img.gif"
	if got != want {
		t.Errorf("URLFor() = %q, want %q", got, want)
	}
}

func TestURLForDefaultTemplate(t *testing.T) {
	svc := New("")
	got := svc.URLFor("KRAX")
	if !strings.Contains(got, "KRAX") {
		t.Errorf("URLFor() = %q, expected to contain KRAX", got)
	}
	if !strings.Contains(got, "radar.weather.gov") {
		t.Errorf("URLFor() = %q, expected default NWS host", got)
	}
}

func TestFetchStoresImageInCache(t *testing.T) {
	body := []byte("GIF89a-fake-image-bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/gif")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	svc := New(server.URL + "/{station}.gif")

	img, err := svc.Fetch("KATX")
	if err != nil {
		t.Fatalf("Fetch() returned error: %v", err)
	}
	if img == nil {
		t.Fatal("Fetch() returned nil image")
	}
	if string(img.Data) != string(body) {
		t.Errorf("Fetch() data = %q, want %q", img.Data, body)
	}
	if img.ContentType != "image/gif" {
		t.Errorf("Fetch() content type = %q, want image/gif", img.ContentType)
	}
	if img.Filename != "KATX.gif" {
		t.Errorf("Fetch() filename = %q, want KATX.gif", img.Filename)
	}

	cached, ok := svc.Get("KATX")
	if !ok {
		t.Fatal("Get() returned no cached image after Fetch()")
	}
	if string(cached.Data) != string(body) {
		t.Errorf("Get() data = %q, want %q", cached.Data, body)
	}
}

func TestFetchReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	svc := New(server.URL + "/{station}.gif")
	if _, err := svc.Fetch("KATX"); err == nil {
		t.Error("Fetch() expected error for 404 response, got nil")
	}
	if _, ok := svc.Get("KATX"); ok {
		t.Error("Get() expected no cached image on failed fetch")
	}
}

func TestFetchEmptyStationID(t *testing.T) {
	svc := New("")
	if _, err := svc.Fetch(""); err == nil {
		t.Error("Fetch(\"\") expected error, got nil")
	}
}

func TestFilenameForExtension(t *testing.T) {
	tests := []struct {
		contentType string
		want        string
	}{
		{"image/gif", "K.gif"},
		{"image/png", "K.png"},
		{"image/jpeg", "K.jpg"},
		{"image/jpg", "K.jpg"},
		{"application/octet-stream", "K.gif"},
	}
	for _, tt := range tests {
		got := filenameFor("K", tt.contentType)
		if got != tt.want {
			t.Errorf("filenameFor(_, %q) = %q, want %q", tt.contentType, got, tt.want)
		}
	}
}
