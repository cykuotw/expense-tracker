package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"net/url"
)

func WithFixtureAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		if !isLocalFixtureRequest(request) {
			http.Error(writer, "local fixture requests only", http.StatusForbidden)
			return
		}
		origin := request.Header.Get("Origin")
		if origin != "" && origin != "http://"+request.Host {
			http.Error(writer, "origin rejected", http.StatusForbidden)
			return
		}
		next(writer, request)
	}
}

// Keep the unauthenticated fixture boundary separate from chat/stream handling.
func isLocalFixtureRequest(request *http.Request) bool {
	peer, _, err := net.SplitHostPort(request.RemoteAddr)
	peerIP, parseError := netip.ParseAddr(peer)
	if err != nil || parseError != nil || !peerIP.IsLoopback() {
		return false
	}

	host := ""
	if parsed, err := url.Parse("http://" + request.Host); err == nil {
		host = parsed.Hostname()
	}
	hostIP, err := netip.ParseAddr(host)
	if host != "localhost" && (err != nil || !hostIP.IsLoopback()) {
		return false
	}

	for _, header := range []string{"Authorization", "Cookie", "X-Actor-ID", "X-User-ID"} {
		if request.Header.Get(header) != "" {
			return false
		}
	}
	return true
}
