package ocr

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"uuid"

	"github.com/aws/aws-lambda-go/events"
)

const (
	MaxCategoryContextBytes = 32 * 1024
	MaxExpenseCategories    = 200
	MaxCategoryLabelBytes   = 256
	// The maximum decoded envelope is under 3.6 MiB; base64 plus event
	// headers still fits within Lambda's 6 MiB synchronous invocation limit.
	MaxUploadBytes = MaxDocumentBytes + MaxCategoryContextBytes + 16*1024
)

var errInvalidUpload = errors.New("invalid receipt upload")

// ExpenseCategory is advisory classification context, never authorization.
type ExpenseCategory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

func parseReceiptUpload(request events.APIGatewayV2HTTPRequest, boundary string) ([]byte, string, []ExpenseCategory, error) {
	body, err := requestBody(request, MaxUploadBytes)
	if err != nil {
		return nil, "", nil, err
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var document []byte
	var imageType string
	var categories []ExpenseCategory
	seen := map[string]bool{}
	for {
		part, err := reader.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", nil, errInvalidUpload
		}
		name := part.FormName()
		if seen[name] || len(seen) >= 2 || (name != "image" && name != "categories") || part.Header.Get("Content-Transfer-Encoding") != "" {
			return nil, "", nil, errInvalidUpload
		}
		seen[name] = true
		limit := int64(MaxCategoryContextBytes)
		if name == "image" {
			limit = MaxDocumentBytes
		}
		data, err := io.ReadAll(io.LimitReader(part, limit+1))
		if err != nil {
			return nil, "", nil, errInvalidUpload
		}
		if int64(len(data)) > limit {
			return nil, "", nil, errDocumentTooLarge
		}
		if name == "image" {
			imageType, _, err = mime.ParseMediaType(part.Header.Get("Content-Type"))
			if err != nil || !SupportedContentType(imageType) || len(data) == 0 {
				return nil, "", nil, errInvalidUpload
			}
			document = data
		} else {
			if err := json.Unmarshal(data, &categories, json.RejectUnknownMembers(true)); err != nil {
				return nil, "", nil, errInvalidUpload
			}
			if !validExpenseCategories(categories) {
				return nil, "", nil, errInvalidUpload
			}
		}
	}
	if !seen["image"] || !seen["categories"] {
		return nil, "", nil, errInvalidUpload
	}
	return document, imageType, categories, nil
}

func validExpenseCategories(categories []ExpenseCategory) bool {
	if len(categories) > MaxExpenseCategories {
		return false
	}
	seen := map[string]bool{}
	for _, category := range categories {
		id, err := uuid.Parse(category.ID)
		if err != nil || id.String() != category.ID || seen[category.ID] ||
			strings.TrimSpace(category.Name) == "" || len(category.Name) > MaxCategoryLabelBytes ||
			strings.TrimSpace(category.Category) == "" || len(category.Category) > MaxCategoryLabelBytes {
			return false
		}
		seen[category.ID] = true
	}
	return true
}

func validateCategorySuggestion(result *Result, categories []ExpenseCategory) {
	if result.CategoryReason != "" && len(result.CategoryReason) <= 500 {
		for _, category := range categories {
			if category.ID == result.SuggestedExpenseTypeID {
				return
			}
		}
	}
	result.SuggestedExpenseTypeID = ""
	result.CategoryReason = ""
}
