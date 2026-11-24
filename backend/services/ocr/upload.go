package ocr

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"unicode/utf8"
	"uuid"

	"github.com/aws/aws-lambda-go/events"
)

const (
	MaxCategoryContextBytes = 32 * 1024
	MaxScanNoteBytes        = 1024
	MaxExpenseCategories    = 200
	MaxCategoryLabelBytes   = 256
	// The maximum decoded envelope is under 3.6 MiB; base64 plus event
	// headers still fits within Lambda's 6 MiB synchronous invocation limit.
	MaxUploadBytes = MaxDocumentBytes + MaxCategoryContextBytes + 16*1024
)

var errScanNoteTooLong = errors.New("scan note exceeds the limit")
var errInvalidUpload = errors.New("invalid receipt upload")

// ExpenseCategory is advisory classification context, never authorization.
type ExpenseCategory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type receiptUpload struct {
	Document   []byte
	ImageType  string
	Categories []ExpenseCategory
	Note       string
}

func parseReceiptUpload(request events.APIGatewayV2HTTPRequest, boundary string) (receiptUpload, error) {
	body, err := requestBody(request, MaxUploadBytes)
	if err != nil {
		return receiptUpload{}, err
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var document []byte
	var imageType string
	var categories []ExpenseCategory
	var note string
	seen := map[string]bool{}
	for {
		part, err := reader.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return receiptUpload{}, errInvalidUpload
		}
		name := part.FormName()
		if seen[name] || (name != "image" && name != "categories" && name != "note") || part.Header.Get("Content-Transfer-Encoding") != "" {
			return receiptUpload{}, errInvalidUpload
		}
		seen[name] = true
		limit := int64(MaxCategoryContextBytes)
		if name == "image" {
			limit = MaxDocumentBytes
		} else if name == "note" {
			// Multipart text fields can encode each newline as two bytes.
			limit = 2 * MaxScanNoteBytes
		}
		data, err := io.ReadAll(io.LimitReader(part, limit+1))
		if err != nil {
			return receiptUpload{}, errInvalidUpload
		}
		if int64(len(data)) > limit {
			if name == "note" {
				return receiptUpload{}, errScanNoteTooLong
			}
			return receiptUpload{}, errDocumentTooLarge
		}
		if name == "image" {
			imageType, _, err = mime.ParseMediaType(part.Header.Get("Content-Type"))
			if err != nil || !SupportedContentType(imageType) || len(data) == 0 {
				return receiptUpload{}, errInvalidUpload
			}
			document = data
		} else if name == "note" {
			if part.FileName() != "" || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
				return receiptUpload{}, errInvalidUpload
			}
			note = strings.TrimSpace(strings.ReplaceAll(string(data), "\r\n", "\n"))
			if len(note) > MaxScanNoteBytes {
				return receiptUpload{}, errScanNoteTooLong
			}
		} else {
			if err := json.Unmarshal(data, &categories, json.RejectUnknownMembers(true)); err != nil {
				return receiptUpload{}, errInvalidUpload
			}
			if !validExpenseCategories(categories) {
				return receiptUpload{}, errInvalidUpload
			}
		}
	}
	if !seen["image"] || !seen["categories"] {
		return receiptUpload{}, errInvalidUpload
	}
	return receiptUpload{Document: document, ImageType: imageType, Categories: categories, Note: note}, nil
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
