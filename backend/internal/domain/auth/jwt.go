package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Subject string

const (
	SubjectStudent Subject = "student"
	SubjectTeam    Subject = "team_member"
)

type Claims struct {
	SubjectType Subject `json:"subj"`
	Role        string  `json:"role"`
	// VendorID scopes a team member to one partner sector (P5, AYO, ...). Empty means
	// unrestricted access (P5 admins overseeing every sector).
	VendorID string `json:"vendorId,omitempty"`
	jwt.RegisteredClaims
}

// UserID returns the authenticated principal's id (the JWT "sub" claim).
func (c *Claims) UserID() string {
	return c.RegisteredClaims.Subject
}

type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenIssuer(secret string) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: 7 * 24 * time.Hour}
}

func (i *TokenIssuer) Issue(userID string, subject Subject, role string) (string, error) {
	return i.issue(userID, subject, role, "")
}

func (i *TokenIssuer) IssueTeam(userID, role, vendorID string) (string, error) {
	return i.issue(userID, SubjectTeam, role, vendorID)
}

func (i *TokenIssuer) issue(userID string, subject Subject, role, vendorID string) (string, error) {
	claims := Claims{
		SubjectType: subject,
		Role:        role,
		VendorID:    vendorID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(i.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}

func (i *TokenIssuer) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return i.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
