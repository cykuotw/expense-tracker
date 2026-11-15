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

func categoryDraftRequest(token, accountID, requestID, imageType string, image []byte, catalog string) events.APIGatewayV2HTTPRequest {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreatePart(textproto.MIMEHeader{"Content-Disposition": {`form-data; name="image"; filename="receipt"`}, "Content-Type": {imageType}})
	_, _ = part.Write(image)
	_ = w.WriteField("categories", catalog)
	_ = w.Close()
	return stubRequest(token, accountID, requestID, w.FormDataContentType(), body.Bytes())
}

func TestReceiptUploadBoundsAndParts(t *testing.T) {
	valid := categoryDraftRequest("", "", "", "image/jpeg", []byte("image"), `[{"id":"00000000-0000-4000-8000-000000000001","name":"Groceries","category":"Food"}]`)
	boundary := strings.TrimPrefix(valid.Headers["Content-Type"], "multipart/form-data; boundary=")
	image, mime, categories, err := parseReceiptUpload(valid, boundary)
	require.NoError(t, err)
	require.Equal(t, []byte("image"), image)
	require.Equal(t, "image/jpeg", mime)
	require.Len(t, categories, 1)
	for name, request := range map[string]events.APIGatewayV2HTTPRequest{
		"bad base64":      {Body: "!", IsBase64Encoded: true},
		"truncated":       {Body: "--" + boundary + "\r\n"},
		"missing parts":   {Body: "--" + boundary + "--\r\n"},
		"duplicate image": {Body: strings.Repeat("--"+boundary+"\r\nContent-Disposition: form-data; name=\"image\"\r\nContent-Type: image/jpeg\r\n\r\nx\r\n", 2) + "--" + boundary + "--\r\n"},
		"extra part":      {Body: "--" + boundary + "\r\nContent-Disposition: form-data; name=\"extra\"\r\n\r\nx\r\n--" + boundary + "--\r\n"},
	} {
		t.Run(name, func(t *testing.T) { _, _, _, err := parseReceiptUpload(request, boundary); require.Error(t, err) })
	}
	for _, catalog := range []string{`{}`, `[{"id":"unknown","name":"X","category":"Y"}]`, `[{"id":"00000000-0000-4000-8000-000000000001","name":"X","category":"Y","extra":true}]`, strings.Repeat(" ", MaxCategoryContextBytes+1)} {
		request := categoryDraftRequest("", "", "", "image/jpeg", []byte("image"), catalog)
		boundary := strings.TrimPrefix(request.Headers["Content-Type"], "multipart/form-data; boundary=")
		_, _, _, err := parseReceiptUpload(request, boundary)
		require.Error(t, err)
	}
	oversize := valid
	oversize.Body = base64.StdEncoding.EncodeToString(make([]byte, MaxUploadBytes+1))
	_, _, _, err = parseReceiptUpload(oversize, boundary)
	require.ErrorIs(t, err, errDocumentTooLarge)
}

func TestMaximumMultipartEventFitsSynchronousLambdaPayload(t *testing.T) {
	request := stubRequest("capability", "account", "request", "multipart/form-data; boundary=test", make([]byte, MaxUploadBytes))
	request.Headers["large-header-budget"] = strings.Repeat("x", 16*1024)
	encoded, err := json.Marshal(request)
	require.NoError(t, err)
	require.Less(t, len(encoded), 6_000_000)
}
