package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json/v2"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

func multipartDraftRequest(token, accountID, requestID, imageType string, image []byte) events.APIGatewayV2HTTPRequest {
	return categoryDraftRequest(token, accountID, requestID, imageType, image, `[]`)
}

func categoryDraftRequest(token, accountID, requestID, imageType string, image []byte, catalog string, notes ...string) events.APIGatewayV2HTTPRequest {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreatePart(textproto.MIMEHeader{"Content-Disposition": {`form-data; name="image"; filename="receipt"`}, "Content-Type": {imageType}})
	_, _ = part.Write(image)
	_ = w.WriteField("categories", catalog)
	for _, note := range notes {
		_ = w.WriteField("note", note)
	}
	_ = w.Close()
	return stubRequest(token, accountID, requestID, w.FormDataContentType(), body.Bytes())
}

func TestReceiptUploadBoundsAndParts(t *testing.T) {
	valid := categoryDraftRequest("", "", "", "image/jpeg", []byte("image"), `[{"id":"00000000-0000-4000-8000-000000000001","name":"Groceries","category":"Food"}]`)
	boundary := strings.TrimPrefix(valid.Headers["Content-Type"], "multipart/form-data; boundary=")
	upload, err := parseReceiptUpload(valid, boundary)
	require.NoError(t, err)
	require.Equal(t, []byte("image"), upload.Document)
	require.Equal(t, "image/jpeg", upload.ImageType)
	require.Len(t, upload.Categories, 1)
	for name, request := range map[string]events.APIGatewayV2HTTPRequest{
		"bad base64":      {Body: "!", IsBase64Encoded: true},
		"truncated":       {Body: "--" + boundary + "\r\n"},
		"missing parts":   {Body: "--" + boundary + "--\r\n"},
		"duplicate image": {Body: strings.Repeat("--"+boundary+"\r\nContent-Disposition: form-data; name=\"image\"\r\nContent-Type: image/jpeg\r\n\r\nx\r\n", 2) + "--" + boundary + "--\r\n"},
		"extra part":      {Body: "--" + boundary + "\r\nContent-Disposition: form-data; name=\"extra\"\r\n\r\nx\r\n--" + boundary + "--\r\n"},
	} {
		t.Run(name, func(t *testing.T) { _, err := parseReceiptUpload(request, boundary); require.Error(t, err) })
	}
	for _, catalog := range []string{`{}`, `[{"id":"unknown","name":"X","category":"Y"}]`, `[{"id":"00000000-0000-4000-8000-000000000001","name":"X","category":"Y","extra":true}]`, strings.Repeat(" ", MaxCategoryContextBytes+1)} {
		request := categoryDraftRequest("", "", "", "image/jpeg", []byte("image"), catalog)
		boundary := strings.TrimPrefix(request.Headers["Content-Type"], "multipart/form-data; boundary=")
		_, err := parseReceiptUpload(request, boundary)
		require.Error(t, err)
	}
	oversize := valid
	oversize.Body = base64.StdEncoding.EncodeToString(make([]byte, MaxUploadBytes+1))
	_, err = parseReceiptUpload(oversize, boundary)
	require.ErrorIs(t, err, errDocumentTooLarge)
}

func TestMaximumMultipartEventFitsSynchronousLambdaPayload(t *testing.T) {
	request := stubRequest("capability", "account", "request", "multipart/form-data; boundary=test", make([]byte, MaxUploadBytes))
	request.Headers["large-header-budget"] = strings.Repeat("x", 16*1024)
	encoded, err := json.Marshal(request)
	require.NoError(t, err)
	require.Less(t, len(encoded), 6_000_000)
}

func TestReceiptUploadScanNote(t *testing.T) {
	request := categoryDraftRequest("", "", "", "image/jpeg", []byte("image"), `[]`, "  Fully covered by CT Money  ")
	boundary := strings.TrimPrefix(request.Headers["Content-Type"], "multipart/form-data; boundary=")
	upload, err := parseReceiptUpload(request, boundary)
	require.NoError(t, err)
	require.Equal(t, "Fully covered by CT Money", upload.Note)

	for _, notes := range [][]string{{"one", "two"}, {strings.Repeat("買", MaxScanNoteBytes)}} {
		request := categoryDraftRequest("", "", "", "image/jpeg", []byte("image"), `[]`, notes...)
		boundary := strings.TrimPrefix(request.Headers["Content-Type"], "multipart/form-data; boundary=")
		_, err := parseReceiptUpload(request, boundary)
		require.Error(t, err)
	}
}

func TestReceiptUploadMultilineNoteLimit(t *testing.T) {
	for _, value := range []string{
		strings.Repeat("a", MaxScanNoteBytes-2) + "\r\nb",
		strings.Repeat("a", MaxScanNoteBytes-1) + "\r\nb",
	} {
		request := categoryDraftRequest("", "", "", "image/jpeg", []byte("image"), "[]", value)
		boundary := strings.TrimPrefix(request.Headers["Content-Type"], "multipart/form-data; boundary=")
		upload, err := parseReceiptUpload(request, boundary)
		normalized := strings.ReplaceAll(value, "\r\n", "\n")
		if len(normalized) == MaxScanNoteBytes {
			require.NoError(t, err)
			require.Equal(t, normalized, upload.Note)
		} else {
			require.ErrorIs(t, err, errScanNoteTooLong)
		}
	}
}
