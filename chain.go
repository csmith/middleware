package middleware

import "net/http"

type chainConfig struct {
	middleware []func(http.Handler) http.Handler
}

type ChainOption func(*chainConfig)

// WithMiddleware appends one or more middleware to the chain.
func WithMiddleware(middleware ...func(http.Handler) http.Handler) ChainOption {
	return func(conf *chainConfig) {
		conf.middleware = append(conf.middleware, middleware...)
	}
}

// Chain is a middleware that chains together other middlewares (i.e., invokes
// them in order). Add middlewares using the WithMiddleware option.
//
// Middlewares are invoked in the order supplied, i.e., the first one passed
// will be invoked with the upstream http.Handler (it will be the innermost),
// then the second one will be given the result of the first middleware
// (wrapping it), and so on.
//
// i.e., a chain of `A`, `B`, and `C` is equivalent to `C(B(A(next)))`.
func Chain(opts ...ChainOption) func(http.Handler) http.Handler {
	conf := &chainConfig{}
	for _, opt := range opts {
		opt(conf)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			for _, m := range conf.middleware {
				next = m(next)
			}
			next.ServeHTTP(w, req)
		})
	}
}
