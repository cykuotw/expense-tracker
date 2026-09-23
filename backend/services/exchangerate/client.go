package exchangerate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

const (
	ProviderName   = "Frankfurter"
	AttributionURL = "https://frankfurter.dev"
)

var ErrUnavailable = errors.New("exchange-rate recommendations are unavailable")

type Recommendation struct {
	SourceCurrency            string          `json:"sourceCurrency"`
	SettlementPreviewCurrency string          `json:"settlementPreviewCurrency"`
	RecommendedRate           decimal.Decimal `json:"recommendedRate"`
	RateSnapshotAt            string          `json:"rateSnapshotAt"`
}

type Result struct {
	ProviderName    string           `json:"providerName"`
	AttributionURL  string           `json:"attributionUrl"`
	Recommendations []Recommendation `json:"recommendations"`
}

type Client interface {
	Recommendations(ctx context.Context, sourceCurrencies []string, targetCurrency string) (Result, error)
}

type cacheEntry struct {
	result    Result
	expiresAt time.Time
}

type FrankfurterClient struct {
	httpClient *http.Client
	baseURL    string
	now        func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry
}

func NewFrankfurterClient() *FrankfurterClient {
	return &FrankfurterClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    "https://api.frankfurter.dev/v2",
		now:        time.Now,
		cache:      make(map[string]cacheEntry),
	}
}

type providerRate struct {
	Date  string      `json:"date"`
	Base  string      `json:"base"`
	Quote string      `json:"quote"`
	Rate  json.Number `json:"rate"`
}

func (c *FrankfurterClient) Recommendations(ctx context.Context, sourceCurrencies []string, targetCurrency string) (Result, error) {
	sources := slices.Clone(sourceCurrencies)
	slices.Sort(sources)
	sources = slices.Compact(sources)
	cacheKey := targetCurrency + "|" + strings.Join(sources, ",")
	c.mu.Lock()
	if cached, ok := c.cache[cacheKey]; ok && c.now().Before(cached.expiresAt) {
		c.mu.Unlock()
		return cached.result, nil
	}
	c.mu.Unlock()

	query := url.Values{}
	query.Set("base", targetCurrency)
	query.Set("quotes", strings.Join(sources, ","))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/rates?"+query.Encode(), nil)
	if err != nil {
		return Result{}, fmt.Errorf("%w: create request", ErrUnavailable)
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("%w: request failed", ErrUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("%w: provider status %d", ErrUnavailable, response.StatusCode)
	}
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	var rows []providerRate
	if err := decoder.Decode(&rows); err != nil {
		return Result{}, fmt.Errorf("%w: decode response", ErrUnavailable)
	}

	bySource := make(map[string]Recommendation, len(sources))
	for _, row := range rows {
		if row.Base != targetCurrency {
			return Result{}, fmt.Errorf("%w: unexpected provider base", ErrUnavailable)
		}
		if _, err := time.Parse(time.DateOnly, row.Date); err != nil {
			return Result{}, fmt.Errorf("%w: invalid snapshot date", ErrUnavailable)
		}
		rate, err := decimal.NewFromString(row.Rate.String())
		if err != nil || !rate.IsPositive() {
			return Result{}, fmt.Errorf("%w: invalid rate", ErrUnavailable)
		}
		canonical := decimal.NewFromInt(1).DivRound(rate, 15)
		if row.Quote == targetCurrency {
			canonical = decimal.NewFromInt(1)
		}
		bySource[row.Quote] = Recommendation{
			SourceCurrency:            row.Quote,
			SettlementPreviewCurrency: targetCurrency,
			RecommendedRate:           canonical,
			RateSnapshotAt:            row.Date,
		}
	}

	result := Result{
		ProviderName:    ProviderName,
		AttributionURL:  AttributionURL,
		Recommendations: make([]Recommendation, 0, len(sources)),
	}
	for _, source := range sources {
		if source == targetCurrency {
			result.Recommendations = append(result.Recommendations, Recommendation{
				SourceCurrency: source, SettlementPreviewCurrency: targetCurrency,
				RecommendedRate: decimal.NewFromInt(1), RateSnapshotAt: latestSnapshot(rows),
			})
			continue
		}
		recommendation, ok := bySource[source]
		if !ok {
			return Result{}, fmt.Errorf("%w: missing rate for %s", ErrUnavailable, source)
		}
		result.Recommendations = append(result.Recommendations, recommendation)
	}
	c.mu.Lock()
	c.cache[cacheKey] = cacheEntry{result: result, expiresAt: c.now().Add(15 * time.Minute)}
	c.mu.Unlock()
	return result, nil
}

func latestSnapshot(rows []providerRate) string {
	latest := ""
	for _, row := range rows {
		if row.Date > latest {
			latest = row.Date
		}
	}
	return latest
}
