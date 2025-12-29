package middleware

import (
	"net/http"
	"strings"
)

type redirectTrailingSlashesConfig struct {
	redirectCode int
}

type RedirectTrailingSlashesOption func(*redirectTrailingSlashesConfig)

// WithRedirectCode sets the HTTP status code to use for redirects.
// Defaults to 308 Permanent Redirect if not specified.
func WithRedirectCode(code int) RedirectTrailingSlashesOption {
	return func(config *redirectTrailingSlashesConfig) {
		config.redirectCode = code
	}
}

// StripTrailingSlashes is a middleware that removes trailing slashes from
// URLs.
func StripTrailingSlashes() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
				r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RedirectTrailingSlashes is a middleware that redirects URLs without trailing
// slashes to include one.
// Uses a 308 Permanent Redirect status code by default.
func RedirectTrailingSlashes(opts ...RedirectTrailingSlashesOption) func(http.Handler) http.Handler {
	config := &redirectTrailingSlashesConfig{
		redirectCode: http.StatusPermanentRedirect,
	}
	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, "/") {
				newURL := *r.URL
				newURL.Path = r.URL.Path + "/"
				http.Redirect(w, r, newURL.String(), config.redirectCode)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
