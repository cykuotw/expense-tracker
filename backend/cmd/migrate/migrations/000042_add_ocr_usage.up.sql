CREATE TABLE ocr_usage (
    request_id UUID PRIMARY KEY,
    account_id UUID REFERENCES users(id) ON DELETE SET NULL,
    model VARCHAR(88) NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('success','request_failed','http_error','response_rejected')),
    usage_status TEXT NOT NULL CHECK (usage_status IN ('available','unavailable','missing','invalid')),
    cost_status TEXT NOT NULL CHECK (cost_status IN ('estimated','usage_unavailable','pricing_unavailable','unsupported_usage')),
    pricing_version VARCHAR(100),
    estimated_cost_usd NUMERIC(20,9) CHECK (estimated_cost_usd >= 0),
    input_tokens BIGINT CHECK (input_tokens BETWEEN 0 AND 1000000000),
    output_tokens BIGINT CHECK (output_tokens BETWEEN 0 AND 1000000000),
    thinking_tokens BIGINT CHECK (thinking_tokens BETWEEN 0 AND 1000000000),
    cached_input_tokens BIGINT CHECK (cached_input_tokens BETWEEN 0 AND 1000000000),
    tool_input_tokens BIGINT CHECK (tool_input_tokens BETWEEN 0 AND 1000000000),
    total_tokens BIGINT CHECK (total_tokens BETWEEN 0 AND 1000000000),
    http_status INTEGER NOT NULL CHECK (http_status = 0 OR http_status BETWEEN 100 AND 599),
    latency_ms BIGINT NOT NULL CHECK (latency_ms >= 0),
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (usage_status = 'available' AND input_tokens IS NOT NULL AND output_tokens IS NOT NULL
         AND thinking_tokens IS NOT NULL AND cached_input_tokens IS NOT NULL
         AND tool_input_tokens IS NOT NULL AND total_tokens IS NOT NULL
         AND cached_input_tokens <= input_tokens
         AND input_tokens + output_tokens + thinking_tokens + tool_input_tokens = total_tokens)
        OR (usage_status <> 'available' AND input_tokens IS NULL AND output_tokens IS NULL
         AND thinking_tokens IS NULL AND cached_input_tokens IS NULL
         AND tool_input_tokens IS NULL AND total_tokens IS NULL)
    ),
    CHECK ((cost_status = 'estimated' AND estimated_cost_usd IS NOT NULL AND pricing_version IS NOT NULL AND usage_status = 'available')
        OR (cost_status <> 'estimated' AND estimated_cost_usd IS NULL AND pricing_version IS NULL))
);
CREATE INDEX ocr_usage_account_time ON ocr_usage(account_id, recorded_at DESC);
