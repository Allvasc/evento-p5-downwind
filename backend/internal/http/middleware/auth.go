package middleware

import (
	"context"
	"net/http"
	"strings"

	"p5wellness/backend/internal/domain/auth"
)

type contextKey string

const claimsContextKey contextKey = "claims"

func RequireAuth(issuer *auth.TokenIssuer, allowedSubject auth.Subject, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"não autenticado"}`, http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := issuer.Parse(token)
			if err != nil {
				http.Error(w, `{"error":"sessão inválida ou expirada"}`, http.StatusUnauthorized)
				return
			}
			if claims.SubjectType != allowedSubject {
				http.Error(w, `{"error":"acesso não permitido"}`, http.StatusForbidden)
				return
			}
			if len(allowedRoles) > 0 && !contains(allowedRoles, claims.Role) {
				http.Error(w, `{"error":"acesso não permitido"}`, http.StatusForbidden)
				return
			}
			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
