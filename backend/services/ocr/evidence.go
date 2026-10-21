package ocr

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const ImprovementPolicyVersion = "ocr-improvement-v1"
const EvidenceTTL = 24 * time.Hour

type EvidenceClaims struct {
	Revision int64  `json:"revision"`
	Checksum string `json:"checksum"`
	jwt.RegisteredClaims
}

func EvidenceTemporaryKey(account, run string) string {
	return "temporary/" + account + "/" + run + ".json.gz"
}
func EvidencePrefix(account, run string) string { return "confirmed/" + account + "/" + run + "/" }

func VerifyEvidenceToken(secret []byte, token, account string, now time.Time) (*EvidenceClaims, error) {
	claims := new(EvidenceClaims)
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) { return secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithAudience("ocr-evidence"), jwt.WithIssuer(CapabilityIssuer), jwt.WithExpirationRequired(), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil || !parsed.Valid || len(secret) < 32 || claims.Subject != account || claims.Revision <= 0 || len(claims.Checksum) != 64 || claims.ID == "" {
		return nil, errors.New("invalid evidence token")
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return nil, errors.New("invalid evidence token")
	}
	if _, err := uuid.Parse(claims.ID); err != nil {
		return nil, errors.New("invalid evidence token")
	}
	return claims, nil
}

type EvidenceWriter struct {
	Client ReceiptPutObjectAPI
	Bucket string
	Secret []byte
}

func (w *EvidenceWriter) Store(ctx context.Context, claims *CapabilityClaims, image []byte, draft Result, now time.Time) (string, error) {
	if claims.ConsentRevision <= 0 || draft.Evidence == nil || w.Bucket == "" {
		return "", nil
	}
	bundle := struct {
		SchemaVersion        int                 `json:"schemaVersion"`
		Image                []byte              `json:"image"`
		Extraction           *ExtractionEvidence `json:"extraction"`
		Draft                Result              `json:"draft"`
		PreprocessingVersion string              `json:"preprocessingVersion"`
	}{1, image, draft.Evidence, draft, "raster-v1"}
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if err := json.NewEncoder(writer).Encode(bundle); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	digest := contentDigest(buffer.Bytes())
	_, err := w.Client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(w.Bucket), Key: aws.String(EvidenceTemporaryKey(claims.Subject, claims.ID)), Body: bytes.NewReader(buffer.Bytes()), ContentType: aws.String("application/gzip"), ServerSideEncryption: s3types.ServerSideEncryptionAes256, CacheControl: aws.String("private, no-store"), Metadata: map[string]string{"sha256": digest, "consent-revision": strconv.FormatInt(claims.ConsentRevision, 10), "provider": "gemini", "model": draft.Evidence.Model, "prompt-digest": draft.Evidence.PromptDigest, "schema-digest": draft.Evidence.SchemaDigest, "adapter-version": draft.Evidence.AdapterVersion, "preprocessing-version": "raster-v1", "evidence-schema-version": "1"}})
	if err != nil {
		return "", err
	}
	tokenClaims := EvidenceClaims{Revision: claims.ConsentRevision, Checksum: digest, RegisteredClaims: jwt.RegisteredClaims{Issuer: CapabilityIssuer, Subject: claims.Subject, ID: claims.ID, Audience: jwt.ClaimStrings{"ocr-evidence"}, ExpiresAt: jwt.NewNumericDate(now.Add(EvidenceTTL))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims).SignedString(w.Secret)
}
