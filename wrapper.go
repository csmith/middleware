package middleware

import "net/http"

type responseWriterWrapper struct {
	http.ResponseWriter
	written int
	status  int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += n
	return n, err
}

func wrap(rw http.ResponseWriter) *responseWriterWrapper {
	if w, ok := rw.(*responseWriterWrapper); ok {
		return w
	}
	return &responseWriterWrapper{ResponseWriter: rw}
}
