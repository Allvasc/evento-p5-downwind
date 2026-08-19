package qrcode

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"strings"
)

const prefix = "p5_"

// GenerateToken creates an opaque, unguessable ticket token: a random identifier plus
// an HMAC signature over it. The signature lets the check-in scanner reject a forged or
// tampered code immediately, without a database round-trip, before ever looking it up.
func GenerateToken(secret string) (token string, lookupKey string, err error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	lookupKey = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	sig := sign(lookupKey, secret)
	return prefix + lookupKey + "." + sig, lookupKey, nil
}

// Verify checks the token's structure and signature. It does not confirm the ticket
// exists or is still valid — that's a status lookup by lookupKey against entitlements.
func Verify(token, secret string) (lookupKey string, ok bool) {
	body := strings.TrimPrefix(token, prefix)
	parts := strings.SplitN(body, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	lookupKey, sig := parts[0], parts[1]
	expected := sign(lookupKey, secret)
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return "", false
	}
	return lookupKey, true
}

func sign(lookupKey, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(lookupKey))
	sum := mac.Sum(nil)
	return fmt.Sprintf("%x", sum)[:16]
}
