package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func withTestClock(t time.Time) TextLogOption {
	return func(config *textLogConfig) {
		config.clock = func() time.Time { return t }
	}
}

func TestTextLog_CommonFormat(t *testing.T) {
	var logOutput string
	sink := func(s string) {
		logOutput = s
	}

	testTime := time.Date(2000, 10, 10, 13, 55, 36, 0, time.FixedZone("PDT", -7*3600))

	handler := TextLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}), WithTextLogSink(sink), withTestClock(testTime))

	req := httptest.NewRequest("GET", "/apache_pb.gif", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Proto = "HTTP/1.0"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := `127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 13`
	assert.Equal(t, expected, logOutput)
}

func TestTextLog_CombinedFormat(t *testing.T) {
	var logOutput string
	sink := func(s string) {
		logOutput = s
	}

	testTime := time.Date(2000, 10, 10, 13, 55, 36, 0, time.FixedZone("PDT", -7*3600))

	handler := TextLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello World!"))
	}), WithTextLogSink(sink), WithTextLogFormat(TextLogFormatCombined), withTestClock(testTime))

	req := httptest.NewRequest("GET", "/apache_pb.gif", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Proto = "HTTP/1.0"
	req.Header.Set("Referer", "http://www.example.com/start.html")
	req.Header.Set("User-Agent", "Mozilla/4.08 [en] (Win98; I ;Nav)")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := `127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 12 "http://www.example.com/start.html" "Mozilla/4.08 [en] (Win98; I ;Nav)"`
	assert.Equal(t, expected, logOutput)
}

func TestTextLog_EscapingSpecialCharacters(t *testing.T) {
	var logOutput string
	sink := func(s string) {
		logOutput = s
	}

	testTime := time.Date(2000, 10, 10, 13, 55, 36, 0, time.FixedZone("PDT", -7*3600))

	handler := TextLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), WithTextLogSink(sink), WithTextLogFormat(TextLogFormatCombined), withTestClock(testTime))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Proto = "HTTP/1.1"
	req.Header.Set("Referer", "http://evil.com\"\n<script>")
	req.Header.Set("User-Agent", "BadBot\t\r")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expected := `127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] "GET /test HTTP/1.1" 200 0 "http://evil.com\"\n<script>" "BadBot\t\r"`
	assert.Equal(t, expected, logOutput)
}

func TestEscapeLogValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{`quote"test`, `quote\"test`},
		{`backslash\test`, `backslash\\test`},
		{"newline\ntest", `newline\ntest`},
		{"tab\ttest", `tab\ttest`},
		{"carriage\rreturn", `carriage\rreturn`},
		{"control\x01char", `control\x01char`},
		{"unicode\u00A0char", `unicode\xa0char`},
		{"<script>", "<script>"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeLogValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
