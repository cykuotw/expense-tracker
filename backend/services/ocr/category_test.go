package ocr

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const groceryID = "00000000-0000-4000-8000-000000000001"

func TestCategorySuggestionParsingDoesNotDiscardReceipt(t *testing.T) {
	for name, optional := range map[string]string{
		"missing":      "",
		"wrong type":   `,"suggestedExpenseTypeId":{},"categoryReason":4`,
		"too long":     `,"suggestedExpenseTypeId":"` + groceryID + `","categoryReason":"` + strings.Repeat("x", 501) + `"`,
		"empty reason": `,"suggestedExpenseTypeId":"` + groceryID + `","categoryReason":" "`,
	} {
		t.Run(name, func(t *testing.T) {
			result, err := ParseGeminiExtraction([]byte(strings.TrimSuffix(sampleExtraction, "}") + optional + "}"))
			require.NoError(t, err)
			require.Equal(t, "5.39", result.Total.Value)
			require.Empty(t, result.SuggestedExpenseTypeID)
			require.Empty(t, result.CategoryReason)
		})
	}
}

func TestGeminiReceivesCategoriesAndConstrainsSuggestion(t *testing.T) {
	categories := []ExpenseCategory{{ID: groceryID, Name: "Groceries", Category: "Food"}}
	for _, returnedID := range []string{groceryID, "unknown", ""} {
		t.Run(returnedID, func(t *testing.T) {
			before := string(GeminiSchema)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var request map[string]any
				require.NoError(t, json.Unmarshal(body, &request))
				parts := request["contents"].([]any)[0].(map[string]any)["parts"].([]any)
				require.Contains(t, parts[0].(map[string]any)["text"], "Groceries")
				require.Contains(t, parts[0].(map[string]any)["text"], "Fully covered by CT Money")
				require.Len(t, parts, 2)
				schema := request["generationConfig"].(map[string]any)["responseJsonSchema"].(map[string]any)
				properties := schema["properties"].(map[string]any)
				require.Contains(t, properties, "loyaltyRedemption")
				require.Contains(t, properties, "redemptionNoteConflict")
				require.Len(t, properties["items"], 2)
				require.Contains(t, properties["items"], "items")
				require.Equal(t, []any{"", groceryID}, schema["properties"].(map[string]any)["suggestedExpenseTypeId"].(map[string]any)["enum"])
				output := strings.TrimSuffix(sampleExtraction, "}") + `,"suggestedExpenseTypeId":"` + returnedID + `","categoryReason":"The purchased item is milk."}`
				text, _ := json.Marshal(output)
				_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":` + string(text) + `}]},"finishReason":"STOP"}]}`))
			}))
			defer server.Close()
			provider, err := NewGeminiProvider("test-secret", DefaultGeminiModel)
			require.NoError(t, err)
			provider.endpoint = server.URL
			result, err := provider.AnalyzeExpense(t.Context(), []byte("image"), categories, "Fully covered by CT Money")
			require.NoError(t, err)
			if returnedID == groceryID {
				require.Equal(t, groceryID, result.SuggestedExpenseTypeID)
			} else {
				require.Empty(t, result.SuggestedExpenseTypeID)
				require.Empty(t, result.CategoryReason)
			}
			require.Equal(t, before, string(GeminiSchema), "request must not mutate shared schema")
		})
	}
}

func TestDraftPassesCatalogAndValidatesProviderSuggestion(t *testing.T) {
	for _, id := range []string{groceryID, "unknown"} {
		now, account, request, token, secret := draftFixture(t, "image/jpeg")
		provider := &providerStub{result: Result{SuggestedExpenseTypeID: id, CategoryReason: "Milk is a grocery purchase.", Total: Field{Value: "5.39"}}}
		handler := NewDraftHandler(RuntimeConfig{FrontendOrigin: "https://app.example.com", CapabilitySecret: secret}, &replayStub{}, provider, nil)
		handler.now = func() time.Time { return now.Add(time.Second) }
		response, err := handler.Handle(t.Context(), categoryDraftRequest(token, account, request, "image/jpeg", encodeTestJPEG(t, 8, 8), `[{"id":"`+groceryID+`","name":"Groceries","category":"Food"}]`))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, groceryID, provider.categories[0].ID)
		var payload DraftResponse
		require.NoError(t, json.Unmarshal([]byte(response.Body), &payload))
		require.True(t, VerifyScanProof(secret, account, request, payload.Scan.Note, payload.Scan.Proof))
		require.False(t, VerifyScanProof(secret, groceryID, request, payload.Scan.Note, payload.Scan.Proof))
		require.False(t, VerifyScanProof(secret, account, groceryID, payload.Scan.Note, payload.Scan.Proof))
		require.False(t, VerifyScanProof(secret, account, request, "changed", payload.Scan.Proof))
		if id == groceryID {
			require.Equal(t, id, payload.Draft.SuggestedExpenseTypeID)
		} else {
			require.Empty(t, payload.Draft.SuggestedExpenseTypeID)
		}
	}
}

func TestDraftRejectsInvalidMultipartBeforeChargingScan(t *testing.T) {
	now, account, requestID, token, secret := draftFixture(t, "image/jpeg")
	image := encodeTestJPEG(t, 8, 8)
	for name, request := range map[string]struct {
		catalog string
		raw     bool
	}{
		"legacy raw upload": {raw: true},
		"invalid catalog":   {catalog: `[{"id":"bad","name":"Groceries","category":"Food"}]`},
		"oversize catalog":  {catalog: strings.Repeat(" ", MaxCategoryContextBytes+1)},
	} {
		t.Run(name, func(t *testing.T) {
			provider := &providerStub{}
			replay := &replayStub{}
			h := NewDraftHandler(RuntimeConfig{FrontendOrigin: "https://app.example.com", CapabilitySecret: secret}, replay, provider, nil)
			h.now = func() time.Time { return now.Add(time.Second) }
			upload := categoryDraftRequest(token, account, requestID, "image/jpeg", image, request.catalog)
			if request.raw {
				upload = stubRequest(token, account, requestID, "image/jpeg", image)
			}
			response, err := h.Handle(t.Context(), upload)
			require.NoError(t, err)
			require.GreaterOrEqual(t, response.StatusCode, 400)
			require.Zero(t, provider.calls)
			require.Empty(t, replay.claimed)
		})
	}
}
