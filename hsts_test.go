package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHSTS_DefaultTLS(t *testing.T) {
	handler := HSTS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "max-age=31536000", rr.Header().Get("Strict-Transport-Security"))
}

func TestHSTS_NoTLS(t *testing.T) {
	handler := HSTS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Strict-Transport-Security"))
}

func TestHSTS_CustomMaxAge(t *testing.T) {
	handler := HSTS(WithHSTSMaxAge(3600))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=3600", rr.Header().Get("Strict-Transport-Security"))
}

func TestHSTS_IncludeSubDomains(t *testing.T) {
	handler := HSTS(WithHSTSIncludeSubDomains())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=31536000; includeSubDomains", rr.Header().Get("Strict-Transport-Security"))
}

func TestHSTS_Preload(t *testing.T) {
	handler := HSTS(WithHSTSPreload())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=31536000; preload", rr.Header().Get("Strict-Transport-Security"))
}

func TestHSTS_AllOptions(t *testing.T) {
	handler := HSTS(
		WithHSTSMaxAge(86400),
		WithHSTSIncludeSubDomains(),
		WithHSTSPreload(),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=86400; includeSubDomains; preload", rr.Header().Get("Strict-Transport-Security"))
}

func TestHSTS_ReverseProxy(t *testing.T) {
	handler := HSTS(WithHSTSReverseProxy())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "max-age=31536000", rr.Header().Get("Strict-Transport-Security"))
}

func TestHSTS_ReverseProxyNotHTTPS(t *testing.T) {
	handler := HSTS(WithHSTSReverseProxy())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "http")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Strict-Transport-Security"))
}

func TestHSTS_ReverseProxyHeaderWithoutOption(t *testing.T) {
	handler := HSTS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Strict-Transport-Security"))
}
