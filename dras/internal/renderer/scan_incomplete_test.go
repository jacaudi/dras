package renderer

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jacaudi/dras/internal/httpretry"
	"github.com/jacaudi/dras/internal/image"
)

// Tests for the benign "scan not yet complete" (No MSG31 records) skip
// classification (issue #122). Covers the 7 categories; the monitor-side
// INFO-vs-WARN log integration lives in the monitor package.

// decodeFailedBody is the wire shape the renderer emits for a mid-write
// upstream volume: HTTP 502 with error=decode_failed and the Py-ART
// ValueError text in detail.
const noMSG31Detail = "No MSG31 records found, cannot read file"

func writeErr(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: code, Detail: detail})
}

// --- Unit -----------------------------------------------------------------
func TestIsScanIncomplete_Unit(t *testing.T) {
	tests := []struct {
		name string
		body errorBody
		want bool
	}{
		{name: "benign_no_msg31", body: errorBody{Error: "decode_failed", Detail: noMSG31Detail}, want: true},
		{name: "case_insensitive", body: errorBody{Error: "decode_failed", Detail: "no msg31 records here"}, want: true},
		{name: "decode_failed_other_detail", body: errorBody{Error: "decode_failed", Detail: "truncated gzip member"}, want: false},
		{name: "other_code_same_detail", body: errorBody{Error: "internal", Detail: noMSG31Detail}, want: false},
		{name: "no_recent_scan", body: errorBody{Error: "no_recent_scan", Detail: "none today"}, want: false},
		{name: "empty", body: errorBody{}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isScanIncomplete(tt.body); got != tt.want {
				t.Errorf("isScanIncomplete(%+v) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

// --- Functional -----------------------------------------------------------
func TestFetch_ScanIncomplete_WrapsSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusBadGateway, "decode_failed", noMSG31Detail)
	}))
	defer srv.Close()

	// Non-retrying client: this asserts classification, not retry behavior.
	c := New(Config{BaseURL: srv.URL, HTTPClient: &http.Client{Timeout: 5 * time.Second}})
	_, err := c.Fetch(t.Context(), "KATX")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, image.ErrScanIncomplete) {
		t.Errorf("err = %v, want errors.Is(err, image.ErrScanIncomplete)", err)
	}
	// Detail is preserved for forensics.
	if !strings.Contains(err.Error(), noMSG31Detail) {
		t.Errorf("err = %v, want to preserve detail %q", err, noMSG31Detail)
	}
}

func TestFetch_DecodeFailedOther_IsHardError(t *testing.T) {
	// A decode_failed that is NOT the mid-write case must stay a hard error
	// (not downgraded), so genuine corruption is still visible as a WARN.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusBadGateway, "decode_failed", "truncated gzip member")
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, HTTPClient: &http.Client{Timeout: 5 * time.Second}})
	_, err := c.Fetch(t.Context(), "KATX")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, image.ErrScanIncomplete) {
		t.Errorf("err = %v, must NOT be classified as scan-incomplete", err)
	}
	if !strings.Contains(err.Error(), "decode_failed") {
		t.Errorf("err = %v, want to contain 'decode_failed'", err)
	}
}

// --- Security -------------------------------------------------------------
// Adversarial/oversized detail strings must neither panic nor be
// misclassified. Only the exact stable substring under decode_failed trips
// the skip; a hostile detail cannot smuggle a real failure into the silent
// INFO path, and a real failure cannot be masked.
func TestFetch_ScanIncomplete_Security_NoMisclassify(t *testing.T) {
	cases := []struct {
		code, detail string
		wantSkip     bool
	}{
		{"decode_failed", strings.Repeat("A", 1<<16), false},            // oversized, unrelated
		{"decode_failed", "MSG31 records missing", false},               // near-miss wording, not the phrase
		{"decode_failed", "prefix no msg31 records suffix", true},       // substring embedded, still benign
		{"internal", "no msg31 records\n'; DROP TABLE scans;--", false}, // injection-ish + wrong code
		{"decode_failed", "no msg31 records\x00 truncated", true},       // NUL byte, still matches phrase
	}
	for _, tc := range cases {
		got := isScanIncomplete(errorBody{Error: tc.code, Detail: tc.detail})
		if got != tc.wantSkip {
			t.Errorf("isScanIncomplete(code=%q detail=%q) = %v, want %v", tc.code, truncate(tc.detail), got, tc.wantSkip)
		}
	}
}

func truncate(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}

// --- Retry ----------------------------------------------------------------
// The renderer 502 is retried with backoff by the default transport (a
// mid-write volume often completes between attempts). This asserts the
// external call is retried AND the sentinel survives to the final error, so
// the INFO classification still applies after retries are exhausted.
func TestFetch_ScanIncomplete_Retry(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		writeErr(w, http.StatusBadGateway, "decode_failed", noMSG31Detail)
	}))
	defer srv.Close()

	rt := &httpretry.Transport{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	}
	c := New(Config{BaseURL: srv.URL, HTTPClient: &http.Client{Transport: rt}})

	_, err := c.Fetch(t.Context(), "KATX")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("server hits = %d, want 3 (all attempts retried with backoff)", got)
	}
	if !errors.Is(err, image.ErrScanIncomplete) {
		t.Errorf("err after retries = %v, want errors.Is(err, image.ErrScanIncomplete)", err)
	}
}

// --- Performance ----------------------------------------------------------
func BenchmarkIsScanIncomplete(b *testing.B) {
	body := errorBody{Error: "decode_failed", Detail: noMSG31Detail}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isScanIncomplete(body)
	}
}

// --- Frame ----------------------------------------------------------------
// Builds & runs end-to-end: real HTTP round-trip through the client against a
// server emitting the exact renderer 502 body, asserting the wrapped sentinel
// reaches the caller.
func TestFetch_ScanIncomplete_Frame_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/render/KATX" {
			t.Errorf("path = %q", r.URL.Path)
		}
		writeErr(w, http.StatusBadGateway, "decode_failed", noMSG31Detail)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, HTTPClient: &http.Client{Timeout: 5 * time.Second}})
	img, err := c.Fetch(t.Context(), "KATX")
	if img != nil {
		t.Errorf("expected nil image on scan-incomplete, got %+v", img)
	}
	if !errors.Is(err, image.ErrScanIncomplete) {
		t.Fatalf("Frame: err = %v, want errors.Is(err, image.ErrScanIncomplete)", err)
	}
}
