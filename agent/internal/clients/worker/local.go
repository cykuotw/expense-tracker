package worker

import (
	"net/http"
	"net/url"
	"time"

	"expense-tracker/agent/internal/tools"
	"expense-tracker/internal/localonly"
)

// NewLocal explicitly selects unsigned loopback transport, retaining the same
// JWT forwarding, response bounds, deadlines and errors as the signed client.
func NewLocal(endpoint string) (*Client, error) {
	if err := localonly.CheckRuntime(); err != nil {
		return nil, err
	}
	base, err := url.Parse(endpoint)
	if err != nil || base.Scheme != "http" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || base.RawPath != "" || localonly.CheckAddress(base.Host) != nil {
		return nil, &tools.Error{Code: tools.Validation}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &Client{endpoint: base, local: true, http: &http.Client{
		Transport: transport, Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}
