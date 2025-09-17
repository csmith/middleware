package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrossOriginProtection(t *testing.T) {
	handler := CrossOriginProtection()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name           string
		method         string
		secFetchSite   string
		expectedStatus int
		expectedBody   string
	}{
		// Safe methods are always allowed
		{"GET request", "GET", "", http.StatusOK, "success"},
		{"HEAD request", "HEAD", "", http.StatusOK, "success"},
		{"OPTIONS request", "OPTIONS", "", http.StatusOK, "success"},
		{"GET with cross-site", "GET", "cross-site", http.StatusOK, "success"},
		{"HEAD with same-site", "HEAD", "same-site", http.StatusOK, "success"},
		{"OPTIONS with same-origin", "OPTIONS", "same-origin", http.StatusOK, "success"},

		// Unsafe methods with valid Sec-Fetch-Site
		{"POST with same-origin", "POST", "same-origin", http.StatusOK, "success"},
		{"PUT with same-origin", "PUT", "same-origin", http.StatusOK, "success"},
		{"DELETE with same-origin", "DELETE", "same-origin", http.StatusOK, "success"},
		{"PATCH with same-origin", "PATCH", "same-origin", http.StatusOK, "success"},
		{"POST with none", "POST", "none", http.StatusOK, "success"},
		{"PUT with none", "PUT", "none", http.StatusOK, "success"},

		// Unsafe methods without Sec-Fetch-Site header (allowed)
		{"POST without header", "POST", "", http.StatusOK, "success"},
		{"PUT without header", "PUT", "", http.StatusOK, "success"},
		{"DELETE without header", "DELETE", "", http.StatusOK, "success"},

		// Unsafe methods with invalid Sec-Fetch-Site
		{"POST with cross-site", "POST", "cross-site", http.StatusForbidden, ""},
		{"PUT with cross-site", "PUT", "cross-site", http.StatusForbidden, ""},
		{"DELETE with cross-site", "DELETE", "cross-site", http.StatusForbidden, ""},
		{"PATCH with cross-site", "PATCH", "cross-site", http.StatusForbidden, ""},
		{"POST with same-site", "POST", "same-site", http.StatusForbidden, ""},
		{"PUT with same-site", "PUT", "same-site", http.StatusForbidden, ""},
		{"CONNECT with cross-site", "CONNECT", "cross-site", http.StatusForbidden, ""},
		{"TRACE with same-site", "TRACE", "same-site", http.StatusForbidden, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tt.secFetchSite)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}
