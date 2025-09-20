# Go HTTP middleware

Contains a collection of generally useful middlewares for use with Go's built-in
`net/http` server.

## Middleware

### Cache Control

Automatically sets a `Cache-Control` header with a max-age based on the
`Content-Type` header. By default, static assets like images, videos, and
downloads get a max-age of 1 year, while text assets like HTML and CSS get a
max-age of 1 hour.

```go
package main

import (
	"net/http"
	"time"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()

	// With default cache times
	http.ListenAndServe(":8080", middleware.CacheControl()(mux))

	// With custom cache times
	http.ListenAndServe(":8080", middleware.CacheControl(middleware.WithCacheTimes(map[string]time.Duration{
		"application/json": time.Duration(0),
		"image/*":          time.Hour * 24,
		"text/css":         time.Hour * 12,
	}))(mux))
}
```

### Compress

Automatically compresses the response body if the client accepts gzip encoding.
Supports configurable compression levels and handles Accept-Encoding headers
with quality values.

```go
package main

import (
	"compress/gzip"
	"net/http"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()

	// With default compression level
	http.ListenAndServe(":8080", middleware.Compress()(mux))

	// With custom compression level
	http.ListenAndServe(":8080", middleware.Compress(middleware.WithGzipLevel(gzip.BestSpeed))(mux))
	
	// With additional custom logic for disabling compression on certain requests 
	http.ListenAndServe(":8080", middleware.Compress(middleware.WithCompressionCheck(func(r *http.Request) bool {
		return r.URL.Path != "/special"
	}))(mux))
}
```

### Chain

Allows you to chain other middleware together, without directly chaining the
function calls.

```go
package main

import (
	"net/http"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()
	notFoundHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
        // ...
	})

	chain := middleware.Chain(middleware.WithMiddleware(
		// Innermost
		middleware.CacheControl(),
		middleware.ErrorHandler(middleware.WithErrorHandler(http.StatusNotFound, notFoundHandler)),
		middleware.TextLog(middleware.WithTextLogFormat(middleware.TextLogFormatCombined)),
		middleware.Recover(),
		// Outermost
    ))
	http.ListenAndServe(":8080", chain(mux))
}
```

### Cross Origin Protection

Defends against CSRF attacks by denying unsafe requests that originated from a
different origin. GET, HEAD and OPTIONS requests are always allowed. Any other
request has its `Sec-Fetch-Site` header verified. If present, it must either be
`same-origin` or `none` for the request to proceed.

Denied requests are responded to with a 403 response with no body. Chain this
middleware with Error Handler to customise this.

```go
package main

import (
	"net/http"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()

	http.ListenAndServe(":8080", middleware.CrossOriginProtection()(mux))
}
```

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
		// Add one more handlers for specific status codes
		middleware.WithErrorHandler(http.StatusNotFound, notFoundHandler),
		middleware.WithErrorHandler(http.StatusInternalServerError, serverErrorHandler),
		// If you want to preserve headers set by the original handler
		middleware.WithClearHeadersOnError(false),
	)(mux)

	http.ListenAndServe(":8080", handler)
}
```

### Headers

Adds headers to a response as late as possible. This may be useful when chained
with other middleware such as ErrorHandler that change headers.

```go
package main

import (
	"net/http"

	"github.com/csmith/middleware"
)

func main() {
	mux := http.NewServeMux()

	handler := middleware.Headers(
		middleware.WithHeader("X-Frame-Options", "DENY"),
		middleware.WithHeader("X-Content-Type-Options", "nosniff"),
		middleware.WithHeader("Cache-Control", "no-cache"),
		middleware.WithHeader("Cache-Control", "no-store"), // Multiple values for same key
	)(mux)

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
	http.ListenAndServe(":8080", middleware.RealAddress()(mux))

	// With custom trusted proxies
	var trustedProxies []net.IPNet // Populate appropriately
	http.ListenAndServe(":8080", middleware.RealAddress(middleware.WithTrustedProxies(trustedProxies))(mux))
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
	http.ListenAndServe(":8080", middleware.Recover()(mux))

	// With custom logger
	http.ListenAndServe(":8080", middleware.Recover(middleware.WithPanicLogger(func(r *http.Request, err any) {
		slog.Error("Panic serving request", "err", err, "url", r.URL)
	}))(mux))
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
	http.ListenAndServe(":8080", middleware.TextLog()(mux))

	// With Combined Log Format
	http.ListenAndServe(":8080", middleware.TextLog(middleware.WithTextLogFormat(middleware.TextLogFormatCombined))(mux))

	// With custom sink
	file, _ := os.OpenFile("access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	http.ListenAndServe(":8080", middleware.TextLog(middleware.WithTextLogSink(func(line string) {
		file.WriteString(line + "\n")
	}))(mux))
}
```

## Issues/Contributing/etc

Bug reports, feature requests, and pull requests are all welcome.