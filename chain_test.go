package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChain_NoMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("original"))
	})

	handler := Chain()(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "original", rr.Body.String())
}

func TestChain_SingleMiddleware(t *testing.T) {
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-1", "applied")
			next.ServeHTTP(w, r)
		})
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("original"))
	})

	handler := Chain(WithMiddleware(middleware1))(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "original", rr.Body.String())
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware-1"))
}

func TestChain_MultipleWithMiddlewareCalls(t *testing.T) {
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-1", "applied")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-2", "applied")
			next.ServeHTTP(w, r)
		})
	}

	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-3", "applied")
			next.ServeHTTP(w, r)
		})
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("original"))
	})

	// Test multiple WithMiddleware calls
	handler := Chain(
		WithMiddleware(middleware1),
		WithMiddleware(middleware2, middleware3),
	)(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "original", rr.Body.String())
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware-1"))
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware-2"))
	assert.Equal(t, "applied", rr.Header().Get("X-Middleware-3"))
}

func TestChain_MiddlewareExecutionOrder(t *testing.T) {
	var order []int

	createMiddleware := func(id int) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, id)
				next.ServeHTTP(w, r)
			})
		}
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, 0)
		w.WriteHeader(http.StatusOK)
	})

	handler := Chain(WithMiddleware(
		createMiddleware(1),
		createMiddleware(2),
		createMiddleware(3),
	))(nextHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expectedOrder := []int{3, 2, 1, 0}
	assert.Equal(t, expectedOrder, order)
}
