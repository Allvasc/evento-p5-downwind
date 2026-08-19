package auth

import "testing"

// TestHashPassword_LongPasswordAndPepper is a regression test for a real production
// incident: bcrypt hard-fails on inputs over 72 bytes, and password+pepper alone
// already approached that limit with a properly generated 32-byte-hex pepper (64
// chars) plus any password longer than ~8 characters — exactly the combination
// PASSWORD_PEPPER's own doc comment recommends generating.
func TestHashPassword_LongPasswordAndPepper(t *testing.T) {
	pepper := "3a8575f6c44ab03044d7863d3fa260ed1ac5ec23cd49bac7fa1ea393b87ec57" // 64 hex chars, like openssl rand -hex 32
	password := "29HV918npKtGXJBUuiGX"                                          // 20 chars — a realistic generated password

	hash, err := HashPassword(password, pepper)
	if err != nil {
		t.Fatalf("HashPassword with a real-world pepper+password length failed: %v", err)
	}
	if !CheckPassword(hash, password, pepper) {
		t.Error("CheckPassword did not accept the password it was just hashed with")
	}
	if CheckPassword(hash, "wrong-password", pepper) {
		t.Error("CheckPassword accepted an incorrect password")
	}
}

func TestHashPassword_EvenLongerInputsStillWork(t *testing.T) {
	pepper := "3a8575f6c44ab03044d7863d3fa260ed1ac5ec23cd49bac7fa1ea393b87ec57"
	password := "a-passphrase-a-user-might-reasonably-choose-that-is-quite-long-indeed"

	hash, err := HashPassword(password, pepper)
	if err != nil {
		t.Fatalf("HashPassword failed for a long passphrase: %v", err)
	}
	if !CheckPassword(hash, password, pepper) {
		t.Error("CheckPassword did not accept the long passphrase it was just hashed with")
	}
}
