// Package card handles the static identity QR for the SpeedPlus card.
//
// Format: spd.card.v1.{userID}.{HMAC-SHA256}
//
// The payload is static — it never changes per user. It is an identity token,
// not a payment token. The HMAC key is the application JWT secret so the
// signature is deterministic and can be regenerated without invalidating
// existing physical cards.
package card

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// BuildPayload creates the signed QR payload for a user.
func BuildPayload(userID uuid.UUID, secret string) string {
	unsigned := "spd.card.v1." + userID.String()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + hex.EncodeToString(mac.Sum(nil))
}

// VerifyPayload validates a scanned card QR and returns the embedded userID.
// Returns an error if the signature is invalid or the format is wrong.
func VerifyPayload(payload, secret string) (uuid.UUID, error) {
	// Expected: spd.card.v1.{userID}.{sig}  — exactly 5 dot-separated parts
	parts := strings.Split(payload, ".")
	if len(parts) != 5 || parts[0] != "spd" || parts[1] != "card" || parts[2] != "v1" {
		return uuid.Nil, fmt.Errorf("card: invalid payload format")
	}

	userIDStr := parts[3]
	sig := parts[4]
	unsigned := strings.Join(parts[:4], ".")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return uuid.Nil, fmt.Errorf("card: signature invalid")
	}

	return uuid.Parse(userIDStr)
}
