// Package image fetches and caches radar images so they can be attached to
// notifications when a change in radar state is detected.
package image

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jacaudi/dras/internal/logger"
)

// DefaultURLTemplate is the default NWS radar image URL pattern. The
// "{station}" placeholder is replaced with the station ID at fetch time.
const DefaultURLTemplate = "https://radar.weather.gov/ridge/standard/{station}_0.gif"

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

// Service downloads radar images and caches the most recent image for each
// station so it can be attached when a change notification fires.
type Service struct {
	httpClient  *http.Client
	urlTemplate string

	mu    sync.RWMutex
	cache map[string]*Image
}

// New creates a new image service. If urlTemplate is empty the default NWS
// template is used.
func New(urlTemplate string) *Service {
	if urlTemplate == "" {
		urlTemplate = DefaultURLTemplate
	}
	return &Service{
		httpClient:  &http.Client{Timeout: defaultTimeout},
		urlTemplate: urlTemplate,
		cache:       make(map[string]*Image),
	}
}

// URLFor returns the radar image URL for the given station based on the
// configured template.
func (s *Service) URLFor(stationID string) string {
	return strings.ReplaceAll(s.urlTemplate, stationPlaceholder, stationID)
}

// Fetch downloads the radar image for the station and stores it in the cache.
// The latest image is also returned so callers can use it immediately.
func (s *Service) Fetch(stationID string) (*Image, error) {
	if stationID == "" {
		return nil, errors.New("stationID cannot be empty")
	}

	url := s.URLFor(stationID)
	logger.WithFields(map[string]string{
		"station": stationID,
		"url":     url,
	}).Debug("Fetching radar image")

	resp, err := s.httpClient.Get(url)
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

	img := &Image{
		StationID:   stationID,
		Data:        data,
		ContentType: contentType,
		Filename:    filenameFor(stationID, contentType),
		FetchedAt:   time.Now(),
	}

	s.mu.Lock()
	s.cache[stationID] = img
	s.mu.Unlock()

	logger.WithFields(map[string]string{
		"station":      stationID,
		"bytes":        fmt.Sprintf("%d", len(data)),
		"content_type": contentType,
	}).Debug("Stored radar image")

	return img, nil
}

// Get returns the most recently cached image for the station, if any.
func (s *Service) Get(stationID string) (*Image, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	img, ok := s.cache[stationID]
	return img, ok
}

// filenameFor builds a sensible attachment filename based on the content type.
func filenameFor(stationID, contentType string) string {
	ext := "gif"
	switch {
	case strings.Contains(contentType, "png"):
		ext = "png"
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		ext = "jpg"
	case strings.Contains(contentType, "gif"):
		ext = "gif"
	}
	return fmt.Sprintf("%s.%s", stationID, ext)
}
