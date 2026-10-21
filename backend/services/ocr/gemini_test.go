package ocr

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleExtraction = `{"merchant":"Shoppers Drug Mart","date":"2022-03-18","currencySuggestion":"CAD","subtotal":"5.39","tax":"","tip":"","total":"5.39","items":[{"description":"NEILSON MILK 2","quantity":"","unitPrice":"","lineTotal":"5.39"}]}`

func TestGeminiStrictExtractionAndProviderTransport(t *testing.T) {
	envelope := func(output, finish string) string {
		encoded, _ := json.Marshal(output)
		return fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"text":%s}]},"finishReason":%q}],"modelVersion":"test-model"}`, encoded, finish)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.URL.RawQuery)
		require.Equal(t, "test-secret", r.Header.Get("x-goog-api-key"))
		body, _ := io.ReadAll(r.Body)
		require.NotContains(t, string(body), "test-secret")
		var request map[string]any
		require.NoError(t, json.Unmarshal(body, &request))
		require.Contains(t, request, "systemInstruction")
		require.Contains(t, request["generationConfig"].(map[string]any), "responseJsonSchema")
		_, _ = w.Write([]byte(envelope(sampleExtraction, "STOP")))
	}))
	defer server.Close()
	provider, err := NewGeminiProvider("test-secret", DefaultGeminiModel)
	require.NoError(t, err)
	provider.endpoint = server.URL
	result, err := provider.AnalyzeExpense(t.Context(), []byte("normalized JPEG"))
	require.NoError(t, err)
	require.Equal(t, "Shoppers Drug Mart", result.Merchant.Value)
	require.Zero(t, result.Merchant.Confidence)
	require.NotNil(t, result.Evidence)
	_,err=ReplayEvidence(*result.Evidence);require.NoError(t,err)
	changed:=*result.Evidence;changed.SchemaDigest="unknown-schema";_,err=ReplayEvidence(changed);require.ErrorIs(t,err,ErrProviderRejected)
	draft := PrepareDraft(result)
	require.Equal(t, ProvenanceInferred, draft.Tax.Provenance)
	require.Equal(t, "0", draft.Tax.Value)
	require.Empty(t, draft.Items[0].Quantity.Value)
	_, _, err = ReplayGeminiResponse([]byte(envelope(sampleExtraction, "MAX_TOKENS")))
	require.ErrorIs(t, err, ErrProviderRejected)
}

func TestGeminiRejectsInvalidOrUnboundedOutput(t *testing.T) {
	for name, raw := range map[string]string{
		"extra":        strings.Replace(sampleExtraction, `"merchant":`, `"confidence":99,"merchant":`, 1),
		"duplicate":    strings.Replace(sampleExtraction, `"merchant":`, `"merchant":"other","merchant":`, 1),
		"null":         strings.Replace(sampleExtraction, `"tax":""`, `"tax":null`, 1),
		"missing":      strings.Replace(sampleExtraction, `"tax":"",`, "", 1),
		"invalid date": strings.Replace(sampleExtraction, "2022-03-18", "2022-02-30", 1),
		"bad currency": strings.Replace(sampleExtraction, `"CAD"`, `"ZZZ"`, 1),
		"amount":       strings.Replace(sampleExtraction, `"5.39"`, `"5e3"`, 1),
		"truncated":    sampleExtraction[:len(sampleExtraction)-1],
		"trailing":     sampleExtraction + "{}",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseGeminiExtraction([]byte(raw))
			require.ErrorIs(t, err, ErrProviderRejected)
		})
	}
	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(sampleExtraction), &data))
	rows := make([]any, 201)
	for i := range rows {
		rows[i] = data["items"].([]any)[0]
	}
	data["items"] = rows
	raw, _ := json.Marshal(data)
	_, err := ParseGeminiExtraction(raw)
	require.ErrorIs(t, err, ErrProviderRejected)
}
