package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type unwrappableResponseWriter interface {
	Unwrap() http.ResponseWriter
}

// nonFlushingWriter hides the Flusher implementation of the wrapped writer.
type nonFlushingWriter struct {
	http.ResponseWriter
}

func observeRequests(observer RequestObserver) func(http.Handler) http.Handler {
	return ObserveRequests(WithRequestObserver(observer))
}

func TestObserveRequests_WithoutObserverIsNoop(t *testing.T) {
	var received http.ResponseWriter
	handler := ObserveRequests()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = w
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/test", nil))

	assert.Same(t, rr, received)
}

func TestObserveRequests_ExplicitStatus(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, "created", rr.Body.String())
	assert.Len(t, results, 1)
	assert.Equal(t, http.StatusCreated, results[0].StatusCode)
	assert.Equal(t, int64(7), results[0].BytesWritten)
}

func TestObserveRequests_DefaultStatusOnWrite(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("implicit"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Len(t, results, 1)
	assert.Equal(t, http.StatusOK, results[0].StatusCode)
	assert.Equal(t, int64(8), results[0].BytesWritten)
}

func TestObserveRequests_DefaultStatusOnEmptyHandler(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Writes nothing at all
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Len(t, results, 1)
	assert.Equal(t, http.StatusOK, results[0].StatusCode)
	assert.Equal(t, int64(0), results[0].BytesWritten)
}

func TestObserveRequests_FirstFinalStatusCaptured(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.WriteHeader(http.StatusTeapot)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("body"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Len(t, results, 1)
	assert.Equal(t, http.StatusAccepted, results[0].StatusCode)
}

func TestObserveRequests_InformationalThenFinalStatus(t *testing.T) {
	var result RequestResult
	writer := newInformationalWriter()

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		result = res
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(writer, httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, []int{http.StatusEarlyHints, http.StatusNoContent}, writer.statuses)
	assert.Equal(t, http.StatusNoContent, result.StatusCode)
}

func TestObserveRequests_InformationalThenWrite(t *testing.T) {
	var result RequestResult
	writer := newInformationalWriter()

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		result = res
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.Write([]byte("body"))
	}))

	handler.ServeHTTP(writer, httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, []int{http.StatusEarlyHints, http.StatusOK}, writer.statuses)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, int64(4), result.BytesWritten)
}

func TestObserveRequests_InformationalThenPanicHasNoFinalStatus(t *testing.T) {
	var result RequestResult
	writer := newInformationalWriter()

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		result = res
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		panic("boom")
	}))

	assert.PanicsWithValue(t, "boom", func() {
		handler.ServeHTTP(writer, httptest.NewRequest("GET", "/test", nil))
	})

	assert.Equal(t, []int{http.StatusEarlyHints}, writer.statuses)
	assert.Equal(t, 0, result.StatusCode)
}

func TestObserveRequests_SwitchingProtocolsIsFinal(t *testing.T) {
	var result RequestResult
	writer := newInformationalWriter()

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		result = res
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusSwitchingProtocols)
	}))

	handler.ServeHTTP(writer, httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, http.StatusSwitchingProtocols, result.StatusCode)
}

