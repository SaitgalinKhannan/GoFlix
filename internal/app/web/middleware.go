package web

import (
	"GoFlix/internal/app/auth"
	"context"
	"net/http"
)

type ctxKey int

const claimsCtxKey ctxKey = iota

// JWTAuth middleware validates an `Authorization: Bearer <access-token>` header
// and stores the claims in the request context. Unauthenticated requests get 401.
func JWTAuth(svc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := auth.ExtractBearer(r)
			if !ok {
				http.Error(w, `{"error":"missing or invalid Authorization header"}`, http.StatusUnauthorized)
				return
			}
			claims, err := svc.Verify(tokenStr, auth.AccessTokenKind())
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsCtxKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext returns the JWT claims stored by JWTAuth, or nil.
func ClaimsFromContext(r *http.Request) *auth.Claims {
	if v, ok := r.Context().Value(claimsCtxKey).(*auth.Claims); ok {
		return v
	}
	return nil
}
