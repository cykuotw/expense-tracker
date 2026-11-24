package ocr

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// Sign the accepted scan independently of photo retention and async usage delivery.
// The subject and request ID are canonical UUIDs from the verified capability.
func scanProof(secret []byte, accountID, requestID, note string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("confirmed-receipt-scan/v1\x00" + accountID + "\x00" + requestID + "\x00" + note))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func VerifyScanProof(secret []byte, accountID, requestID, note, proof string) bool {
	if len(secret) < 32 || len(requestID) != 36 || len(note) > MaxScanNoteBytes {
		return false
	}
	return hmac.Equal([]byte(proof), []byte(scanProof(secret, accountID, requestID, note)))
}
