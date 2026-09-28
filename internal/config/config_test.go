package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validSecret = strings.Repeat("s", 32)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("JWT_SECRET", validSecret)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, 24*time.Hour, cfg.JWTTTL)
	assert.Equal(t, 5*time.Second, cfg.HISTimeout)
	assert.Equal(t, "release", cfg.GinMode)
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("PORT", "9000")
	t.Setenv("JWT_TTL", "1h")
	t.Setenv("HIS_TIMEOUT", "2s")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "9000", cfg.Port)
	assert.Equal(t, time.Hour, cfg.JWTTTL)
	assert.Equal(t, 2*time.Second, cfg.HISTimeout)
}

func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"missing database url", map[string]string{"JWT_SECRET": validSecret}, "DATABASE_URL"},
		{"missing secret", map[string]string{"DATABASE_URL": "x"}, "JWT_SECRET"},
		{"short secret", map[string]string{"DATABASE_URL": "x", "JWT_SECRET": "short"}, "JWT_SECRET"},
		{"bad ttl", map[string]string{"DATABASE_URL": "x", "JWT_SECRET": validSecret, "JWT_TTL": "forever"}, "JWT_TTL"},
		{"bad his timeout", map[string]string{"DATABASE_URL": "x", "JWT_SECRET": validSecret, "HIS_TIMEOUT": "soon"}, "HIS_TIMEOUT"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range []string{"DATABASE_URL", "JWT_SECRET", "JWT_TTL", "HIS_TIMEOUT"} {
				t.Setenv(k, tc.env[k])
			}
			_, err := Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
