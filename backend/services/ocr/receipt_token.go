package ocr

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	ReceiptTokenIssuer   = "expense-tracker-ocr"
	ReceiptTokenAudience = "expense-tracker-receipt-attach"
	ReceiptTokenGrant    = "keep_receipt"
	ReceiptTokenTTL      = time.Hour
	TemporaryReceiptTTL  = 24 * time.Hour
)

var (
	ErrInvalidReceiptToken = errors.New("invalid receipt token")
	receiptChecksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// TemporaryReceipt describes a server-created object. Key never comes from a client.
type TemporaryReceipt struct {
	AccountID      string
	ObjectID       string
	Key            string
	ChecksumSHA256 string
	ContentType    string
	ByteSize       int64
	Width          int
	Height         int
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

type ReceiptTokenClaims struct {
	Grant           string `json:"grant"`
	ChecksumSHA256  string `json:"checksum_sha256"`
	ContentType     string `json:"content_type"`
	ByteSize        int64  `json:"byte_size"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	ObjectExpiresAt int64  `json:"object_expires_at"`
	jwt.RegisteredClaims
}

func TemporaryReceiptKey(objectID string) (string, error) {
	parsed, err := uuid.Parse(objectID)
	if err != nil || parsed.Version() != 4 || parsed.Variant() != uuid.RFC4122 || parsed.String() != objectID {
		return "", ErrInvalidReceiptToken
	}
	return "temporary/" + objectID + ".jpg", nil
}

func MintReceiptToken(secret []byte, receipt TemporaryReceipt) (string, error) {
	if len(secret) < 32 || !validTemporaryReceipt(receipt) {
		return "", ErrInvalidReceiptToken
	}
	issuedAt := receipt.CreatedAt.UTC().Truncate(time.Second)
	claims := ReceiptTokenClaims{
		Grant: ReceiptTokenGrant, ChecksumSHA256: receipt.ChecksumSHA256,
		ContentType: receipt.ContentType, ByteSize: receipt.ByteSize,
		Width: receipt.Width, Height: receipt.Height,
		ObjectExpiresAt: receipt.ExpiresAt.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: ReceiptTokenIssuer, Subject: receipt.AccountID,
			Audience: jwt.ClaimStrings{ReceiptTokenAudience},
			IssuedAt: jwt.NewNumericDate(issuedAt), NotBefore: jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(ReceiptTokenTTL)), ID: receipt.ObjectID,
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("sign receipt token: %w", err)
	}
	return token, nil
}

func VerifyReceiptToken(secret []byte, tokenString string, now time.Time, accountID string) (TemporaryReceipt, error) {
	if len(secret) < 32 || tokenString == "" || accountID == "" {
		return TemporaryReceipt{}, ErrInvalidReceiptToken
	}
	claims := &ReceiptTokenClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(ReceiptTokenIssuer), jwt.WithAudience(ReceiptTokenAudience),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(),
		jwt.WithTimeFunc(func() time.Time { return now.UTC() }),
	)
	parsed, err := parser.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (any, error) { return secret, nil })
	if err != nil || !parsed.Valid || claims.Subject != accountID ||
		claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return TemporaryReceipt{}, ErrInvalidReceiptToken
	}
	key, err := TemporaryReceiptKey(claims.ID)
	if err != nil || claims.Grant != ReceiptTokenGrant ||
		claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) != ReceiptTokenTTL {
		return TemporaryReceipt{}, ErrInvalidReceiptToken
	}
	receipt := TemporaryReceipt{
		AccountID: claims.Subject, ObjectID: claims.ID, Key: key,
		ChecksumSHA256: claims.ChecksumSHA256, ContentType: claims.ContentType,
		ByteSize: claims.ByteSize, Width: claims.Width, Height: claims.Height,
		CreatedAt: claims.IssuedAt.Time,
		ExpiresAt: time.Unix(claims.ObjectExpiresAt, 0).UTC(),
	}
	if !validTemporaryReceipt(receipt) || !now.Before(receipt.ExpiresAt) {
		return TemporaryReceipt{}, ErrInvalidReceiptToken
	}
	return receipt, nil
}

func validTemporaryReceipt(receipt TemporaryReceipt) bool {
	if _, err := uuid.Parse(receipt.AccountID); err != nil {
		return false
	}
	key, err := TemporaryReceiptKey(receipt.ObjectID)
	return err == nil && receipt.Key == key &&
		receipt.ContentType == "image/jpeg" &&
		receipt.ByteSize > 0 && receipt.ByteSize <= DefaultMaxOutputBytes &&
		receipt.Width > 0 && receipt.Height > 0 &&
		receipt.Width <= DefaultMaxDimension && receipt.Height <= DefaultMaxDimension &&
		int64(receipt.Width)*int64(receipt.Height) <= DefaultMaxPixels &&
		receiptChecksumPattern.MatchString(receipt.ChecksumSHA256) &&
		receipt.CreatedAt.Equal(receipt.CreatedAt.UTC().Truncate(time.Second)) &&
		receipt.ExpiresAt.Sub(receipt.CreatedAt) == TemporaryReceiptTTL
}
