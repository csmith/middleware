package middleware

import (
	"net/http"
	"strconv"
)

type hstsConfig struct {
	maxAge            int
	includeSubDomains bool
	preload           bool
	reverseProxy      bool
}

type HSTSOption func(*hstsConfig)

// WithHSTSMaxAge sets the max-age (in seconds) for the HSTS header. If not set,
// a default of 1 year (31536000 seconds) is used.
func WithHSTSMaxAge(maxAge int) HSTSOption {
	return func(config *hstsConfig) {
		config.maxAge = maxAge
	}
}

// WithHSTSIncludeSubDomains adds the includeSubDomains directive to the HSTS
// header, which instructs the browser to apply the policy to all subdomains.
func WithHSTSIncludeSubDomains() HSTSOption {
	return func(config *hstsConfig) {
		config.includeSubDomains = true
	}
}

// WithHSTSPreload adds the preload directive to the HSTS header, which signals
// that the site should be included in browser HSTS preload lists.
func WithHSTSPreload() HSTSOption {
	return func(config *hstsConfig) {
		config.preload = true
	}
}

// WithHSTSReverseProxy enables support for running behind a reverse proxy. When
// enabled, HSTS headers will be set if the request was made over HTTPS either
// directly or via a reverse proxy that set the X-Forwarded-Proto header to "https".
func WithHSTSReverseProxy() HSTSOption {
	return func(config *hstsConfig) {
		config.reverseProxy = true
	}
}

// HSTS is a middleware that adds a Strict-Transport-Security (HSTS) header to
// responses.
//
// By default the header is only added when the request was made directly over
// HTTPS . Use WithHSTSReverseProxy to also set the header when running behind
// a reverse proxy that provides the X-Forwarded-Proto header.
func HSTS(opts ...HSTSOption) func(http.Handler) http.Handler {
	config := &hstsConfig{
		maxAge: 31536000,
	}
	for _, opt := range opts {
		opt(config)
	}

	value := "max-age=" + strconv.Itoa(config.maxAge)
	if config.includeSubDomains {
		value += "; includeSubDomains"
	}
	if config.preload {
		value += "; preload"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS != nil || (config.reverseProxy && r.Header.Get("X-Forwarded-Proto") == "https") {
				w.Header().Set("Strict-Transport-Security", value)
			}
			next.ServeHTTP(w, r)
		})
	}
}
