package middleware

import (
	"net"
	"net/http"
	"strings"
)

type realAddressConfig struct {
	trustedProxies []net.IPNet
}

var defaultTrustedProxies = []net.IPNet{
	mustParseCIDR("10.0.0.0/8"),
	mustParseCIDR("127.0.0.0/8"),
	mustParseCIDR("172.16.0.0/12"),
	mustParseCIDR("192.168.0.0/16"),
	mustParseCIDR("::1/128"),
	mustParseCIDR("fc00::/7"),
}

type RealAddressOption func(*realAddressConfig)

// WithTrustedProxies configures the IP ranges that RealAddress will accept
// X-Forwarded-For hops from.
func WithTrustedProxies(trustedProxies []net.IPNet) RealAddressOption {
	return func(config *realAddressConfig) {
		config.trustedProxies = trustedProxies
	}
}

// RealAddress is a middleware that sets the RemoteAddr property on the http.Request
// to the client's real IP address according to the X-Forwarded-For header.
//
// By default, only proxies on private IP addresses will be trusted. If you need to
// trust other addresses, use the WithTrustedProxies option.
func RealAddress(next http.Handler, opts ...RealAddressOption) http.Handler {
	conf := realAddressConfig{
		trustedProxies: defaultTrustedProxies,
	}
	for _, opt := range opts {
		opt(&conf)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = selectRealAddress(collateForwardedHops(r), conf.trustedProxies)

		next.ServeHTTP(w, r)
	})
}

func collateForwardedHops(r *http.Request) []string {
	var res []string
	values := r.Header.Values("X-Forwarded-For")
	for _, v := range values {
		hops := strings.Split(v, ",")
		for i := range hops {
			res = append(res, strings.TrimSpace(hops[i]))
		}
	}
	res = append(res, r.RemoteAddr)
	return res
}

func selectRealAddress(hops []string, trustedProxies []net.IPNet) string {
	for i := len(hops) - 1; i >= 0; i-- {
		trusted := false
		ip := parseAddress(hops[i])
		if ip == nil {
			// If we can't parse the address at all, return the last good address.
			// In theory this can panic if the first hop is bad, but that will always
			// come from RemoteAddr.
			return hops[i+1]
		}

		for j := range trustedProxies {
			if trustedProxies[j].Contains(ip) {
				trusted = true
				break
			}
		}

		if !trusted {
			return hops[i]
		}
	}

	// Everything in the chain was trusted for some reason, just return the closest IP to the client
	return hops[0]
}

func parseAddress(address string) net.IP {
	if host, _, err := net.SplitHostPort(address); err == nil {
		return net.ParseIP(host)
	}
	return net.ParseIP(address)
}

func mustParseCIDR(cidr string) net.IPNet {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return *ipnet
}
