package auth

import (
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// bcrypt hard-fails on inputs over 72 bytes. A user password concatenated with a
// properly-generated pepper (32 random bytes hex-encoded = 64 characters, exactly what
// PASSWORD_PEPPER is meant to be) already leaves almost no room for the password
// itself before hitting that ceiling. Pre-hashing password+pepper with SHA-256 into a
// fixed 44-byte digest sidesteps the limit entirely, regardless of how long either the
// password or the pepper is.
func combine(password, pepper string) []byte {
	sum := sha256.Sum256([]byte(password + pepper))
	return []byte(base64.StdEncoding.EncodeToString(sum[:]))
}

// HashPassword combines the password with the app-wide pepper before hashing, so a
// leaked bcrypt hash alone (without the pepper, kept only in server config) is not
// enough to brute-force offline.
func HashPassword(password, pepper string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(combine(password, pepper), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password, pepper string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), combine(password, pepper)) == nil
}