func TestObserveRequests_MultipleWritesAccumulate(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello "))
		w.Write([]byte("world"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "hello world", rr.Body.String())
	assert.Len(t, results, 1)
	assert.Equal(t, http.StatusOK, results[0].StatusCode)
	assert.Equal(t, int64(11), results[0].BytesWritten)
}

func TestObserveRequests_ShortAndErroringWrites(t *testing.T) {
	var results []RequestResult
	var handlerErr error

	underlying := httptest.NewRecorder()
	limited := &limitedWriter{ResponseWriter: underlying, remaining: 4}

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, handlerErr = w.Write([]byte("0123456789"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)

	handler.ServeHTTP(limited, req)

	assert.True(t, errors.Is(handlerErr, io.ErrShortWrite))
	assert.Equal(t, "0123", underlying.Body.String())
	assert.Len(t, results, 1)
	assert.Equal(t, int64(4), results[0].BytesWritten)
}

func TestObserveRequests_ErroredWriteCountsNothing(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("unwritten"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(&failingWriter{ResponseWriter: rr}, req)

	assert.Len(t, results, 1)
	assert.Equal(t, http.StatusOK, results[0].StatusCode)
	assert.Equal(t, int64(0), results[0].BytesWritten)
	assert.Equal(t, "", rr.Body.String())
}

func TestObserveRequests_PanicBeforeResponse(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	assert.PanicsWithValue(t, "boom", func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Len(t, results, 1)
	assert.Equal(t, 0, results[0].StatusCode)
	assert.Equal(t, int64(0), results[0].BytesWritten)
}

func TestObserveRequests_PanicAfterResponseCommitted(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("partial"))
		panic("boom")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	assert.PanicsWithValue(t, "boom", func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Len(t, results, 1)
	// The response was already committed, so its status is the observed outcome.
	assert.Equal(t, http.StatusAccepted, results[0].StatusCode)
	assert.Equal(t, int64(7), results[0].BytesWritten)
}

func TestObserveRequests_RecoveredPanicReportsRecoveryStatus(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(Recover()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Recover swallows the panic, so nothing propagates to us.
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Len(t, results, 1)
	assert.Equal(t, http.StatusInternalServerError, results[0].StatusCode)
}

func TestObserveRequests_ObserverInvokedExactlyOnce(t *testing.T) {
	calls := 0

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		calls++
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, 1, calls)
}

func TestObserveRequests_ObserverInvokedExactlyOnceOnPanic(t *testing.T) {
	calls := 0

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		calls++
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	assert.PanicsWithValue(t, "boom", func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Equal(t, 1, calls)
}

func TestObserveRequests_RequestIdentity(t *testing.T) {
	var handlerRequest *http.Request
	var observedRequest *http.Request

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		observedRequest = r
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerRequest = r
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("POST", "/test?foo=bar", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Same(t, req, handlerRequest)
	assert.Same(t, req, observedRequest)
}

func TestObserveRequests_DurationRecorded(t *testing.T) {
	var results []RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		results = append(results, res)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Len(t, results, 1)
	assert.GreaterOrEqual(t, results[0].Duration, 15*time.Millisecond)
}

func TestObserveRequests_Unwrap(t *testing.T) {
	var innerWriter http.ResponseWriter
	handler := observeRequests(func(r *http.Request, res RequestResult) {})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			innerWriter = w
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	unwrap, ok := innerWriter.(unwrappableResponseWriter)
	assert.True(t, ok, "wrapper should implement Unwrap() http.ResponseWriter")
	assert.Same(t, rr, unwrap.Unwrap())
}

func TestObserveRequests_FlushForwarded(t *testing.T) {
	handler := observeRequests(func(r *http.Request, res RequestResult) {})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("first"))
			w.(http.Flusher).Flush()
			w.Write([]byte("second"))
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "firstsecond", rr.Body.String())
	assert.True(t, rr.Flushed)
}

func TestObserveRequests_FlushCommitsStatus(t *testing.T) {
	var result RequestResult

	handler := observeRequests(func(r *http.Request, res RequestResult) {
		result = res
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		panic("boom")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	assert.PanicsWithValue(t, "boom", func() {
		handler.ServeHTTP(rr, req)
	})
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestObserveRequests_DoesNotExposeUnsupportedFlusher(t *testing.T) {
	handler := observeRequests(func(r *http.Request, res RequestResult) {})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, ok := w.(http.Flusher)
			assert.False(t, ok)
			w.Write([]byte("body"))
		}),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(&nonFlushingWriter{ResponseWriter: rr}, req)

	assert.Equal(t, "body", rr.Body.String())
}

type informationalWriter struct {
	header   http.Header
	statuses []int
	body     []byte
	final    bool
}

func newInformationalWriter() *informationalWriter {
	return &informationalWriter{header: make(http.Header)}
}

func (w *informationalWriter) Header() http.Header {
	return w.header
}

func (w *informationalWriter) WriteHeader(code int) {
	if w.final {
		return
	}
	w.statuses = append(w.statuses, code)
	if code >= http.StatusOK || code == http.StatusSwitchingProtocols {
		w.final = true
	}
}

func (w *informationalWriter) Write(p []byte) (int, error) {
	if !w.final {
		w.WriteHeader(http.StatusOK)
	}
	w.body = append(w.body, p...)
	return len(p), nil
}

type limitedWriter struct {
	http.ResponseWriter
	remaining int
}

func (l *limitedWriter) Write(b []byte) (int, error) {
	if len(b) > l.remaining {
		n, _ := l.ResponseWriter.Write(b[:l.remaining])
		l.remaining = 0
		return n, io.ErrShortWrite
	}
	l.remaining -= len(b)
	return l.ResponseWriter.Write(b)
}

type failingWriter struct {
	http.ResponseWriter
}

func (f *failingWriter) Write(b []byte) (int, error) {
	return 0, errors.New("write failed")
}
