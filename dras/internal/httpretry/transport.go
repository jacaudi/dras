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
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
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

	// PerAttemptTimeout, if > 0, bounds each individual request attempt.
	// Each attempt's context is derived from the parent request context
	// with this deadline; on expiry the attempt fails with
	// context.DeadlineExceeded and the retry loop tries again. The overall
	// budget is therefore PerAttemptTimeout * MaxAttempts + total backoff,
	// rather than one shared deadline. This is the standard fix for
	// "Client.Timeout exceeded while awaiting headers" — a single slow
	// attempt can no longer starve subsequent attempts of budget.
	//
	// Set this instead of http.Client.Timeout: the latter applies a single
	// deadline to the entire round-trip including all retries, which
	// defeats the retry loop when an individual attempt is slow.
	PerAttemptTimeout time.Duration
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
//
// When PerAttemptTimeout > 0, each attempt runs under its own derived
// context with that deadline. The returned response's Body wraps the
// per-attempt cancel so closing the body releases the timer; callers must
// close the body as usual.
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
	perAttempt := t.PerAttemptTimeout

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
		// Bail before issuing if the parent context is already done.
		if err := req.Context().Err(); err != nil {
			return nil, err
		}

		// Build the per-attempt request. When PerAttemptTimeout is set
		// each attempt gets its own deadline; otherwise we reuse the
		// parent context directly.
		attemptReq := req
		var cancel context.CancelFunc
		if perAttempt > 0 {
			var attemptCtx context.Context
			attemptCtx, cancel = context.WithTimeout(req.Context(), perAttempt)
			attemptReq = req.WithContext(attemptCtx)
		}
		if bodyBytes != nil {
			attemptReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, lastErr = base.RoundTrip(attemptReq)

		// Parent-context cancellation must not be retried — the caller
		// gave up. Per-attempt deadlines are retryable (treated as a
		// transient slow attempt) and fall through to shouldRetry.
		if pErr := req.Context().Err(); pErr != nil {
			drainAndClose(resp)
			if cancel != nil {
				cancel()
			}
			return nil, pErr
		}

		retry := shouldRetry(resp, lastErr)
		isLast := attempt >= maxAttempts
		if !retry || isLast {
			// Terminal: success, non-transient error, or out of attempts.
			// Wrap the body so closing it releases the per-attempt
			// context timer; callers already close response bodies.
			if resp != nil && cancel != nil {
				resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
			} else if cancel != nil {
				cancel()
			}
			return resp, lastErr
		}

		// Compute wait before next attempt; honor Retry-After if present.
		wait := backoffFor(attempt, initial, maxBackoff, resp)

		// Log the upcoming retry. Best-effort — never block on logger.
		attrs := []any{
			"attempt", strconv.Itoa(attempt),
			"max_attempts", strconv.Itoa(maxAttempts),
			"wait_ms", strconv.FormatInt(wait.Milliseconds(), 10),
			"url", req.URL.String(),
		}
		if lastErr != nil {
			attrs = append(attrs, "err", lastErr.Error())
		}
		if resp != nil {
			attrs = append(attrs, "status", strconv.Itoa(resp.StatusCode))
		}
		slog.Debug("retrying transient HTTP failure", attrs...)

		// Drain and close any previous response body so the connection
		// can be reused, then release the per-attempt context.
		drainAndClose(resp)
		resp = nil
		if cancel != nil {
			cancel()
		}

		// Wait, respecting parent context cancellation.
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

// drainAndClose discards and closes a response body, ignoring errors. The
// drain lets the underlying connection return to the pool for reuse.
func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// cancelOnClose wraps a response body so that Close() also cancels the
// per-attempt context. The caller's normal "defer resp.Body.Close()" then
// releases the timer; without this the timer would leak until it fires
// (bounded by PerAttemptTimeout but still ugly).
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
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
