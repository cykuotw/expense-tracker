package ocr

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

type replayStub struct {
	claimed string
	err     error
}

func (s *replayStub) Claim(_ context.Context, requestID string, _ time.Time) error {
	s.claimed = requestID
	return s.err
}

func stubRequest(token, accountID, requestID, contentType string, body []byte) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		Headers: map[string]string{
			"Origin":         "https://app.example.com",
			"Authorization":  "Bearer " + token,
			"Content-Type":   contentType,
			AccountIDHeader:  accountID,
			OCRRequestHeader: requestID,
		},
		Body:            base64.StdEncoding.EncodeToString(body),
		IsBase64Encoded: true,
	}
}
