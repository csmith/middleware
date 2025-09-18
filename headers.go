package middleware

import "net/http"

type headersConfig struct {
	headers map[string][]string
}

type HeadersOption func(*headersConfig)

// WithHeader specifies one header to be added to a request. The same key can
// be used multiple times, resulting in multiple headers being set.
func WithHeader(key, value string) HeadersOption {
	return func(config *headersConfig) {
		config.headers[key] = append(config.headers[key], value)
	}
}

// Headers is a middleware that adds headers to a response as late as possible.
// This may be useful when chained with other middleware such as ErrorHandler
// that change headers.
func Headers(opts ...HeadersOption) func(http.Handler) http.Handler {
	conf := &headersConfig{
		headers: make(map[string][]string),
	}
	for _, opt := range opts {
		opt(conf)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(&headersWrapper{
				ResponseWriter: w,
				conf:           conf,
			}, r)
		})
	}
}

type headersWrapper struct {
	http.ResponseWriter
	conf    *headersConfig
	headers bool
}

func (h *headersWrapper) WriteHeader(code int) {
	h.headers = true
	for k := range h.conf.headers {
		for _, v := range h.conf.headers[k] {
			h.ResponseWriter.Header().Add(k, v)
		}
	}
	h.ResponseWriter.WriteHeader(code)
}

func (h *headersWrapper) Write(b []byte) (int, error) {
	if !h.headers {
		h.WriteHeader(http.StatusOK)
	}
	return h.ResponseWriter.Write(b)
}
