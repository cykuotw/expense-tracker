// Package localonly constrains development adapters to explicit local execution.
package localonly

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
)

func CheckRuntime() error {
	for _, key := range []string{"AWS_LAMBDA_RUNTIME_API", "AWS_LAMBDA_FUNCTION_NAME"} {
		if _, present := os.LookupEnv(key); present {
			return fmt.Errorf("local mode is unavailable in Lambda")
		}
	}
	if strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") || os.Getenv("MODE") == "release" {
		return fmt.Errorf("local mode is unavailable in Lambda or release mode")
	}
	return nil
}

// Literal loopback listeners/endpoints avoid DNS and wildcard exposure.
func CheckAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	ip, parseErr := netip.ParseAddr(host)
	if err != nil || parseErr != nil || !ip.IsLoopback() || ip.Zone() != "" || port == "" {
		return fmt.Errorf("local mode requires a literal loopback address and port")
	}
	return nil
}

func RequestAllowed(request *http.Request) bool {
	if CheckRuntime() != nil {
		return false
	}
	peer, _, err := net.SplitHostPort(request.RemoteAddr)
	ip, parseErr := netip.ParseAddr(peer)
	if err != nil || parseErr != nil || !ip.IsLoopback() {
		return false
	}
	if CheckAddress(request.Host) != nil {
		return false
	}
	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"} {
		if len(request.Header.Values(header)) != 0 {
			return false
		}
	}
	origin := request.Header.Get("Origin")
	return origin == "" || origin == "http://"+request.Host
}
