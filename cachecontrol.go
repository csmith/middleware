package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type cacheControlConfig struct {
	cacheTimes map[string]time.Duration
}

type CacheControlOption func(*cacheControlConfig)

// WithCacheTimes allows setting a custom cache time policy for the CacheControl
// middleware.
//
// cacheTimes is a map of mime types to the max-age that they should be cached.
// The `*` character can be used in place of a subtype (e.g. `image/*`) to match
// all subtypes. More specific entries will be used before wildcard entries.
func WithCacheTimes(cacheTimes map[string]time.Duration) CacheControlOption {
	return func(config *cacheControlConfig) {
		config.cacheTimes = cacheTimes
	}
}

var defaultCacheTimes = map[string]time.Duration{
	"application/*":        time.Hour * 24 * 365,
	"application/xml":      time.Hour,
	"application/atom+xml": time.Hour,
	"application/json":     time.Hour,
	"application/ld+json":  time.Hour,
	"audio/*":              time.Hour * 24 * 365,
	"font/*":               time.Hour * 24 * 365,
	"image/*":              time.Hour * 24 * 365,
	"text/*":               time.Hour,
	"video/*":              time.Hour * 24 * 365,
}

// CacheControl is a middleware that automatically sets a Cache-Control header
// with a max-age based on the Content-Type header set by the next handler.
//
// By default "static" assets like images, videos, and downloads will have a
// max age of 1 year, while text assets like HTML and CSS will have a max-age
// of 1 hour. Use WithCacheTimes to pass custom times.
//
// If the upstream handler sets the Cache-Control header, it will not be changed
// by this middleware.
func CacheControl(opts ...CacheControlOption) func(http.Handler) http.Handler {
	config := &cacheControlConfig{
		cacheTimes: defaultCacheTimes,
	}
	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := &cacheControlWrapper{
				ResponseWriter: w,
				conf:           config,
			}

			next.ServeHTTP(wrapped, r)
		})
	}
}

type cacheControlWrapper struct {
	http.ResponseWriter
	conf    *cacheControlConfig
	headers bool
}

func (c *cacheControlWrapper) WriteHeader(code int) {
	c.headers = true

	if c.ResponseWriter.Header().Get("Cache-Control") != "" {
		// Already has a Cache-Control header, don't replace it
		c.ResponseWriter.WriteHeader(code)
		return
	}

	// See if we have a duration for the full type
	contentType, _, _ := strings.Cut(c.Header().Get("Content-Type"), ";")
	if t, ok := c.conf.cacheTimes[contentType]; ok {
		c.ResponseWriter.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(t.Seconds())))
		c.ResponseWriter.WriteHeader(code)
		return
	}

	// If not try the main type ("audio", "image", etc)
	mainType, _, _ := strings.Cut(contentType, "/")
	if t, ok := c.conf.cacheTimes[fmt.Sprintf("%s/*", mainType)]; ok {
		c.ResponseWriter.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(t.Seconds())))
		c.ResponseWriter.WriteHeader(code)
		return
	}

	c.ResponseWriter.WriteHeader(code)
}

func (c *cacheControlWrapper) Write(b []byte) (int, error) {
	if !c.headers {
		c.WriteHeader(http.StatusOK)
	}

	n, err := c.ResponseWriter.Write(b)
	return n, err
}

func (c *cacheControlWrapper) Flush() {
	if flusher, ok := c.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
