package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hospital-middleware/internal/auth"
)

func init() { gin.SetMode(gin.TestMode) }

func newRouter(tokens auth.TokenManager) *gin.Engine {
	r := gin.New()
	r.GET("/protected", RequireAuth(tokens), func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"hospital_id": claims.HospitalID})
	})
	return r
}

func TestRequireAuth(t *testing.T) {
	tokens := auth.NewTokenManager(strings.Repeat("k", 32), time.Hour)
	valid, _, err := tokens.Generate(1, "alice", 42, "hospital-a")
	require.NoError(t, err)

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantBody   string
	}{
		{"valid token", "Bearer " + valid, http.StatusOK, `"hospital_id":42`},
		{"lowercase scheme", "bearer " + valid, http.StatusOK, `"hospital_id":42`},
		{"no header", "", http.StatusUnauthorized, "missing or malformed"},
		{"wrong scheme", "Basic abc", http.StatusUnauthorized, "missing or malformed"},
		{"empty token", "Bearer  ", http.StatusUnauthorized, "missing or malformed"},
		{"invalid token", "Bearer abc.def.ghi", http.StatusUnauthorized, "invalid or expired"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			newRouter(tokens).ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.wantBody)
			if tc.wantStatus == http.StatusUnauthorized {
				assert.NotEmpty(t, w.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestClaimsFrom_Missing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, ok := ClaimsFrom(c)
	assert.False(t, ok)

	c.Set(claimsKey, "not claims")
	_, ok = ClaimsFrom(c)
	assert.False(t, ok)
}
