package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaders_SingleHeader(t *testing.T) {
	handler := Headers(WithHeader("X-Custom", "test-value"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "test-value", rr.Header().Get("X-Custom"))
	assert.Equal(t, "test content", rr.Body.String())
}

func TestHeaders_MultipleHeaders(t *testing.T) {
	handler := Headers(
		WithHeader("X-Custom", "value1"),
		WithHeader("X-Another", "value2"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "value1", rr.Header().Get("X-Custom"))
	assert.Equal(t, "value2", rr.Header().Get("X-Another"))
}

func TestHeaders_SameKeyMultipleValues(t *testing.T) {
	handler := Headers(
		WithHeader("X-Multiple", "value1"),
		WithHeader("X-Multiple", "value2"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	values := rr.Header().Values("X-Multiple")
	assert.Contains(t, values, "value1")
	assert.Contains(t, values, "value2")
	assert.Len(t, values, 2)
}

func TestHeaders_AppliedLate(t *testing.T) {
	handler := Headers(WithHeader("X-Custom", "middleware-value"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "handler-value")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	values := rr.Header().Values("X-Custom")
	assert.Contains(t, values, "handler-value")
	assert.Contains(t, values, "middleware-value")
	assert.Len(t, values, 2)
}

func TestHeaders_WriteWithoutWriteHeader(t *testing.T) {
	handler := Headers(WithHeader("X-Custom", "test-value"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "test-value", rr.Header().Get("X-Custom"))
	assert.Equal(t, "test content", rr.Body.String())
}
