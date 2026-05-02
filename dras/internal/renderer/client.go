// Package renderer is an HTTP client for the dras-renderer service. It
// implements image.Source so the monitor can use it interchangeably with the
// legacy ridge-image fetcher.
package renderer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jacaudi/dras/internal/httpretry"
	"github.com/jacaudi/dras/internal/image"
	"github.com/jacaudi/dras/internal/logger"
)

// Config configures a Client.
type Config struct {
	// BaseURL of the renderer (e.g. "http://dras-renderer:8080"). Required.
	BaseURL string
	// Timeout for the entire HTTP round-trip including body read.
	Timeout time.Duration
	// HTTPClient allows callers to inject a custom client (testing).
	HTTPClient *http.Client
	// UserAgent is sent on every request.
	UserAgent string
}

// Client calls the renderer's /render/{station} endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// New constructs a Client. Panics on empty BaseURL.
//
// When cfg.HTTPClient is nil, the default client wraps http.DefaultTransport
// with httpretry.Transport so cold-start races (renderer pod still binding),
// transient 5xx (502/503/504/500), 408/429, and network-layer errors (EOF
// from a worker that hit OOM mid-request, connection refused) are
// transparently retried with exponential backoff. Issue #101 / #103.
func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		panic("renderer.New: BaseURL is required")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{
			Timeout:   cfg.Timeout,
			Transport: httpretry.DefaultTransport(),
		}
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: hc,
		userAgent:  cfg.UserAgent,
	}
}

// Fetch retrieves the rendered image for the station. The supplied ctx controls
// the entire HTTP round-trip lifecycle.
func (c *Client) Fetch(ctx context.Context, stationID string) (*image.Image, error) {
	url := fmt.Sprintf("%s/render/%s", c.baseURL, stationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build renderer request: %w", err)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("renderer request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read renderer body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errBody errorBody
		if jsonErr := json.Unmarshal(body, &errBody); jsonErr == nil && errBody.Error != "" {
			return nil, fmt.Errorf("renderer returned %d: %s (%s)",
				resp.StatusCode, errBody.Error, errBody.Detail)
		}
		return nil, fmt.Errorf("renderer returned %d: %s", resp.StatusCode, string(body))
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode renderer envelope: %w", err)
	}

	pngBytes, err := base64.StdEncoding.DecodeString(env.Image)
	if err != nil {
		return nil, fmt.Errorf("decode base64 image: %w", err)
	}

	scanTime, err := time.Parse(time.RFC3339, env.Metadata.ScanTime)
	if err != nil {
		// Tolerate parse failures: log and use now.
		logger.WithFields(map[string]string{
			"station":   stationID,
			"scan_time": env.Metadata.ScanTime,
		}).Debug("renderer returned non-RFC3339 scan_time; using now")
		scanTime = time.Now().UTC()
	}

	filename := fmt.Sprintf("%s-%s.png", stationID, scanTime.UTC().Format("20060102T150405Z"))

	return &image.Image{
		StationID:   stationID,
		Data:        pngBytes,
		ContentType: "image/png",
		Filename:    filename,
		FetchedAt:   scanTime,
	}, nil
}

// Latest always returns no cached image. The renderer is the source of truth;
// dras does not cache rendered images locally and does not fall back to a
// stale render.
func (c *Client) Latest(stationID string) (*image.Image, bool) {
	return nil, false
}

// Sentinel for callers that want to detect a renderer-source error specifically.
var ErrRenderer = errors.New("renderer error")
