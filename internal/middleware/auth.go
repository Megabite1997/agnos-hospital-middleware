// Package middleware contains Gin middleware.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/auth"
)

const claimsKey = "staff_claims"

// RequireAuth rejects requests without a valid "Authorization: Bearer <jwt>"
// header and stores the token claims on the context for handlers.
func RequireAuth(tokens auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		scheme, token, ok := strings.Cut(header, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			abortUnauthorized(c, "missing or malformed Authorization header")
			return
		}

		claims, err := tokens.Parse(strings.TrimSpace(token))
		if err != nil {
			abortUnauthorized(c, "invalid or expired token")
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

// ClaimsFrom returns the claims set by RequireAuth.
func ClaimsFrom(c *gin.Context) (*auth.Claims, bool) {
	v, ok := c.Get(claimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*auth.Claims)
	return claims, ok
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.Header("WWW-Authenticate", `Bearer realm="hospital-middleware"`)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{"code": "UNAUTHORIZED", "message": msg},
	})
}
