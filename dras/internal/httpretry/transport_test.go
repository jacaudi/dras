package httpretry

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeTransport is a stub RoundTripper that returns a scripted sequence of
// (resp, err) pairs.
type fakeTransport struct {
	calls    atomic.Int32
	scripted []roundTripResult
}

type roundTripResult struct {
	statusCode int
	body       string
	err        error
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := int(f.calls.Add(1)) - 1
	if idx >= len(f.scripted) {
		// Out of script: fail loudly so test sees the unexpected call.
		return nil, errors.New("fakeTransport: more calls than scripted")
	}
	r := f.scripted[idx]
	if r.err != nil {
		return nil, r.err
	}
	return &http.Response{
		StatusCode: r.statusCode,
		Body:       io.NopCloser(strings.NewReader(r.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func newRequest(t *testing.T, ctx context.Context) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/x", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	return req
}

func TestRoundTrip_SuccessOnFirstAttempt(t *testing.T) {
	stub := &fakeTransport{scripted: []roundTripResult{
		{statusCode: 200, body: "ok"},
	}}
	tr := &Transport{
		Base:           stub,
		MaxAttempts:    4,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     1 * time.Millisecond,
	}
	resp, err := tr.RoundTrip(newRequest(t, context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if got := stub.calls.Load(); got != 1 {
		t.Errorf("got %d calls, want 1", got)
	}
}

func TestRoundTrip_RetriesOn5xxThenSucceeds(t *testing.T) {
	stub := &fakeTransport{scripted: []roundTripResult{
		{statusCode: 503, body: "unavailable"},
		{statusCode: 502, body: "bad gateway"},
		{statusCode: 200, body: "ok"},
	}}
	tr := &Transport{
		Base:           stub,
		MaxAttempts:    4,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     1 * time.Millisecond,
	}
	resp, err := tr.RoundTrip(newRequest(t, context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if got := stub.calls.Load(); got != 3 {
		t.Errorf("got %d calls, want 3", got)
	}
}

func TestRoundTrip_RetriesOnNetworkErrorThenSucceeds(t *testing.T) {
	netErr := errors.New("connection refused")
	stub := &fakeTransport{scripted: []roundTripResult{
		{err: netErr},
		{err: netErr},
		{statusCode: 200, body: "ok"},
	}}
	tr := &Transport{
		Base:           stub,
		MaxAttempts:    4,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     1 * time.Millisecond,
	}
	resp, err := tr.RoundTrip(newRequest(t, context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if got := stub.calls.Load(); got != 3 {
		t.Errorf("got %d calls, want 3", got)
	}
}

func TestRoundTrip_DoesNotRetryOn4xx(t *testing.T) {
	stub := &fakeTransport{scripted: []roundTripResult{
		{statusCode: 404, body: "not found"},
	}}
	tr := &Transport{
		Base:           stub,
		MaxAttempts:    4,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     1 * time.Millisecond,
	}
	resp, err := tr.RoundTrip(newRequest(t, context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("got status %d, want 404", resp.StatusCode)
	}
	if got := stub.calls.Load(); got != 1 {
		t.Errorf("got %d calls, want 1 (4xx must not retry)", got)
	}
}

func TestRoundTrip_GivesUpAfterMaxAttempts(t *testing.T) {
	stub := &fakeTransport{scripted: []roundTripResult{
		{statusCode: 503}, {statusCode: 503}, {statusCode: 503},
	}}
	tr := &Transport{
		Base:           stub,
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     1 * time.Millisecond,
	}
	resp, err := tr.RoundTrip(newRequest(t, context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Errorf("got status %d, want 503 (last response surfaced)", resp.StatusCode)
	}
	if got := stub.calls.Load(); got != 3 {
		t.Errorf("got %d calls, want exactly 3 (capped by MaxAttempts)", got)
	}
}

func TestRoundTrip_HonorsContextCancellation(t *testing.T) {
	stub := &fakeTransport{scripted: []roundTripResult{
		{statusCode: 503}, {statusCode: 503}, {statusCode: 503},
	}}
	tr := &Transport{
		Base:           stub,
		MaxAttempts:    4,
		InitialBackoff: 200 * time.Millisecond, // long enough to observe cancel
		MaxBackoff:     200 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := tr.RoundTrip(newRequest(t, ctx))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// Should have made fewer than 4 calls because ctx canceled mid-backoff.
	if got := stub.calls.Load(); got >= 4 {
		t.Errorf("got %d calls, expected fewer (context cancellation should short-circuit)", got)
	}
}

func TestRoundTrip_PerAttemptTimeoutRetriesAfterSlowAttempt(t *testing.T) {
	// First attempt sleeps past the per-attempt timeout; second attempt
	// returns immediately. The retry loop should give the second attempt
	// its own clock and succeed.
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(500 * time.Millisecond):
				return
			}
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := &http.Client{Transport: &Transport{
		Base:              http.DefaultTransport,
		MaxAttempts:       4,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        5 * time.Millisecond,
		PerAttemptTimeout: 50 * time.Millisecond,
	}}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("body = %q, want ok", body)
	}
	// The first attempt is canceled at ~50ms; second attempt starts and
	// returns immediately. Total wall time should be well under the
	// would-be-shared 500ms first-attempt budget.
	if elapsed := time.Since(start); elapsed > 300*time.Millisecond {
		t.Errorf("elapsed = %v, expected <300ms (per-attempt timeout failed to scope)", elapsed)
	}
	if got := n.Load(); got != 2 {
		t.Errorf("server saw %d calls, want 2 (1 slow + 1 fast)", got)
	}
}

func TestRoundTrip_PerAttemptTimeoutGivesUpAfterMaxAttempts(t *testing.T) {
	// Every attempt sleeps past the per-attempt timeout. Final error
	// should be context.DeadlineExceeded (the last attempt's per-attempt
	// timeout firing).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := &http.Client{Transport: &Transport{
		Base:              http.DefaultTransport,
		MaxAttempts:       3,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        5 * time.Millisecond,
		PerAttemptTimeout: 30 * time.Millisecond,
	}}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := client.Do(req)
	if err == nil {
		t.Fatal("expected error after all attempts time out")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") &&
		!errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestRoundTrip_PerAttemptTimeoutBodyReadableAfterReturn(t *testing.T) {
	// Success case with PerAttemptTimeout set. The response body wraps a
	// cancel-on-close; reading the body before close must work, and
	// closing the body must not error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	tr := &Transport{
		Base:              http.DefaultTransport,
		MaxAttempts:       4,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        5 * time.Millisecond,
		PerAttemptTimeout: 1 * time.Second,
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "hello" {
		t.Errorf("body = %q, want hello", body)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestRoundTrip_ParentCancelStopsRetriesNotPerAttempt(t *testing.T) {
	// Parent context cancellation must short-circuit the retry loop,
	// even if the per-attempt timeout would have allowed another try.
	stub := &fakeTransport{scripted: []roundTripResult{
		{statusCode: 503}, {statusCode: 503}, {statusCode: 503}, {statusCode: 503},
	}}
	tr := &Transport{
		Base:              stub,
		MaxAttempts:       4,
		InitialBackoff:    200 * time.Millisecond,
		MaxBackoff:        200 * time.Millisecond,
		PerAttemptTimeout: 1 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := tr.RoundTrip(newRequest(t, ctx))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// End-to-end test against a real httptest server: the renderer's actual
// failure pattern (alternating EOF/500) gets recovered.
func TestRoundTrip_EndToEndRecoversAlternatingFailures(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch n.Add(1) {
		case 1:
			// Simulate EOF mid-response: hijack the conn and close it.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Errorf("server: hijacker not available")
				return
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = conn.Close()
		case 2:
			http.Error(w, "internal", http.StatusInternalServerError)
		default:
			_, _ = w.Write([]byte("ok"))
		}
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: &Transport{
			Base:           http.DefaultTransport,
			MaxAttempts:    4,
			InitialBackoff: 1 * time.Millisecond,
			MaxBackoff:     5 * time.Millisecond,
		},
		Timeout: 2 * time.Second,
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("got body %q, want %q", body, "ok")
	}
	if got := n.Load(); got != 3 {
		t.Errorf("got %d server calls, want 3 (1 EOF + 1 500 + 1 success)", got)
	}
}
