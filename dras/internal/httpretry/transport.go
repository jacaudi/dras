// Package httpretry provides an http.RoundTripper that retries transient
// failures (network errors, 5xx, 408, 429) with exponential backoff and
// jitter. It is used by both the renderer client and the legacy NWS radar
// image client to absorb upstream flakiness — connection refused on a
// cold-starting renderer, transient 5xx from NWS ridge, EOF from a renderer
// worker that hit OOM mid-request, etc.
//
// All requests dras makes to upstream services are GETs; retrying GETs is
// always safe.
package httpretry

import (
	"bytes"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/jacaudi/dras/internal/logger"
)

// Transport wraps a base http.RoundTripper with bounded retry-with-backoff
// for transient failures. Retries are GET-safe (network error, 5xx, 408,
// 429). Non-transient failures (4xx other than 408/429) and successes are
// returned immediately.
type Transport struct {
	// Base is the underlying transport. Defaults to http.DefaultTransport.
	Base http.RoundTripper

	// MaxAttempts is the total request count including the first try. A
	// value <= 1 disables retries. Defaults to 4 (1 initial + 3 retries).
	MaxAttempts int

	// InitialBackoff is the wait before the first retry. Subsequent retries
	// double the wait up to MaxBackoff. Defaults to 1s.
	InitialBackoff time.Duration

	// MaxBackoff caps the per-retry wait. Defaults to 30s.
	MaxBackoff time.Duration
}

// DefaultTransport returns a Transport with the documented defaults.
func DefaultTransport() *Transport {
	return &Transport{
		Base:           http.DefaultTransport,
		MaxAttempts:    4,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
	}
}

// RoundTrip executes req with retry-on-transient-failure. The request body
// is buffered up front so it can be replayed on each attempt. Context
// cancellation aborts the retry loop immediately.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	maxAttempts := t.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	initial := t.InitialBackoff
	if initial <= 0 {
		initial = 1 * time.Second
	}
	maxBackoff := t.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}

	// Buffer the body so we can reset it on each attempt.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
	}

	var (
		resp    *http.Response
		lastErr error
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Restore body for this attempt.
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, lastErr = base.RoundTrip(req)

		if !shouldRetry(resp, lastErr) {
			return resp, lastErr
		}
		if attempt >= maxAttempts {
			return resp, lastErr
		}

		// Compute wait before next attempt; honor Retry-After if present.
		wait := backoffFor(attempt, initial, maxBackoff, resp)

		// Log the upcoming retry. Best-effort — never block on logger.
		fields := map[string]string{
			"attempt":      strconv.Itoa(attempt),
			"max_attempts": strconv.Itoa(maxAttempts),
			"wait_ms":      strconv.FormatInt(wait.Milliseconds(), 10),
			"url":          req.URL.String(),
		}
		if lastErr != nil {
			fields["err"] = lastErr.Error()
		}
		if resp != nil {
			fields["status"] = strconv.Itoa(resp.StatusCode)
		}
		logger.WithFields(fields).Debug("retrying transient HTTP failure")

		// Drain and close any previous response body so the connection
		// can be reused.
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			resp = nil
		}

		// Wait, respecting context cancellation.
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		}
	}

	return resp, lastErr
}

// shouldRetry returns true if the (resp, err) pair represents a transient
// failure worth retrying.
func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		// Network errors (connection refused, EOF, DNS failure, timeout
		// at the transport layer) all surface as non-nil err. Context
		// cancellation also lands here — but the caller's context check
		// inside the retry loop handles it before we hit this path.
		return !errors.Is(err, http.ErrSchemeMismatch)
	}
	if resp == nil {
		return false
	}
	switch resp.StatusCode {
	case http.StatusRequestTimeout, // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	return false
}

// backoffFor returns the wait duration before retry attempt+1. Honors a
// Retry-After response header (in seconds) when present. Otherwise uses
// exponential backoff with full jitter capped at maxBackoff.
func backoffFor(attempt int, initial, maxBackoff time.Duration, resp *http.Response) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
				wait := time.Duration(secs) * time.Second
				if wait > maxBackoff {
					wait = maxBackoff
				}
				return wait
			}
		}
	}
	// Exponential: initial, 2*initial, 4*initial, ... capped at max.
	d := initial << (attempt - 1)
	if d <= 0 || d > maxBackoff {
		d = maxBackoff
	}
	// Full-jitter (AWS-style): pick uniform in [0, d). This avoids the
	// thundering-herd pattern where every client retries at exactly the
	// same multiple of the backoff base.
	jitterNs := rand.Int64N(int64(d))
	return time.Duration(jitterNs)
}
