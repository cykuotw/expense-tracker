package ocr

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/textract/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTextractClient struct {
	response *textract.AnalyzeExpenseOutput
	err      error
	input    *textract.AnalyzeExpenseInput
}

func (client *fakeTextractClient) AnalyzeExpense(
	_ context.Context,
	input *textract.AnalyzeExpenseInput,
	_ ...func(*textract.Options),
) (*textract.AnalyzeExpenseOutput, error) {
	client.input = input
	return client.response, client.err
}

func TestTextractProviderNormalizesSummaryAndItems(t *testing.T) {
	client := &fakeTextractClient{response: &textract.AnalyzeExpenseOutput{
		ExpenseDocuments: []types.ExpenseDocument{{
			SummaryFields: []types.ExpenseField{
				expenseField("VENDOR_NAME", "Example Market", 91),
				expenseFieldWithCurrency("TOTAL", "$12.34", "CAD", 99),
			},
			LineItemGroups: []types.LineItemGroup{{
				LineItems: []types.LineItemFields{{LineItemExpenseFields: []types.ExpenseField{
					expenseField("ITEM", "Milk", 98),
					expenseField("QUANTITY", "2", 95),
					expenseField("PRICE", "$8.00", 97),
				}}},
			}},
		}},
	}}
	provider := NewTextractProvider(client)

	result, err := provider.AnalyzeExpense(t.Context(), []byte("normalized image"))
	require.NoError(t, err)
	require.NotNil(t, client.input)
	assert.Equal(t, []byte("normalized image"), client.input.Document.Bytes)
	assert.Equal(t, "Example Market", result.Merchant.Value)
	assert.Equal(t, "$12.34", result.Total.Value)
	assert.Equal(t, "CAD", result.Currency.Value)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Milk", result.Items[0].Description.Value)
	assert.Equal(t, "2", result.Items[0].Quantity.Value)
	assert.Equal(t, "$8.00", result.Items[0].LineTotal.Value)
}

func TestTextractProviderDoesNotExposeProviderError(t *testing.T) {
	client := &fakeTextractClient{err: errors.New("sensitive provider payload")}
	provider := NewTextractProvider(client)

	_, err := provider.AnalyzeExpense(t.Context(), []byte("image"))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "sensitive provider payload")
}

func TestTextractProviderRejectsMultipleDocuments(t *testing.T) {
	client := &fakeTextractClient{response: &textract.AnalyzeExpenseOutput{
		ExpenseDocuments: []types.ExpenseDocument{{}, {}},
	}}
	provider := NewTextractProvider(client)

	_, err := provider.AnalyzeExpense(t.Context(), []byte("image"))
	assert.ErrorIs(t, err, ErrUnexpectedExpenseCount)
}

func expenseField(fieldType, value string, confidence float32) types.ExpenseField {
	return types.ExpenseField{
		Type:           &types.ExpenseType{Text: aws.String(fieldType)},
		ValueDetection: &types.ExpenseDetection{Text: aws.String(value), Confidence: aws.Float32(confidence)},
	}
}

func expenseFieldWithCurrency(fieldType, value, currency string, confidence float32) types.ExpenseField {
	field := expenseField(fieldType, value, confidence)
	field.Currency = &types.ExpenseCurrency{Code: aws.String(currency), Confidence: aws.Float32(confidence)}
	return field
}
