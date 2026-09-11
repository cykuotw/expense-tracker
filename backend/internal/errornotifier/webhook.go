package errornotifier

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	discordHTTPTimeout   = 5 * time.Second
	maxResponseBodyBytes = 8 * 1024
)

var webhookPathPattern = regexp.MustCompile(`^/api/webhooks/[0-9]+/[A-Za-z0-9._-]{20,}$`)

type WebhookProvider struct {
	url *url.URL
}

func NewWebhookProvider(value string) (*WebhookProvider, error) {
	webhook, err := parseDiscordWebhookURL(value)
	if err != nil {
		return nil, err
	}
	return &WebhookProvider{url: webhook}, nil
}

func (p *WebhookProvider) URL(_ context.Context) (*url.URL, error) {
	result := *p.url
	return &result, nil
}

func parseDiscordWebhookURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host != "discord.com" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || !webhookPathPattern.MatchString(parsed.EscapedPath()) {
		return nil, permanent("validate Discord webhook URL")
	}
	return parsed, nil
}

type DiscordClient struct {
	client *http.Client
}

func NewDiscordClient(client *http.Client) (*DiscordClient, error) {
	if client == nil {
		return nil, permanent("configure Discord client")
	}
	return &DiscordClient{client: client}, nil
}

func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: discordHTTPTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (c *DiscordClient) Post(ctx context.Context, webhook *url.URL, message string) error {
	if webhook == nil || len(message) == 0 || len(message) > maxDiscordMessageBytes {
		return permanent("validate Discord request")
	}
	validated, err := parseDiscordWebhookURL(webhook.String())
	if err != nil {
		return permanent("validate Discord request")
	}
	body, err := json.Marshal(struct {
		Content         string `json:"content"`
		AllowedMentions struct {
			Parse []string `json:"parse"`
		} `json:"allowed_mentions"`
	}{Content: message, AllowedMentions: struct {
		Parse []string `json:"parse"`
	}{Parse: []string{}}})
	if err != nil {
		return permanent("build Discord request")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, validated.String(), bytes.NewReader(body))
	if err != nil {
		return permanent("build Discord request")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return retryable("send Discord request")
	}
	if response == nil || response.Body == nil {
		return retryable("read Discord response")
	}
	defer response.Body.Close()

	read, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if readErr != nil {
		return retryable("read Discord response")
	}
	if read > maxResponseBodyBytes {
		return permanent("validate Discord response")
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		return retryable("Discord response")
	}
	return permanent("Discord response")
}
