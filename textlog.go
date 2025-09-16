package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type TextLogFormat int

const (
	// TextLogFormatCommon is the "Common Log Format" as used by Apache
	TextLogFormatCommon TextLogFormat = iota
	// TextLogFormatCombined is the "Combined Log Format" as used by Apache and Nginx
	TextLogFormatCombined
)

type textLogConfig struct {
	sink   func(string)
	format TextLogFormat
	clock  func() time.Time
}

type TextLogOption func(*textLogConfig)

// WithTextLogSink specifies where logs should be written to by TextLog.
func WithTextLogSink(sink func(string)) TextLogOption {
	return func(config *textLogConfig) {
		config.sink = sink
	}
}

// WithTextLogFormat specifies the log format used by TextLog.
func WithTextLogFormat(format TextLogFormat) TextLogOption {
	return func(config *textLogConfig) {
		config.format = format
	}
}

// TextLog logs details of each request in a textual format.
//
// By default each request will be logged to stdout in the 'common' log format.
// Use WithTextLogSink to handle the log lines differently, and
// WithTextLogFormat to change the format.
func TextLog(next http.Handler, opts ...TextLogOption) http.Handler {
	conf := &textLogConfig{
		sink: func(s string) {
			fmt.Printf(s)
		},
		format: TextLogFormatCommon,
		clock:  time.Now,
	}

	for _, opt := range opts {
		opt(conf)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := wrap(w)
		start := conf.clock()
		next.ServeHTTP(wrapped, r)
		conf.sink(formatTextLog(conf.format, r, wrapped.status, wrapped.written, start))
	})
}

func formatTextLog(format TextLogFormat, r *http.Request, status int, written int, start time.Time) string {
	switch format {
	case TextLogFormatCommon:
		address := r.RemoteAddr
		if ip, _, err := net.SplitHostPort(address); err == nil {
			address = ip
		}
		return fmt.Sprintf(
			`%s - - %s "%s %s %s" %d %d`,
			address,
			start.Format("[02/Jan/2006:15:04:05 -0700]"),
			escapeLogValue(r.Method),
			escapeLogValue(r.URL.String()),
			escapeLogValue(r.Proto),
			status,
			written,
		)

	case TextLogFormatCombined:
		return fmt.Sprintf(
			`%s "%s" "%s"`,
			formatTextLog(TextLogFormatCommon, r, status, written, start),
			escapeLogValue(r.Referer()),
			escapeLogValue(r.UserAgent()),
		)

	default:
		return fmt.Sprintf("Unknown text log format: %d", format)
	}
}

func escapeLogValue(s string) string {
	var result strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			result.WriteString(`\"`)
		case '\\':
			result.WriteString(`\\`)
		case '\n':
			result.WriteString(`\n`)
		case '\t':
			result.WriteString(`\t`)
		case '\r':
			result.WriteString(`\r`)
		case '\v':
			result.WriteString(`\v`)
		case '\f':
			result.WriteString(`\f`)
		case '\b':
			result.WriteString(`\b`)
		case '\a':
			result.WriteString(`\a`)
		default:
			if r < 32 || r > 126 {
				result.WriteString(fmt.Sprintf(`\x%02x`, r))
			} else {
				result.WriteRune(r)
			}
		}
	}
	return result.String()
}
