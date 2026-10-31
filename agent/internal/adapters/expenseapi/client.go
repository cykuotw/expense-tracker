// Package expenseapi adapts Agent operations to bounded backend Worker calls without placing credentials
// in tool arguments, session state, model content, or returned errors.
package expenseapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uuid"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"
	"expense-tracker/internal/agentcontract"
	"expense-tracker/internal/localonly"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

type Client struct {
	endpoint    *url.URL
	region      string
	credentials aws.CredentialsProvider
	http        *http.Client
	local       bool
}

// New requires HTTPS and explicit AWS credential retrieval (normally the
// temporary runtime-role provider). Tests inject synthetic providers/transports.
// No insecure/local authentication mode exists in the production client.
func New(endpoint, region string, credentials aws.CredentialsProvider, transport http.RoundTripper) (*Client, error) {
	base, err := url.Parse(endpoint)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || base.RawPath != "" || region == "" || credentials == nil {
		return nil, &models.Error{Code: models.Validation}
	}
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{endpoint: base, region: region, credentials: credentials, http: &http.Client{
		Transport: transport, Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (client *Client) Preflight(ctx context.Context) (agentcontract.Preflight, error) {
	var result agentcontract.Preflight
	if err := client.get(ctx, agentcontract.PreflightPath, &result); err != nil {
		return agentcontract.Preflight{}, err
	}
	if _, err := uuid.Parse(result.ActorID); err != nil || !result.ExpiresAt.After(time.Now()) {
		return agentcontract.Preflight{}, &models.Error{Code: models.Unauthenticated}
	}
	return result, nil
}

// Authenticate resolves the request credential independently of domain tools.
func (client *Client) Authenticate(ctx context.Context) (models.Identity, error) {
	actor, err := client.Preflight(ctx)
	if err != nil {
		return models.Identity{}, err
	}
	return models.Identity{ID: actor.ActorID, ExpiresAt: actor.ExpiresAt}, nil
}

func (client *Client) GetBalance(ctx context.Context, groupID string) (getbalance.Result, error) {
	var result getbalance.Result
	if _, err := uuid.Parse(groupID); err != nil {
		return result, &models.Error{Code: models.Validation}
	}
	err := client.get(ctx, agentcontract.Prefix+"/balance/"+groupID, &result)
	if err != nil {
		return getbalance.Result{}, err
	}
	if result.GroupID != groupID || len(result.Balances) > agentcontract.MaxBalanceRows {
		return getbalance.Result{}, &models.Error{Code: models.Internal}
	}
	return result, nil
}

func (client *Client) get(ctx context.Context, path string, result any) error {
	token := credentials.AccessToken(ctx)
	if token == "" || len(token) > 8192 || strings.ContainsAny(token, "\r\n") {
		return &models.Error{Code: models.Unauthenticated}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	requestURL := client.endpoint.Clone()
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") + path
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return &models.Error{Code: models.Unavailable}
	}
	request.Header.Set(agentcontract.AccessTokenHeader, token)
	request.Header.Set("Accept", "application/json")
	if client.local {
		if localonly.CheckRuntime() != nil {
			return &models.Error{Code: models.Forbidden}
		}
	} else {
		credentials, err := client.credentials.Retrieve(ctx)
		if err != nil {
			return &models.Error{Code: models.Unavailable}
		}
		hash := sha256.Sum256(nil)
		err = v4.NewSigner().SignHTTP(ctx, credentials, request, hex.EncodeToString(hash[:]), "execute-api", client.region, time.Now().UTC())
		if err != nil {
			return &models.Error{Code: models.Unavailable}
		}
	}
	response, err := client.http.Do(request)
	if err != nil {
		return &models.Error{Code: models.Unavailable}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		code := models.Unavailable
		switch response.StatusCode {
		case 400:
			code = models.Validation
		case 401:
			code = models.Unauthenticated
		case 403:
			code = models.Forbidden
		case 404:
			code = models.NotFound
		case 422, 429:
			code = models.Limit
		}
		return &models.Error{Code: code}
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, agentcontract.MaxResponseBytes+1))
	if err != nil || len(payload) > agentcontract.MaxResponseBytes || json.Unmarshal(payload, result, json.RejectUnknownMembers(true)) != nil {
		return &models.Error{Code: models.Internal}
	}
	return nil
}
