package middleware

import "net/http"

// CrossOriginProtection is a middleware that denies unsafe requests that
// originated from a different origin, to defend against CSRF attacks.
//
// GET, HEAD and OPTIONS requests are always allowed. Any other request has
// its Sec-Fetch-Site header verified. If present, it must either be
// "same-origin" or "none" for the request to proceed.
//
// Denied requests are responded to with a 403 response with no body.
// Chain this middleware with ErrorHandler to customise this.
func CrossOriginProtection() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			secFetchHeader := r.Header.Get("Sec-Fetch-Site")
			if secFetchHeader != "same-origin" && secFetchHeader != "none" && secFetchHeader != "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
