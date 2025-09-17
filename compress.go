package middleware

import (
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
)

type compressConfig struct {
	gzipLevel int
}

type CompressOption func(*compressConfig)

// WithGzipLevel sets the compression level for gzip encoding
func WithGzipLevel(level int) CompressOption {
	return func(config *compressConfig) {
		config.gzipLevel = level
	}
}

// Compress is a middleware that automatically compresses the response body
// if the client will accept it. It supports gzip encoding.
//
// If an invalid gzip level is set with WithGzipLevel, requests will be silently
// served with no compression.
func Compress(opts ...CompressOption) func(http.Handler) http.Handler {
	config := &compressConfig{
		gzipLevel: gzip.DefaultCompression,
	}
	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			encs := parseEncodings(r.Header.Values("Accept-Encoding"))
			if encs["gzip"] > 0 || encs["*"] > 0 {
				writer, err := gzip.NewWriterLevel(w, config.gzipLevel)
				if err != nil {
					// Bad gzip level, just serve unencoded response
					next.ServeHTTP(w, r)
					return
				}

				defer writer.Close()
				next.ServeHTTP(&gzipWrapper{
					ResponseWriter: w,
					w:              writer,
				}, r)
			} else {
				next.ServeHTTP(&gzipWrapper{
					ResponseWriter: w,
				}, r)
			}
		})
	}
}

func parseEncodings(encoding []string) map[string]float64 {
	codings := make(map[string]float64)
	for i := range encoding {
		parts := strings.Split(encoding[i], ",")
		for p := range parts {
			coding, params, _ := strings.Cut(strings.TrimSpace(parts[p]), ";")
			coding = strings.ToLower(strings.TrimSpace(coding))
			params = strings.TrimSpace(params)
			value := 1.0
			if strings.HasPrefix(params, "q=") {
				value, _ = strconv.ParseFloat(strings.TrimPrefix(params, "q="), 64)
			}
			codings[coding] = value
		}
	}
	return codings
}

type gzipWrapper struct {
	http.ResponseWriter
	w       *gzip.Writer
	headers bool
}

func (g *gzipWrapper) WriteHeader(code int) {
	g.headers = true
	g.ResponseWriter.Header().Add("Vary", "Accept-Encoding")
	if g.w != nil {
		g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		g.ResponseWriter.Header().Del("Content-Length")
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipWrapper) Write(b []byte) (int, error) {
	if !g.headers {
		g.WriteHeader(http.StatusOK)
	}
	if g.w != nil {
		return g.w.Write(b)
	}
	return g.ResponseWriter.Write(b)
}
