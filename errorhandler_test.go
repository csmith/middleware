package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorHandler_NoErrorHandlerRegistered(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})

	handler := ErrorHandler(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "not found", rr.Body.String())
}

func TestErrorHandler_WithCustomErrorHandlers(t *testing.T) {
	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("custom 404"))
	})

	serverErrorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("custom 500"))
	})

	tests := []struct {
		name           string
		statusCode     int
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "404 error",
			statusCode:     http.StatusNotFound,
			expectedBody:   "custom 404",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "500 error",
			statusCode:     http.StatusInternalServerError,
			expectedBody:   "custom 500",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "unhandled error",
			statusCode:     http.StatusBadRequest,
			expectedBody:   "bad request",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("bad request"))
			})

			handler := ErrorHandler(nextHandler,
				WithErrorHandler(http.StatusNotFound, notFoundHandler),
				WithErrorHandler(http.StatusInternalServerError, serverErrorHandler))

			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestErrorHandler_HeadersClearedByDefault(t *testing.T) {
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Error", "error-page")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("custom 404"))
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Original-Header", "original-value")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("default response"))
	})

	handler := ErrorHandler(nextHandler, WithErrorHandler(http.StatusNotFound, errorHandler))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "custom 404", rr.Body.String())
	assert.Equal(t, "error-page", rr.Header().Get("X-Custom-Error"))
	assert.Equal(t, "", rr.Header().Get("X-Original-Header"))
	assert.Equal(t, "", rr.Header().Get("Content-Type"))
}

func TestErrorHandler_HeadersNotClearedWhenDisabled(t *testing.T) {
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Error", "error-page")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("custom 404"))
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Original-Header", "original-value")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("default response"))
	})

	handler := ErrorHandler(nextHandler,
		WithErrorHandler(http.StatusNotFound, errorHandler),
		WithClearHeadersOnError(false))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "custom 404", rr.Body.String())
	assert.Equal(t, "error-page", rr.Header().Get("X-Custom-Error"))
	assert.Equal(t, "original-value", rr.Header().Get("X-Original-Header"))
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}
