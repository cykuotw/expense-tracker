// Package ocrusage stores content-free per-attempt OCR accounting independently
// of expense confirmation and receipt retention.
package ocrusage

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"regexp"
	"slices"
	"time"
	"uuid"
)

const EventType = "ocr_usage.v1"

type Event struct {
	Type  string `json:"type"`
	Usage Record `json:"usage"`
}

type Record struct {
	RequestID        string    `json:"requestId"`
	AccountID        string    `json:"accountId"`
	Model            string    `json:"model"`
	Outcome          string    `json:"outcome"`
	UsageStatus      string    `json:"usageStatus"`
	CostStatus       string    `json:"costStatus"`
	PricingVersion   string    `json:"pricingVersion"`
	EstimatedCostUSD *float64  `json:"estimatedCostUsd"`
	Tokens           *Tokens   `json:"tokens"`
	HTTPStatus       int       `json:"httpStatus"`
	LatencyMillis    int64     `json:"latencyMillis"`
	RecordedAt       time.Time `json:"recordedAt"`
}

type Tokens struct {
	Input       int64 `json:"input"`
	Output      int64 `json:"output"`
	Thinking    int64 `json:"thinking"`
	CachedInput int64 `json:"cachedInput"`
	ToolInput   int64 `json:"toolInput"`
	Total       int64 `json:"total"`
}

var validModel = regexp.MustCompile(`^gemini-[a-zA-Z0-9.-]{1,80}$`)
var validVersion = regexp.MustCompile(`^[a-zA-Z0-9.-]{1,100}$`)

func (r Record) Validate() error {
	invalid := errors.New("invalid OCR usage record")
	for _, id := range []string{r.RequestID, r.AccountID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == (uuid.UUID{}) {
			return invalid
		}
	}
	if !validModel.MatchString(r.Model) || r.RecordedAt.IsZero() || r.LatencyMillis < 0 ||
		r.HTTPStatus != 0 && (r.HTTPStatus < 100 || r.HTTPStatus > 599) ||
		!slices.Contains([]string{"success", "request_failed", "http_error", "response_rejected"}, r.Outcome) ||
		!slices.Contains([]string{"available", "unavailable", "missing", "invalid"}, r.UsageStatus) ||
		!slices.Contains([]string{"estimated", "usage_unavailable", "pricing_unavailable", "unsupported_usage"}, r.CostStatus) {
		return invalid
	}
	if (r.Tokens != nil) != (r.UsageStatus == "available") {
		return invalid
	}
	if u := r.Tokens; u != nil {
		for _, count := range []int64{u.Input, u.Output, u.Thinking, u.CachedInput, u.ToolInput, u.Total} {
			if count < 0 || count > 1_000_000_000 {
				return invalid
			}
		}
		if u.CachedInput > u.Input || u.Input+u.Output+u.Thinking+u.ToolInput != u.Total {
			return invalid
		}
	}
	if r.CostStatus == "estimated" {
		if r.Tokens == nil || r.EstimatedCostUSD == nil || !validVersion.MatchString(r.PricingVersion) {
			return invalid
		}
		if cost := *r.EstimatedCostUSD; math.IsNaN(cost) || math.IsInf(cost, 0) || cost < 0 || cost > 1_000_000 {
			return invalid
		}
	} else if r.EstimatedCostUSD != nil || r.PricingVersion != "" {
		return invalid
	}
	return nil
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Save is immutable and idempotent because Lambda's async delivery is at least once.
func (s *Store) Save(ctx context.Context, r Record) error {
	if err := r.Validate(); err != nil {
		return err
	}
	var input, output, thinking, cached, tool, total any
	if u := r.Tokens; u != nil {
		input, output, thinking, cached, tool, total = u.Input, u.Output, u.Thinking, u.CachedInput, u.ToolInput, u.Total
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO ocr_usage
		(request_id,account_id,model,outcome,usage_status,cost_status,pricing_version,
		estimated_cost_usd,input_tokens,output_tokens,thinking_tokens,cached_input_tokens,
		tool_input_tokens,total_tokens,http_status,latency_ms,recorded_at)
		VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT(request_id) DO NOTHING`, r.RequestID, r.AccountID, r.Model, r.Outcome,
		r.UsageStatus, r.CostStatus, r.PricingVersion, r.EstimatedCostUSD, input, output, thinking, cached,
		tool, total, r.HTTPStatus, r.LatencyMillis, r.RecordedAt)
	return err
}
