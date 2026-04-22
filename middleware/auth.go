package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
)

const AuthUserId = "auth.userId"

type KeyValidator interface {
	ValidateKey(key string) error
}

func writeUnauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
}

func Auth(s KeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")

			if !strings.HasSuffix(authorization, "Bearer ") {
				writeUnauthorized(w)
				return
			}

			encodedToken := strings.TrimPrefix(authorization, "Bearer ")

			token, err := base64.StdEncoding.DecodeString(encodedToken)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			userId := string(token)

			ctx := context.WithValue(r.Context(), AuthUserId, userId)
			req := r.WithContext(ctx)

			next.ServeHTTP(w, req)
		})
	}
}
