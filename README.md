# Go HTTP middleware

Contains a collection of generally useful middlewares for use with Go's built-in
`net/http` server.

## Middleware

### Error Handler

Handles HTTP status codes by invoking custom handlers. When a registered status
code is returned by the next handler, its response is dropped and replaced with
the custom handler's response.

```go
package main

import (
	"net/http"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()

	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Custom 404 page"))
	})

	serverErrorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Custom 500 page"))
	})

	handler := middleware.ErrorHandler(
		mux,
		// Add one more handlers for specific status codes
		middleware.WithErrorHandler(http.StatusNotFound, notFoundHandler),
		middleware.WithErrorHandler(http.StatusInternalServerError, serverErrorHandler),
		// If you want to preserve headers set by the original handler
		middleware.WithClearHeadersOnError(false),
	)

	http.ListenAndServe(":8080", handler)
}
```

### Real Address

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

### Recover

Recovers from downstream panics by logging them and returning a 500 error to
the client.

```go
package main

import (
	"log/slog"
	"net/http"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()

	// With default options
	http.ListenAndServe(":8080", middleware.Recover(mux))

	// With custom logger
	http.ListenAndServe(":8080", middleware.Recover(mux, middleware.WithPanicLogger(func(r *http.Request, err any) {
		slog.Error("Panic serving request", "err", err, "url", r.URL)
	})))
}
```

### Text Log

Logs details of each request in either Common Log Format or Combined Log Format.

```go
package main

import (
	"net/http"
	"os"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()

	// With default options (Common Log Format to stdout)
	http.ListenAndServe(":8080", middleware.TextLog(mux))

	// With Combined Log Format
	http.ListenAndServe(":8080", middleware.TextLog(mux, middleware.WithTextLogFormat(middleware.TextLogFormatCombined)))

	// With custom sink
	file, _ := os.OpenFile("access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	http.ListenAndServe(":8080", middleware.TextLog(mux, middleware.WithTextLogSink(func(line string) {
		file.WriteString(line + "\n")
	})))
}
```

## Issues/Contributing/etc

Bug reports, feature requests, and pull requests are all welcome.