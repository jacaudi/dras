// Package image fetches and caches radar images so they can be attached to
// notifications when a change in radar state is detected.
package image

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jacaudi/dras/internal/logger"
)

// DefaultURLTemplate is the default NWS radar image URL pattern. The
// "{station}" placeholder is replaced with the station ID at fetch time.
// This is the highest-resolution single-image per-station product NWS
// publishes via a static URL (600x550 GIF).
const DefaultURLTemplate = "https://radar.weather.gov/ridge/standard/{station}_0.gif"

// DefaultRetention is the default sliding-window length for cached images.
const DefaultRetention = time.Hour

// stationPlaceholder is the token replaced with the station ID inside a URL
// template.
const stationPlaceholder = "{station}"

// defaultTimeout is the HTTP timeout used when callers do not supply a custom
// client.
const defaultTimeout = 30 * time.Second

// Image is a downloaded radar image plus the metadata needed to attach it to a
// notification.
type Image struct {
	StationID   string
	Data        []byte
	ContentType string
	Filename    string
	FetchedAt   time.Time
}

// Source supplies radar images for stations. Implementations decide where
// images come from (downloaded GIFs, rendered Level II data, etc).
type Source interface {
	// Fetch returns the latest image for the station.
	Fetch(stationID string) (*Image, error)
	// Latest returns the most recent successful image for the station, if
	// any is still within the implementation's retention window.
	Latest(stationID string) (*Image, bool)
}

// Config configures an image Service.
type Config struct {
	// URLTemplate is the radar image URL with "{station}" as the station-ID
	// placeholder. Empty defaults to DefaultURLTemplate.
	URLTemplate string
	// Retention controls how long fetched images are kept in the per-station
	// history. Zero or negative defaults to DefaultRetention.
	Retention time.Duration
	// UserAgent is sent on every image request. Empty means no override.
	UserAgent string
	// HTTPClient is the client used for fetching images. nil installs a
	// client with a sensible timeout.
	HTTPClient *http.Client
}

// Service downloads radar images and caches the most recent images for each
// station so they can be attached when a change notification fires. The cache
// is a sliding window keyed by FetchedAt.
type Service struct {
	httpClient  *http.Client
	urlTemplate string
	userAgent   string
	retention   time.Duration

	mu      sync.RWMutex
	history map[string][]*Image
}

// New creates a new image service from the supplied config. Empty / zero
// fields fall back to sensible defaults.
func New(cfg Config) *Service {
	tmpl := cfg.URLTemplate
	if tmpl == "" {
		tmpl = DefaultURLTemplate
	}

	retention := cfg.Retention
	if retention <= 0 {
		retention = DefaultRetention
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	return &Service{
		httpClient:  client,
		urlTemplate: tmpl,
		userAgent:   cfg.UserAgent,
		retention:   retention,
		history:     make(map[string][]*Image),
	}
}

// Retention returns the configured retention window.
func (s *Service) Retention() time.Duration {
	return s.retention
}

// URLFor returns the radar image URL for the given station based on the
// configured template.
func (s *Service) URLFor(stationID string) string {
	return strings.ReplaceAll(s.urlTemplate, stationPlaceholder, stationID)
}

// Fetch downloads the radar image for the station, appends it to the per
// -station history, prunes images outside the retention window, and returns
// the freshly downloaded image so callers can use it immediately.
func (s *Service) Fetch(stationID string) (*Image, error) {
	if stationID == "" {
		return nil, errors.New("stationID cannot be empty")
	}

	url := s.URLFor(stationID)
	logger.WithFields(map[string]string{
		"station": stationID,
		"url":     url,
	}).Debug("Fetching radar image")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building radar image request for %s: %w", stationID, err)
	}
	if s.userAgent != "" {
		req.Header.Set("User-Agent", s.userAgent)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching radar image for %s: %w", stationID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d fetching radar image for %s", resp.StatusCode, stationID)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading radar image for %s: %w", stationID, err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	now := time.Now()
	img := &Image{
		StationID:   stationID,
		Data:        data,
		ContentType: contentType,
		Filename:    filenameFor(stationID, contentType, now),
		FetchedAt:   now,
	}

	s.mu.Lock()
	s.history[stationID] = pruneHistory(append(s.history[stationID], img), now, s.retention)
	count := len(s.history[stationID])
	s.mu.Unlock()

	logger.WithFields(map[string]string{
		"station":      stationID,
		"bytes":        fmt.Sprintf("%d", len(data)),
		"content_type": contentType,
		"history":      fmt.Sprintf("%d", count),
	}).Debug("Stored radar image")

	return img, nil
}

// Latest returns the most recent image for the station, if any is still within
// the retention window.
func (s *Service) Latest(stationID string) (*Image, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	imgs := s.history[stationID]
	if len(imgs) == 0 {
		return nil, false
	}
	return imgs[len(imgs)-1], true
}

// History returns a copy of the cached images for the station ordered from
// oldest to newest. Images outside the retention window are excluded.
func (s *Service) History(stationID string) []*Image {
	s.mu.RLock()
	defer s.mu.RUnlock()
	imgs := s.history[stationID]
	if len(imgs) == 0 {
		return nil
	}
	out := make([]*Image, len(imgs))
	copy(out, imgs)
	return out
}

// pruneHistory drops entries whose FetchedAt is older than (now - retention)
// and ensures the slice stays sorted oldest-first. Returns a slice that may
// share backing storage with the input.
func pruneHistory(imgs []*Image, now time.Time, retention time.Duration) []*Image {
	if len(imgs) == 0 {
		return imgs
	}
	// Keep order stable by FetchedAt; appends are normally already sorted
	// but be defensive in case fetches complete out of order under load.
	sort.SliceStable(imgs, func(i, j int) bool {
		return imgs[i].FetchedAt.Before(imgs[j].FetchedAt)
	})
	cutoff := now.Add(-retention)
	keepFrom := 0
	for i, img := range imgs {
		if !img.FetchedAt.Before(cutoff) {
			keepFrom = i
			break
		}
		keepFrom = i + 1
	}
	return imgs[keepFrom:]
}

// filenameFor builds a sensible attachment filename based on the content type
// and fetch time so historical images do not collide.
func filenameFor(stationID, contentType string, fetchedAt time.Time) string {
	ext := "gif"
	switch {
	case strings.Contains(contentType, "png"):
		ext = "png"
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		ext = "jpg"
	case strings.Contains(contentType, "gif"):
		ext = "gif"
	}
	return fmt.Sprintf("%s-%s.%s", stationID, fetchedAt.UTC().Format("20060102T150405Z"), ext)
}
