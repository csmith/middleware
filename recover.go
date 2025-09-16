package middleware

import (
	"log"
	"net/http"
)

type RecoverPanicLogger func(r *http.Request, err any)

type recoverConfig struct {
	logger RecoverPanicLogger
}

type RecoverOption func(*recoverConfig)

// WithPanicLogger configures the logger that Recover will use to log the
// details of the panic.
func WithPanicLogger(logger RecoverPanicLogger) RecoverOption {
	return func(config *recoverConfig) {
		config.logger = logger
	}
}

// Recover is a middleware that will recover from downstream panics, log the
// error, and send a 500 response to the client.
func Recover(next http.Handler, opts ...RecoverOption) http.Handler {
	config := &recoverConfig{logger: defaultPanicLogger}
	for _, opt := range opts {
		opt(config)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				config.logger(r, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func defaultPanicLogger(r *http.Request, err any) {
	log.Printf("panic recovered: %v", err)
}
