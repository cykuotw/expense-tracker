package expenseapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"expense-tracker/agent/internal/fixtures"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"
	"expense-tracker/internal/agentcontract"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/stretchr/testify/require"
)

const groupID = "22222222-2222-4222-8222-222222222222"
const actorID = "11111111-1111-4111-8111-111111111111"

var syntheticCredentials = aws.Credentials{AccessKeyID: "SYNTHETICKEY", SecretAccessKey: "synthetic-secret", SessionToken: "synthetic-session"}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func provideSyntheticCredentials(context.Context) (aws.Credentials, error) {
	return syntheticCredentials, nil
}
func reply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func newClient(t *testing.T, transport transportFunc) *Client {
	t.Helper()
	client, err := New("https://worker.example.test/api", "ca-central-1", aws.CredentialsProviderFunc(provideSyntheticCredentials), transport)
	require.NoError(t, err)
	return client
}

func TestSignedConstructionAndRequestScopedJWT(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}
	client := newClient(t, func(request *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		access := request.Header.Get(agentcontract.AccessTokenHeader)
		seen[access] = true
		require.Equal(t, "GET", request.Method)
		require.Equal(t, "/api"+agentcontract.PreflightPath, request.URL.Path)
		require.Empty(t, request.URL.RawQuery)
		require.Empty(t, request.Header.Get("Cookie"))
		require.Empty(t, request.Header.Get("X-Actor-ID"))
		require.Equal(t, syntheticCredentials.SessionToken, request.Header.Get("X-Amz-Security-Token"))
		signed := request.Header.Get("Authorization")
		require.Contains(t, signed, "/ca-central-1/execute-api/aws4_request")
		require.Contains(t, signed, "x-agent-access-token")
		date, err := time.Parse("20060102T150405Z", request.Header.Get("X-Amz-Date"))
		require.NoError(t, err)
		hash := sha256.Sum256(nil)
		copy := request.Clone(request.Context())
		copy.Header.Del("Authorization")
		require.NoError(t, v4.NewSigner().SignHTTP(request.Context(), syntheticCredentials, copy, hex.EncodeToString(hash[:]), "execute-api", "ca-central-1", date))
		require.Equal(t, signed, copy.Header.Get("Authorization"))
		copy.Header.Set(agentcontract.AccessTokenHeader, "forged-token")
		require.NoError(t, v4.NewSigner().SignHTTP(request.Context(), syntheticCredentials, copy, hex.EncodeToString(hash[:]), "execute-api", "ca-central-1", date))
		require.NotEqual(t, signed, copy.Header.Get("Authorization"))
		return reply(200, fmt.Sprintf(`{"actorId":%q,"expiresAt":%q}`, actorID, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))), nil
	})
	var wg sync.WaitGroup
	for _, value := range []string{"first-access-token", "second-access-token"} {
		wg.Go(func() {
			_, err := client.Preflight(credentials.WithAccessToken(t.Context(), value))
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	require.Len(t, seen, 2)
	_, err := client.Preflight(t.Context())
	require.Equal(t, models.Unauthenticated, models.PublicCode(err))
	require.Len(t, seen, 2)
}

func TestBoundedFailuresAndNoRedirects(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		code   models.Code
	}{
		{"unauthorized", 401, "private JWT detail", models.Unauthenticated},
		{"forbidden", 403, "private principal detail", models.Forbidden},
		{"not found", 404, "private group", models.NotFound},
		{"rate limit", 429, "private response", models.Limit},
		{"server failure", 500, "private SQL", models.Unavailable},
		{"malformed", 200, "not json", models.Internal},
		{"oversize", 200, strings.Repeat(" ", agentcontract.MaxResponseBytes+1), models.Internal},
		{"unknown field", 200, `{"actorId":"` + actorID + `","accessToken":"private"}`, models.Internal},
		{"expired preflight", 200, `{"actorId":"` + actorID + `","expiresAt":"2000-01-01T00:00:00Z"}`, models.Unauthenticated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newClient(t, func(*http.Request) (*http.Response, error) { return reply(tc.status, tc.body), nil })
			_, err := client.Preflight(credentials.WithAccessToken(t.Context(), "private-token"))
			require.Equal(t, tc.code, models.PublicCode(err))
			require.NotContains(t, err.Error(), "private")
		})
	}
	calls := 0
	client := newClient(t, func(*http.Request) (*http.Response, error) {
		calls++
		response := reply(302, "")
		response.Header.Set("Location", "https://foreign.example.test")
		return response, nil
	})
	_, err := client.Preflight(credentials.WithAccessToken(t.Context(), "private-token"))
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

func TestClientValidationAndBalanceContract(t *testing.T) {
	for _, endpoint := range []string{"http://localhost", "https://user:pass@example.test", "https://example.test?token=x", "https://example.test#x", "://broken"} {
		_, err := New(endpoint, "region", aws.CredentialsProviderFunc(provideSyntheticCredentials), nil)
		require.Error(t, err)
	}
	client := newClient(t, func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/api"+agentcontract.Prefix+"/balance/"+groupID, r.URL.Path)
		return reply(200, `{"groupId":"`+groupID+`","balances":[{"from":"Alice","to":"Bob","amount":"12.34","currency":"CAD"}]}`), nil
	})
	ctx := llm.WithRunBudget(credentials.WithAccessToken(t.Context(), "request-only-access"), fixtures.Limits())
	result, err := getbalance.Fetch(getbalance.WithGroup(ctx, groupID), client, getbalance.Input{GroupID: groupID})
	require.NoError(t, err)
	require.Equal(t, "12.34", result.Balances[0].Amount)
	for _, invalid := range []string{"../preflight", "group?actorId=forged", "demo"} {
		_, err = client.GetBalance(ctx, invalid)
		require.Equal(t, models.Validation, models.PublicCode(err))
	}
}
