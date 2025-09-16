# Go HTTP middleware

Contains a collection of generally useful middlewares for use with Go's built-in
`net/http` server.

## Real Address

Gets the real address of the client by parsing `X-Forwarded-For` headers from
trusted proxies. By default only private IP ranges are trusted. Sets the
`RemoteAddr` field of the request to the first untrusted hop (or the last hop
if they're all trusted).

```go
package main

import (
	"net"
	"net/http"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()

	// With default options
	http.ListenAndServe(":8080", middleware.RealAddress(mux))

	// With custom trusted proxies
	var trustedProxies []net.IPNet // Populate appropriately
	http.ListenAndServe(":8080", middleware.RealAddress(mux, middleware.WithTrustedProxies(trustedProxies)))
}
```