package ocr

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
)

const replayRetentionAfterExpiry = 5 * time.Minute

var ErrCapabilityReplay = errors.New("OCR capability has already been used")

type ReplayStore interface {
	Claim(ctx context.Context, requestID string, expiresAt time.Time) error
}

type dynamoPutItemClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type DynamoReplayStore struct {
	client    dynamoPutItemClient
	tableName string
}

func NewDynamoReplayStore(client dynamoPutItemClient, tableName string) *DynamoReplayStore {
	return &DynamoReplayStore{client: client, tableName: tableName}
}

func (s *DynamoReplayStore) Claim(ctx context.Context, requestID string, expiresAt time.Time) error {
	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		ConditionExpression: aws.String("attribute_not_exists(request_id)"),
		Item: map[string]dynamodbtypes.AttributeValue{
			"request_id": &dynamodbtypes.AttributeValueMemberS{Value: requestID},
			"expires_at": &dynamodbtypes.AttributeValueMemberN{
				Value: strconv.FormatInt(expiresAt.Add(replayRetentionAfterExpiry).Unix(), 10),
			},
		},
	})
	if err == nil {
		return nil
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) && apiError.ErrorCode() == "ConditionalCheckFailedException" {
		return ErrCapabilityReplay
	}
	return fmt.Errorf("claim OCR capability: %w", err)
}
