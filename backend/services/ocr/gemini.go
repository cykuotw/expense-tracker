package ocr

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const DefaultGeminiModel = "gemini-3.8-flash"
const GeminiAdapterVersion = "gemini-v1"

//go:embed gemini_prompt.txt
var GeminiPrompt string

//go:embed gemini_schema.json
var GeminiSchema json.RawMessage

var modelName = regexp.MustCompile(`^gemini-[a-zA-Z0-9.-]{1,80}$`)
var exactDecimal = regexp.MustCompile(`^-?[0-9]{1,12}(?:\.[0-9]{1,6})?$`)

type ExtractionEvidence struct {
	Provider       string          `json:"provider"`
	Model          string          `json:"model"`
	ModelVersion   string          `json:"modelVersion,omitempty"`
	PromptDigest   string          `json:"promptDigest"`
	SchemaDigest   string          `json:"schemaDigest"`
	AdapterVersion string          `json:"adapterVersion"`
	Response       json.RawMessage `json:"response"`
}

func contentDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type GeminiProvider struct {
	client               *http.Client
	key, model, endpoint string
}

func NewGeminiProvider(key, model string) (*GeminiProvider, error) {
	if strings.TrimSpace(key) == "" || strings.HasPrefix(strings.ToUpper(key), "REPLACE") || !modelName.MatchString(model) {
		return nil, errors.New("Gemini API key and valid model must be configured")
	}
	return &GeminiProvider{
		key: key, model: model,
		endpoint: "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":generateContent",
		client:   &http.Client{Timeout: 18 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

func (p *GeminiProvider) AnalyzeExpense(ctx context.Context, document []byte) (Result, error) {
	request := map[string]any{
		"systemInstruction": map[string]any{"parts": []any{map[string]any{"text": GeminiPrompt}}},
		"contents":          []any{map[string]any{"role": "user", "parts": []any{map[string]any{"inlineData": map[string]any{"mimeType": "image/jpeg", "data": base64.StdEncoding.EncodeToString(document)}}}}},
		"generationConfig":  map[string]any{"responseMimeType": "application/json", "responseJsonSchema": GeminiSchema, "maxOutputTokens": 16000},
	}
	body, err := json.Marshal(request)
	if err != nil {
		return Result{}, ErrProviderRequestFailed
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, ErrProviderRequestFailed
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.key)
	response, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		return Result{}, ErrProviderRequestFailed
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return Result{}, ErrProviderThrottled
	}
	if response.StatusCode != http.StatusOK {
		return Result{}, ErrProviderRequestFailed
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil || len(raw) > 2*1024*1024 {
		return Result{}, ErrProviderRejected
	}
	result, version, err := ReplayGeminiResponse(raw)
	if err != nil {
		return Result{}, err
	}
	result.Evidence = &ExtractionEvidence{Provider: "gemini", Model: p.model, ModelVersion: version,
		PromptDigest: contentDigest([]byte(GeminiPrompt)), SchemaDigest: contentDigest(GeminiSchema), AdapterVersion: GeminiAdapterVersion, Response: raw}
	return result, nil
}

func ReplayEvidence(evidence ExtractionEvidence) (Result, error) {
	if evidence.Provider != "gemini" || evidence.AdapterVersion != GeminiAdapterVersion || evidence.SchemaDigest != contentDigest(GeminiSchema) {
		return Result{}, ErrProviderRejected
	}
	result, _, err := ReplayGeminiResponse(evidence.Response)
	return result, err
}

// ReplayGeminiResponse validates saved output locally. It never invokes a model.
func ReplayGeminiResponse(raw []byte) (Result, string, error) {
	var envelope struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		ModelVersion string `json:"modelVersion"`
	}
	if len(raw) > 2*1024*1024 || json.Unmarshal(raw, &envelope) != nil || len(envelope.Candidates) != 1 || envelope.Candidates[0].FinishReason != "STOP" {
		return Result{}, "", ErrProviderRejected
	}
	var output strings.Builder
	for _, part := range envelope.Candidates[0].Content.Parts {
		if !part.Thought {
			output.WriteString(part.Text)
		}
	}
	result, err := ParseGeminiExtraction([]byte(output.String()))
	return result, envelope.ModelVersion, err
}

func ParseGeminiExtraction(raw []byte) (Result, error) {
	if len(raw) > 1024*1024 {
		return Result{}, ErrProviderRejected
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	value, err := uniqueJSONValue(decoder, 0)
	if err != nil {
		return Result{}, ErrProviderRejected
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Result{}, ErrProviderRejected
	}
	var schema map[string]any
	if json.Unmarshal(GeminiSchema, &schema) != nil || !matchesSchema(value, schema) {
		return Result{}, ErrProviderRejected
	}
	fields := value.(map[string]any)
	field := func(key string) Field { return Field{Value: fields[key].(string)} }
	result := Result{Merchant: field("merchant"), Date: field("date"), Currency: field("currencySuggestion"), Subtotal: field("subtotal"), Tax: field("tax"), Tip: field("tip"), Total: field("total"), Items: []Item{}}
	if result.Date.Value != "" {
		if _, err := time.Parse("2006-01-02", result.Date.Value); err != nil {
			return Result{}, ErrProviderRejected
		}
	}
	if result.Currency.Value != "" && !strings.Contains(" CAD USD TWD EUR GBP JPY CNY HKD KRW SGD AUD NZD CHF THB MYR PHP IDR VND INR MXN ", " "+result.Currency.Value+" ") {
		return Result{}, ErrProviderRejected
	}
	amounts := []Field{result.Subtotal, result.Tax, result.Tip, result.Total}
	for _, rawItem := range fields["items"].([]any) {
		item := rawItem.(map[string]any)
		row := Item{Description: Field{Value: item["description"].(string)}, Quantity: Field{Value: item["quantity"].(string)}, UnitPrice: Field{Value: item["unitPrice"].(string)}, LineTotal: Field{Value: item["lineTotal"].(string)}}
		amounts = append(amounts, row.Quantity, row.UnitPrice, row.LineTotal)
		result.Items = append(result.Items, row)
	}
	for _, field := range amounts {
		if field.Value != "" && !exactDecimal.MatchString(field.Value) {
			return Result{}, ErrProviderRejected
		}
	}
	return result, nil
}

func matchesSchema(value any, schema map[string]any) bool {
	switch schema["type"] {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return false
		}
		properties := schema["properties"].(map[string]any)
		if len(object) != len(properties) {
			return false
		}
		for key, property := range properties {
			v, ok := object[key]
			if !ok || !matchesSchema(v, property.(map[string]any)) {
				return false
			}
		}
		return true
	case "array":
		array, ok := value.([]any)
		if !ok || len(array) > 200 {
			return false
		}
		for _, item := range array {
			if !matchesSchema(item, schema["items"].(map[string]any)) {
				return false
			}
		}
		return true
	case "string":
		s, ok := value.(string)
		return ok && len(s) <= 2048
	}
	return false
}

// Reject duplicate keys as well as excessive nesting before schema validation.
func uniqueJSONValue(decoder *json.Decoder, depth int) (any, error) {
	if depth > 8 {
		return nil, ErrProviderRejected
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delim {
	case '{':
		result := map[string]any{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, ErrProviderRejected
			}
			if _, exists := result[key]; exists {
				return nil, ErrProviderRejected
			}
			v, err := uniqueJSONValue(decoder, depth+1)
			if err != nil {
				return nil, err
			}
			result[key] = v
		}
		_, err := decoder.Token()
		return result, err
	case '[':
		result := []any{}
		for decoder.More() {
			if len(result) >= 200 {
				return nil, ErrProviderRejected
			}
			v, err := uniqueJSONValue(decoder, depth+1)
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		}
		_, err := decoder.Token()
		return result, err
	}
	return nil, ErrProviderRejected
}
