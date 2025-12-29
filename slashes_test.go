package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripTrailingSlashes_WithTrailingSlash(t *testing.T) {
	handler := StripTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}))

	req := httptest.NewRequest("GET", "/test/path/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "/test/path", rr.Body.String())
}

func TestStripTrailingSlashes_WithoutTrailingSlash(t *testing.T) {
	handler := StripTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}))

	req := httptest.NewRequest("GET", "/test/path", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "/test/path", rr.Body.String())
}

func TestStripTrailingSlashes_RootPath(t *testing.T) {
	handler := StripTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "/", rr.Body.String())
}

func TestStripTrailingSlashes_QueryParams(t *testing.T) {
	handler := StripTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path + "?" + r.URL.RawQuery))
	}))

	req := httptest.NewRequest("GET", "/test/path/?foo=bar&baz=qux", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "/test/path?foo=bar&baz=qux", rr.Body.String())
}

func TestRedirectTrailingSlashes_WithoutTrailingSlash(t *testing.T) {
	handler := RedirectTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test/path", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusPermanentRedirect, rr.Code)
	assert.Equal(t, "/test/path/", rr.Header().Get("Location"))
}

func TestRedirectTrailingSlashes_ithTrailingSlash(t *testing.T) {
	handler := RedirectTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}))

	req := httptest.NewRequest("GET", "/test/path/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "/test/path/", rr.Body.String())
}

func TestRedirectTrailingSlashes_RootPath(t *testing.T) {
	handler := RedirectTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "/", rr.Body.String())
}

func TestRedirectTrailingSlashes_QueryParams(t *testing.T) {
	handler := RedirectTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test/path?foo=bar&baz=qux", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusPermanentRedirect, rr.Code)
	assert.Equal(t, "/test/path/?foo=bar&baz=qux", rr.Header().Get("Location"))
}

func TestRedirectTrailingSlashes_DefaultRedirectCode(t *testing.T) {
	handler := RedirectTrailingSlashes()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusPermanentRedirect, rr.Code)
	assert.Equal(t, "/test/", rr.Header().Get("Location"))
}

func TestRedirectTrailingSlashes_CustomRedirectCode(t *testing.T) {
	handler := RedirectTrailingSlashes(WithRedirectCode(http.StatusMovedPermanently))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "/test/", rr.Header().Get("Location"))
}
