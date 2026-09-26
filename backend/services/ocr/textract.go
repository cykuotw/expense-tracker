package ocr

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/textract/types"
	"github.com/aws/smithy-go"
)

var (
	ErrProviderRequestFailed  = errors.New("expense analysis provider request failed")
	ErrProviderRejected       = errors.New("expense analysis provider rejected the document")
	ErrProviderThrottled      = errors.New("expense analysis provider throttled the request")
	ErrUnexpectedExpenseCount = errors.New("provider returned an unexpected number of expense documents")
)

type textractAPI interface {
	AnalyzeExpense(context.Context, *textract.AnalyzeExpenseInput, ...func(*textract.Options)) (*textract.AnalyzeExpenseOutput, error)
}

// TextractProvider adapts AnalyzeExpense to the provider-neutral OCR contract.
type TextractProvider struct {
	client textractAPI
}

func NewTextractProvider(client textractAPI) *TextractProvider {
	return &TextractProvider{client: client}
}

func (provider *TextractProvider) AnalyzeExpense(ctx context.Context, document []byte) (Result, error) {
	response, err := provider.client.AnalyzeExpense(ctx, &textract.AnalyzeExpenseInput{
		Document: &types.Document{Bytes: document},
	})
	if err != nil {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.ErrorCode() {
			case "BadDocumentException", "DocumentTooLargeException", "UnsupportedDocumentException", "InvalidParameterException":
				return Result{}, ErrProviderRejected
			case "ProvisionedThroughputExceededException", "ThrottlingException":
				return Result{}, ErrProviderThrottled
			}
		}
		return Result{}, ErrProviderRequestFailed
	}
	if response == nil || len(response.ExpenseDocuments) != 1 {
		return Result{}, ErrUnexpectedExpenseCount
	}
	return normalizeExpenseDocument(response.ExpenseDocuments[0]), nil
}

func normalizeExpenseDocument(document types.ExpenseDocument) Result {
	var result Result
	for _, field := range document.SummaryFields {
		typeName := normalizedType(field)
		value := normalizedField(field)
		switch typeName {
		case "VENDOR_NAME":
			setHigherConfidence(&result.Merchant, value)
		case "INVOICE_RECEIPT_DATE":
			setHigherConfidence(&result.Date, value)
		case "SUBTOTAL":
			setHigherConfidence(&result.Subtotal, value)
		case "TAX":
			setHigherConfidence(&result.Tax, value)
		case "TIP":
			setHigherConfidence(&result.Tip, value)
		case "TOTAL":
			setHigherConfidence(&result.Total, value)
		}
		if field.Currency != nil && field.Currency.Code != nil {
			setHigherConfidence(&result.Currency, Field{
				Value:      strings.TrimSpace(aws.ToString(field.Currency.Code)),
				Confidence: aws.ToFloat32(field.Currency.Confidence),
			})
		}
	}

	for _, group := range document.LineItemGroups {
		for _, line := range group.LineItems {
			item := Item{}
			for _, field := range line.LineItemExpenseFields {
				value := normalizedField(field)
				switch normalizedType(field) {
				case "ITEM":
					setHigherConfidence(&item.Description, value)
				case "QUANTITY":
					setHigherConfidence(&item.Quantity, value)
				case "UNIT_PRICE":
					setHigherConfidence(&item.UnitPrice, value)
				case "PRICE":
					setHigherConfidence(&item.LineTotal, value)
				case "EXPENSE_ROW":
					setHigherConfidence(&item.RawRow, value)
				}
			}
			if item.Description.Value == "" {
				item.Description = item.RawRow
			}
			result.Items = append(result.Items, item)
		}
	}
	return result
}

func normalizedType(field types.ExpenseField) string {
	if field.Type == nil || field.Type.Text == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(aws.ToString(field.Type.Text)))
}

func normalizedField(field types.ExpenseField) Field {
	if field.ValueDetection == nil {
		return Field{}
	}
	return Field{
		Value:      strings.TrimSpace(aws.ToString(field.ValueDetection.Text)),
		Confidence: aws.ToFloat32(field.ValueDetection.Confidence),
	}
}

func setHigherConfidence(destination *Field, candidate Field) {
	if candidate.Value != "" && (destination.Value == "" || candidate.Confidence > destination.Confidence) {
		*destination = candidate
	}
}
