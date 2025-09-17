package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRealAddress(t *testing.T) {
	tests := []struct {
		name           string
		trustedProxies []string
		headers        []string
		remoteAddr     string
		expectedAddr   string
	}{
		{
			name:         "no forwarded headers",
			remoteAddr:   "192.168.1.100:8080",
			expectedAddr: "192.168.1.100:8080",
		},
		{
			name:         "single forwarded header from trusted proxy",
			headers:      []string{"203.0.113.1"},
			remoteAddr:   "192.168.1.1:8080",
			expectedAddr: "203.0.113.1",
		},
		{
			name:         "multiple forwarded headers from trusted proxies",
			headers:      []string{"203.0.113.1, 192.168.1.2"},
			remoteAddr:   "192.168.1.1:8080",
			expectedAddr: "203.0.113.1",
		},
		{
			name:         "forwarded header from untrusted proxy",
			headers:      []string{"203.0.113.1"},
			remoteAddr:   "203.0.113.50:8080",
			expectedAddr: "203.0.113.50:8080",
		},
		{
			name:         "chain with trusted and untrusted proxies, with multiple headers",
			headers:      []string{"203.0.113.1, 198.51.100.1", "192.168.1.100"},
			remoteAddr:   "192.168.1.1:8080",
			expectedAddr: "198.51.100.1",
		},
		{
			name:         "chain with trusted and untrusted proxies, with single header",
			headers:      []string{"203.0.113.1,198.51.100.1,192.168.1.100"},
			remoteAddr:   "192.168.1.1:8080",
			expectedAddr: "198.51.100.1",
		},
		{
			name:         "all hops trusted",
			headers:      []string{"192.168.1.2, 192.168.1.5"},
			remoteAddr:   "192.168.1.1:8080",
			expectedAddr: "192.168.1.2",
		},
		{
			name:         "IPv6 addresses",
			headers:      []string{"2001:db8::1"},
			remoteAddr:   "[::1]:8080",
			expectedAddr: "2001:db8::1",
		},
		{
			name:           "custom trusted proxies - trusted upstream",
			trustedProxies: []string{"203.0.113.0/24", "198.51.100.0/24"},
			headers:        []string{"10.0.0.1"},
			remoteAddr:     "203.0.113.50:8080",
			expectedAddr:   "10.0.0.1",
		},
		{
			name:           "custom trusted proxies - untrusted upstream",
			trustedProxies: []string{"203.0.113.0/24"},
			headers:        []string{"203.0.113.1"},
			remoteAddr:     "198.51.100.50:8080",
			expectedAddr:   "198.51.100.50:8080",
		},
		{
			name:         "invalid IP in chain",
			headers:      []string{"192.168.1.10, invalid-ip, 192.168.1.100"},
			remoteAddr:   "192.168.1.1:8080",
			expectedAddr: "192.168.1.100",
		},
		{
			name:         "defaults to trusting loopback addresses",
			headers:      []string{"203.0.113.1"},
			remoteAddr:   "127.0.0.1:8080",
			expectedAddr: "203.0.113.1",
		},
		{
			name:         "defaults to trusting 10.x.x.x",
			headers:      []string{"203.0.113.1"},
			remoteAddr:   "10.0.0.1:8080",
			expectedAddr: "203.0.113.1",
		},
		{
			name:         "defaults to trusting 172.16.x.x",
			headers:      []string{"203.0.113.1"},
			remoteAddr:   "172.16.0.1:8080",
			expectedAddr: "203.0.113.1",
		},
		{
			name:         "defaults to trusting IPv6 loopback",
			headers:      []string{"203.0.113.1"},
			remoteAddr:   "[::1]:8080",
			expectedAddr: "203.0.113.1",
		},
		{
			name:         "empty X-Forwarded-For header",
			headers:      []string{""},
			remoteAddr:   "192.168.1.1:8080",
			expectedAddr: "192.168.1.1:8080",
		},
		{
			name:         "only commas in X-Forwarded-For",
			headers:      []string{",,"},
			remoteAddr:   "192.168.1.1:8080",
			expectedAddr: "192.168.1.1:8080",
		},
		{
			name:           "empty trusted proxies list",
			trustedProxies: []string{},
			headers:        []string{"203.0.113.1"},
			remoteAddr:     "192.168.1.1:8080",
			expectedAddr:   "192.168.1.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []RealAddressOption
			if tt.trustedProxies != nil {
				var trustedNets []net.IPNet
				for _, cidr := range tt.trustedProxies {
					trustedNets = append(trustedNets, mustParseCIDR(cidr))
				}
				opts = append(opts, WithTrustedProxies(trustedNets))
			}

			var actualAddr string
			handler := RealAddress(opts...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				actualAddr = r.RemoteAddr
			}))

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			for _, header := range tt.headers {
				req.Header.Add("X-Forwarded-For", header)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedAddr, actualAddr)
		})
	}
}
