package ocr

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

const (
	AccountIDHeader  = "X-OCR-Account-ID"
	OCRRequestHeader = "X-OCR-Request-ID"
)

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

var errDocumentTooLarge = errors.New("document too large")

func requestBody(request events.APIGatewayV2HTTPRequest, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, errDocumentTooLarge
	}
	if !request.IsBase64Encoded {
		if int64(len(request.Body)) > maxBytes {
			return nil, errDocumentTooLarge
		}
		return []byte(request.Body), nil
	}
	maxEncoded := base64.StdEncoding.EncodedLen(int(maxBytes))
	if len(request.Body) > maxEncoded {
		return nil, errDocumentTooLarge
	}
	decoded, err := base64.StdEncoding.DecodeString(request.Body)
	if err != nil {
		return nil, err
	}
	if int64(len(decoded)) > maxBytes {
		return nil, errDocumentTooLarge
	}
	return decoded, nil
}

func bearerToken(value string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(value), " ")
	return token, found && strings.EqualFold(scheme, "Bearer") && token != "" && !strings.Contains(token, " ")
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
