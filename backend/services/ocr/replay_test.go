package ocr

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
)

type dynamoStub struct {
	input *dynamodb.PutItemInput
	err   error
}

func (s *dynamoStub) PutItem(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	s.input = input
	return &dynamodb.PutItemOutput{}, s.err
}

func TestDynamoReplayStoreUsesConditionalTTLRecord(t *testing.T) {
	client := &dynamoStub{}
	expiresAt := time.Unix(1_800_000_000, 0)
	if err := NewDynamoReplayStore(client, "replay-table").Claim(t.Context(), "request-id", expiresAt); err != nil {
		t.Fatal(err)
	}
	if client.input == nil || *client.input.TableName != "replay-table" ||
		*client.input.ConditionExpression != "attribute_not_exists(request_id)" {
		t.Fatalf("unexpected input: %#v", client.input)
	}
	if _, ok := client.input.Item["request_id"].(*dynamodbtypes.AttributeValueMemberS); !ok {
		t.Fatal("request_id was not stored as a string")
	}
	if _, ok := client.input.Item["expires_at"].(*dynamodbtypes.AttributeValueMemberN); !ok {
		t.Fatal("expires_at was not stored as a number")
	}
}

func TestDynamoReplayStoreWrapsInfrastructureFailure(t *testing.T) {
	backendError := errors.New("backend unavailable")
	err := NewDynamoReplayStore(&dynamoStub{err: backendError}, "replay-table").Claim(
		t.Context(), "request-id", time.Now(),
	)
	if !errors.Is(err, backendError) {
		t.Fatalf("expected wrapped backend error, got %v", err)
	}
}

func TestDynamoReplayStoreMapsConditionalFailureToReplay(t *testing.T) {
	err := NewDynamoReplayStore(&dynamoStub{err: &smithy.GenericAPIError{
		Code: "ConditionalCheckFailedException", Message: "already claimed",
	}}, "replay-table").Claim(t.Context(), "request-id", time.Now())
	if !errors.Is(err, ErrCapabilityReplay) {
		t.Fatalf("expected replay error, got %v", err)
	}
}
