package localonly

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalBoundary(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8000", "[::1]:8000"} {
		require.NoError(t, CheckAddress(address))
	}
	for _, address := range []string{"0.0.0.0:8000", ":8000", "localhost:8000", "example.test:8000", "192.168.1.1:8000", "[::1%eth0]:8000"} {
		require.Error(t, CheckAddress(address))
	}
	request := httptest.NewRequest("GET", "http://127.0.0.1:8000/", nil)
	request.RemoteAddr = "127.0.0.1:9000"
	require.True(t, RequestAllowed(request))
	request.RemoteAddr = "192.0.2.1:9000"
	require.False(t, RequestAllowed(request))
	request.RemoteAddr = "127.0.0.1:9000"
	request.Host = "foreign.test:8000"
	require.False(t, RequestAllowed(request))
	request.Host = "127.0.0.1:8000"
	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"} {
		request.Header.Set(header, "synthetic")
		require.False(t, RequestAllowed(request))
		request.Header.Del(header)
	}
	request.Header.Set("Origin", "https://foreign.test")
	require.False(t, RequestAllowed(request))
	request.Header.Set("Origin", "http://127.0.0.1:8000")
	require.True(t, RequestAllowed(request))
}

func TestRuntimeIsolation(t *testing.T) {
	for _, key := range []string{"AWS_LAMBDA_RUNTIME_API", "AWS_LAMBDA_FUNCTION_NAME", "AWS_EXECUTION_ENV", "MODE"} {
		t.Run(key, func(t *testing.T) {
			value := "synthetic"
			if key == "AWS_EXECUTION_ENV" {
				value = "AWS_Lambda_go"
			}
			if key == "MODE" {
				value = "release"
			}
			t.Setenv(key, value)
			require.Error(t, CheckRuntime())
		})
	}
}
