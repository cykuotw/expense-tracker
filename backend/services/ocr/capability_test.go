package ocr

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestCapabilityRoundTripAndContract(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	secret := []byte(strings.Repeat("s", 32))
	accountID := uuid.NewString()
	requestID := uuid.NewString()

	token, expiresAt, err := MintCapability(secret, now, accountID, requestID, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if !expiresAt.Equal(now.Add(CapabilityTTL)) {
		t.Fatalf("expiresAt = %s", expiresAt)
	}
	claims, err := VerifyCapability(secret, token, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != accountID || claims.ID != requestID || claims.Grant != CapabilityGrant ||
		claims.ContentType != "image/jpeg" || claims.MaxBytes != MaxDocumentBytes {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestCapabilityRejectsInvalidTokens(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	secret := []byte(strings.Repeat("s", 32))
	accountID := uuid.NewString()
	requestID := uuid.NewString()
	valid, _, err := MintCapability(secret, now, accountID, requestID, "image/png")
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"missing":      "",
		"malformed":    "not-a-token",
		"wrong secret": mustCapability(t, []byte(strings.Repeat("x", 32)), now, accountID, requestID),
		"expired":      valid,
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			verifyAt := now.Add(time.Second)
			if name == "expired" {
				verifyAt = now.Add(CapabilityTTL + time.Second)
			}
			if _, err := VerifyCapability(secret, token, verifyAt); !errors.Is(err, ErrInvalidCapability) {
				t.Fatalf("expected invalid capability, got %v", err)
			}
		})
	}

	wrongAudience := CapabilityClaims{
		Grant: CapabilityGrant, ContentType: "image/png", MaxBytes: MaxDocumentBytes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: CapabilityIssuer, Subject: accountID, Audience: jwt.ClaimStrings{"wrong"},
			IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(CapabilityTTL)), ID: requestID,
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongAudience).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyCapability(secret, token, now.Add(time.Second)); !errors.Is(err, ErrInvalidCapability) {
		t.Fatalf("expected wrong audience rejection, got %v", err)
	}
}

func TestMintCapabilityRejectsInvalidEnvelope(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	now := time.Now()
	for name, values := range map[string][3]string{
		"account": {"not-a-uuid", uuid.NewString(), "image/jpeg"},
		"request": {uuid.NewString(), "not-a-uuid", "image/jpeg"},
		"type":    {uuid.NewString(), uuid.NewString(), "image/webp"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := MintCapability(secret, now, values[0], values[1], values[2]); !errors.Is(err, ErrInvalidCapability) {
				t.Fatalf("expected invalid capability, got %v", err)
			}
		})
	}
}

func mustCapability(t *testing.T, secret []byte, now time.Time, accountID, requestID string) string {
	t.Helper()
	token, _, err := MintCapability(secret, now, accountID, requestID, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	return token
}
