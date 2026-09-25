package ocr

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	CapabilityIssuer   = "expense-tracker-worker"
	CapabilityAudience = "expense-tracker-ocr"
	CapabilityGrant    = "receipt_ocr"
	MaxDocumentBytes   = 3_670_016
	CapabilityTTL      = 60 * time.Second
)

var ErrInvalidCapability = errors.New("invalid OCR capability")

type CapabilityClaims struct {
	Grant       string `json:"grant"`
	ContentType string `json:"content_type"`
	MaxBytes    int64  `json:"max_bytes"`
	jwt.RegisteredClaims
}

func SupportedContentType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image/jpeg", "image/png":
		return true
	default:
		return false
	}
}

func MintCapability(secret []byte, now time.Time, accountID, requestID, contentType string) (string, time.Time, error) {
	if len(secret) < 32 {
		return "", time.Time{}, fmt.Errorf("OCR capability secret is too short")
	}
	if _, err := uuid.Parse(accountID); err != nil {
		return "", time.Time{}, ErrInvalidCapability
	}
	parsedRequestID, err := uuid.Parse(requestID)
	if err != nil || parsedRequestID.Version() != 4 || parsedRequestID.Variant() != uuid.RFC4122 {
		return "", time.Time{}, ErrInvalidCapability
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if !SupportedContentType(contentType) {
		return "", time.Time{}, ErrInvalidCapability
	}

	issuedAt := now.UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(CapabilityTTL)
	claims := CapabilityClaims{
		Grant:       CapabilityGrant,
		ContentType: contentType,
		MaxBytes:    MaxDocumentBytes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    CapabilityIssuer,
			Subject:   accountID,
			Audience:  jwt.ClaimStrings{CapabilityAudience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ID:        parsedRequestID.String(),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign OCR capability: %w", err)
	}
	return token, expiresAt, nil
}

func VerifyCapability(secret []byte, tokenString string, now time.Time) (*CapabilityClaims, error) {
	if len(secret) < 32 || strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidCapability
	}
	claims := &CapabilityClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(CapabilityIssuer),
		jwt.WithAudience(CapabilityAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(func() time.Time { return now.UTC() }),
	)
	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidCapability
		}
		return secret, nil
	})
	if err != nil || !token.Valid || !validCapabilityClaims(claims) {
		return nil, ErrInvalidCapability
	}
	return claims, nil
}

func validCapabilityClaims(claims *CapabilityClaims) bool {
	if claims == nil || claims.Grant != CapabilityGrant || claims.MaxBytes != MaxDocumentBytes ||
		!SupportedContentType(claims.ContentType) || claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return false
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return false
	}
	requestID, err := uuid.Parse(claims.ID)
	if err != nil || requestID.Version() != 4 || requestID.Variant() != uuid.RFC4122 {
		return false
	}
	return claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) == CapabilityTTL
}
