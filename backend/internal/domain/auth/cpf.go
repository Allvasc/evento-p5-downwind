package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// HashCPF returns a deterministic SHA-256 hash of the normalized CPF salted with the
// server-side pepper, used for uniqueness/lookup without keeping the raw CPF indexed.
func HashCPF(cpf, pepper string) string {
	sum := sha256.Sum256([]byte(NormalizeCPF(cpf) + pepper))
	return hex.EncodeToString(sum[:])
}

func LastFour(cpf string) string {
	digits := NormalizeCPF(cpf)
	if len(digits) < 4 {
		return digits
	}
	return digits[len(digits)-4:]
}

// NormalizeCPF strips everything but digits.
func NormalizeCPF(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// IsValidCPF checks the two verification digits of a Brazilian CPF.
// An empty string is considered valid — CPF is optional at registration; callers must
// check emptiness separately if they require it. A non-empty string that normalizes to
// no digits at all (e.g. "abc.def-gh") is NOT the same as empty and is rejected —
// otherwise garbage input would silently be stored as "no CPF" instead of surfacing a
// validation error.
func IsValidCPF(raw string) bool {
	if raw == "" {
		return true
	}
	cpf := NormalizeCPF(raw)
	if len(cpf) != 11 || allSameDigit(cpf) {
		return false
	}

	digits := make([]int, 11)
	for i, r := range cpf {
		digits[i] = int(r - '0')
	}

	if checkDigit(digits[:9], 10) != digits[9] {
		return false
	}
	if checkDigit(digits[:10], 11) != digits[10] {
		return false
	}
	return true
}

func checkDigit(digits []int, startWeight int) int {
	sum := 0
	weight := startWeight
	for _, d := range digits {
		sum += d * weight
		weight--
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

func allSameDigit(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}
