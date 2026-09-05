package middleware

import (
	"net/http"
	"time"
)

// RequestResult describes the outcome of a single request as observed by
// ObserveRequests.
type RequestResult struct {
	// StatusCode is the final, non-informational status code committed to the
	// response. A handler that returns without writing anything reports 200,
	// matching the behaviour of net/http. A value of 0 indicates the request
	// ended before any final response existed, e.g. the handler panicked before
	// writing a final header.
	StatusCode int
	// BytesWritten is the number of body bytes successfully written to the
	// response. Short or failing writes only count the bytes actually written.
	BytesWritten int64
	// Duration is the total time spent serving the request.
	Duration time.Duration
}

// RequestObserver is invoked once per request with the incoming request and
// the observed result. It runs synchronously on the request's goroutine, so
// slow observers will delay responses.
type RequestObserver func(*http.Request, RequestResult)

type observeRequestsConfig struct {
	observer RequestObserver
}

// ObserveRequestsOption configures the ObserveRequests middleware.
type ObserveRequestsOption func(*observeRequestsConfig)

// WithRequestObserver configures the function invoked after each request.
// Passing nil disables observation.
func WithRequestObserver(observer RequestObserver) ObserveRequestsOption {
	return func(config *observeRequestsConfig) {
		config.observer = observer
	}
}

// ObserveRequests is a middleware that invokes the configured observer exactly
// once after each downstream request, including when the downstream handlers
// panic (in which case the panic is re-raised after the observer has run).
// With no observer configured, the middleware returns the next handler unchanged.
func ObserveRequests(opts ...ObserveRequestsOption) func(http.Handler) http.Handler {
	config := &observeRequestsConfig{}
	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		if config.observer == nil {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := &observingWrapper{ResponseWriter: w}
			var writer http.ResponseWriter = wrapped
			if flusher, ok := w.(http.Flusher); ok {
				writer = &flushingObservingWrapper{
					observingWrapper: wrapped,
					flusher:          flusher,
				}
			}
			start := time.Now()

			completed := false
			defer func() {
				status := wrapped.status
				if completed && status == 0 {
					// The handler returned normally without committing a
					// response, which net/http serves as an implicit 200.
					status = http.StatusOK
				}
				config.observer(r, RequestResult{
					StatusCode:   status,
					BytesWritten: wrapped.written,
					Duration:     time.Since(start),
				})
			}()

			next.ServeHTTP(writer, r)
			completed = true
		})
	}
}

type observingWrapper struct {
	http.ResponseWriter
	written    int64
	status     int
	wroteFinal bool
}

func (o *observingWrapper) WriteHeader(code int) {
	o.ResponseWriter.WriteHeader(code)

	// net/http permits any number of informational responses before the final
	// status. A 101 response is the exception: it completes the HTTP exchange.
	if !o.wroteFinal && (code >= http.StatusOK || code == http.StatusSwitchingProtocols) {
		o.status = code
		o.wroteFinal = true
	}
}

func (o *observingWrapper) Write(b []byte) (int, error) {
	if !o.wroteFinal {
		o.WriteHeader(http.StatusOK)
	}
	n, err := o.ResponseWriter.Write(b)
	o.written += int64(n)
	return n, err
}

// Unwrap returns the underlying ResponseWriter, allowing wrappers further
// downstream (such as http.ResponseController) to reach it.
func (o *observingWrapper) Unwrap() http.ResponseWriter {
	return o.ResponseWriter
}

type flushingObservingWrapper struct {
	*observingWrapper
	flusher http.Flusher
}

func (o *flushingObservingWrapper) Flush() {
	if !o.wroteFinal {
		o.WriteHeader(http.StatusOK)
	}
	o.flusher.Flush()
}
