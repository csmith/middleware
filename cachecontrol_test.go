package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheControl_DefaultBehavior(t *testing.T) {
	handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test image"))
	}))

	req := httptest.NewRequest("GET", "/test.png", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := "max-age=31536000" // 1 year in seconds
	assert.Equal(t, expected, rr.Header().Get("Cache-Control"))
}

func TestCacheControl_SpecificContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expectedAge string
	}{
		{"JSON", "application/json", "max-age=3600"},               // 1 hour
		{"XML", "application/xml", "max-age=3600"},                 // 1 hour
		{"Atom", "application/atom+xml", "max-age=3600"},           // 1 hour
		{"Text", "text/plain", "max-age=3600"},                     // 1 hour
		{"CSS", "text/css", "max-age=3600"},                        // 1 hour
		{"Image", "image/jpeg", "max-age=31536000"},                // 1 year
		{"Audio", "audio/mpeg", "max-age=31536000"},                // 1 year
		{"Video", "video/mp4", "max-age=31536000"},                 // 1 year
		{"Font", "font/woff2", "max-age=31536000"},                 // 1 year
		{"Binary", "application/octet-stream", "max-age=31536000"}, // 1 year
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedAge, rr.Header().Get("Cache-Control"))
		})
	}
}

func TestCacheControl_ContentTypeWithCharset(t *testing.T) {
	handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test.json", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := "max-age=3600" // Should match application/json, not fallback to application
	assert.Equal(t, expected, rr.Header().Get("Cache-Control"))
}

func TestCacheControl_FallbackToMainType(t *testing.T) {
	handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp") // Not in specific types, should fallback to "image"
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test.webp", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := "max-age=31536000" // Should match "image" main type
	assert.Equal(t, expected, rr.Header().Get("Cache-Control"))
}

func TestCacheControl_NoMatchingType(t *testing.T) {
	handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "unknown/type")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Cache-Control"))
}

func TestCacheControl_NoContentType(t *testing.T) {
	handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Cache-Control"))
}

func TestCacheControl_ExistingCacheControlHeader(t *testing.T) {
	handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache") // Pre-existing header
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test.png", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should preserve existing Cache-Control header
	assert.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
}

func TestCacheControl_CustomCacheTimes(t *testing.T) {
	customTimes := map[string]time.Duration{
		"application/json": time.Minute * 30,
		"image":            time.Hour * 2,
		"custom/type":      time.Second * 45,
	}

	handler := CacheControl(WithCacheTimes(customTimes))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test.json", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := "max-age=1800" // 30 minutes in seconds
	assert.Equal(t, expected, rr.Header().Get("Cache-Control"))
}

func TestCacheControl_CustomCacheTimesMainTypeFallback(t *testing.T) {
	customTimes := map[string]time.Duration{
		"image/*": time.Hour * 2,
	}

	handler := CacheControl(WithCacheTimes(customTimes))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test.webp", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := "max-age=7200" // 2 hours in seconds
	assert.Equal(t, expected, rr.Header().Get("Cache-Control"))
}

func TestCacheControl_WriteMethod(t *testing.T) {
	handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("test content")) // Test Write method
	}))

	req := httptest.NewRequest("GET", "/test.txt", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=3600", rr.Header().Get("Cache-Control"))
	assert.Equal(t, "test content", rr.Body.String())
}

func TestCacheControl_MultipleWrites(t *testing.T) {
	handler := CacheControl()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("first "))
		w.Write([]byte("second"))
	}))

	req := httptest.NewRequest("GET", "/test.txt", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=3600", rr.Header().Get("Cache-Control"))
	assert.Equal(t, "first second", rr.Body.String())
}
