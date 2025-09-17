package middleware

import "net/http"

type errorHandlerConfig struct {
	handlers     map[int]http.Handler
	clearHeaders bool
}

type ErrorHandlerOption func(*errorHandlerConfig)

// WithErrorHandler registers a handler to be invoked when the specified status
// code is returned by the next handler in the chain.
func WithErrorHandler(statusCode int, handler http.Handler) ErrorHandlerOption {
	return func(cfg *errorHandlerConfig) {
		cfg.handlers[statusCode] = handler
	}
}

// WithClearHeadersOnError sets whether or not the headers should be cleared
// when a custom handler is invoked. True by default.
func WithClearHeadersOnError(clearHeaders bool) ErrorHandlerOption {
	return func(cfg *errorHandlerConfig) {
		cfg.clearHeaders = clearHeaders
	}
}

// ErrorHandler is a middleware that handles HTTP status codes by invoking
// a custom handler. Specific error codes can be handled by calling
// WithErrorHandler. If the next handler writes a status code that has a
// registered handler, its response will be dropped.
func ErrorHandler(opts ...ErrorHandlerOption) func(http.Handler) http.Handler {
	config := &errorHandlerConfig{
		handlers:     make(map[int]http.Handler),
		clearHeaders: true,
	}
	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := &errorHandlingWrapper{
				ResponseWriter: w,
				req:            r,
				conf:           config,
				drop:           false,
			}

			next.ServeHTTP(wrapped, r)
		})
	}
}

type errorHandlingWrapper struct {
	http.ResponseWriter
	req     *http.Request
	conf    *errorHandlerConfig
	drop    bool
	headers bool
}

func (e *errorHandlingWrapper) WriteHeader(code int) {
	e.headers = true
	if h, ok := e.conf.handlers[code]; ok {
		e.drop = true

		if e.conf.clearHeaders {
			for k := range e.ResponseWriter.Header() {
				e.ResponseWriter.Header().Del(k)
			}
		}

		h.ServeHTTP(e.ResponseWriter, e.req)
	} else {
		e.ResponseWriter.WriteHeader(code)
	}
}

func (e *errorHandlingWrapper) Write(b []byte) (int, error) {
	if !e.headers {
		e.WriteHeader(http.StatusOK)
	}

	if e.drop {
		return len(b), nil
	}

	n, err := e.ResponseWriter.Write(b)
	return n, err
}
