package auth

import "testing"

func TestIsValidCPF(t *testing.T) {
	cases := []struct {
		name string
		cpf  string
		want bool
	}{
		{"empty is valid (optional field)", "", true},
		{"valid, unformatted", "52998224725", true},
		{"valid, formatted", "529.982.247-25", true},
		{"wrong check digits", "52998224726", false},
		{"all same digit", "11111111111", false},
		{"too short", "1234567890", false},
		{"too long", "123456789012", false},
		{"non-numeric garbage only", "abc.def-gh", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsValidCPF(c.cpf); got != c.want {
				t.Errorf("IsValidCPF(%q) = %v, want %v", c.cpf, got, c.want)
			}
		})
	}
}

func TestNormalizeCPF(t *testing.T) {
	if got := NormalizeCPF("529.982.247-25"); got != "52998224725" {
		t.Errorf("NormalizeCPF() = %q, want %q", got, "52998224725")
	}
	if got := NormalizeCPF(""); got != "" {
		t.Errorf("NormalizeCPF(empty) = %q, want empty", got)
	}
}

func TestLastFour(t *testing.T) {
	if got := LastFour("529.982.247-25"); got != "4725" {
		t.Errorf("LastFour() = %q, want %q", got, "4725")
	}
	if got := LastFour("12"); got != "12" {
		t.Errorf("LastFour(short) = %q, want %q", got, "12")
	}
}

func TestHashCPF(t *testing.T) {
	h1 := HashCPF("529.982.247-25", "pepper")
	h2 := HashCPF("52998224725", "pepper")
	if h1 != h2 {
		t.Error("HashCPF should be stable across formatting differences (normalizes first)")
	}
	if HashCPF("52998224725", "other-pepper") == h1 {
		t.Error("HashCPF should differ when the pepper changes")
	}
	if HashCPF("11144477735", "pepper") == h1 {
		t.Error("HashCPF should differ for a different CPF")
	}
}
