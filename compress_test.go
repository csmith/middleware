package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompress_NoAcceptEncoding(t *testing.T) {
	handler := Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "test content", rr.Body.String())
	assert.Equal(t, "Accept-Encoding", rr.Header().Get("Vary"))
}

func TestCompress_GzipAccepted(t *testing.T) {
	handler := Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	reader, err := gzip.NewReader(rr.Body)
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(decompressed))
}

func TestCompress_WildcardAccepted(t *testing.T) {
	handler := Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "*")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	reader, err := gzip.NewReader(rr.Body)
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(decompressed))
}

func TestCompress_NoGzipOrWildcard(t *testing.T) {
	handler := Compress()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "deflate, br")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "test content", rr.Body.String())
}

func TestCompress_InvalidGzipLevel(t *testing.T) {
	handler := Compress(WithGzipLevel(15))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "test content", rr.Body.String())
}

func TestCompress_ValidGzipLevel(t *testing.T) {
	handler := Compress(WithGzipLevel(gzip.BestSpeed))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	reader, err := gzip.NewReader(rr.Body)
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(decompressed))
}

func TestParseEncodings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]float64
	}{
		{
			name:     "simple gzip",
			input:    []string{"gzip"},
			expected: map[string]float64{"gzip": 1.0},
		},
		{
			name:     "gzip with weight",
			input:    []string{"gzip;q=0.8"},
			expected: map[string]float64{"gzip": 0.8},
		},
		{
			name:     "multiple encodings single header",
			input:    []string{"gzip;q=0.8, deflate;q=0.9"},
			expected: map[string]float64{"gzip": 0.8, "deflate": 0.9},
		},
		{
			name:     "multiple header values",
			input:    []string{"gzip;q=0.8", "deflate;q=0.9"},
			expected: map[string]float64{"gzip": 0.8, "deflate": 0.9},
		},
		{
			name:     "zero weight",
			input:    []string{"gzip;q=0"},
			expected: map[string]float64{"gzip": 0.0},
		},
		{
			name:     "wildcard",
			input:    []string{"*;q=0.1"},
			expected: map[string]float64{"*": 0.1},
		},
		{
			name:     "case insensitive encoding",
			input:    []string{"GZIP, Deflate"},
			expected: map[string]float64{"gzip": 1.0, "deflate": 1.0},
		},
		{
			name:     "whitespace handling",
			input:    []string{" gzip ; q=0.8 , deflate "},
			expected: map[string]float64{"gzip": 0.8, "deflate": 1.0},
		},
		{
			name:     "invalid q value",
			input:    []string{"gzip;q=invalid"},
			expected: map[string]float64{"gzip": 0.0},
		},
		{
			name:     "max precision q value",
			input:    []string{"gzip;q=0.123"},
			expected: map[string]float64{"gzip": 0.123},
		},
		{
			name:     "complex real world example",
			input:    []string{"gzip, deflate, br;q=0.9, *;q=0.1"},
			expected: map[string]float64{"gzip": 1.0, "deflate": 1.0, "br": 0.9, "*": 0.1},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEncodings(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
