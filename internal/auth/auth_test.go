package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var secret = strings.Repeat("k", 32)

func TestToken_GenerateAndParse(t *testing.T) {
	m := NewTokenManager(secret, time.Hour)

	token, exp, err := m.Generate(10, "alice", 2, "hospital-a")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), exp, 5*time.Second)

	claims, err := m.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, int64(10), claims.StaffID)
	assert.Equal(t, "alice", claims.Username)
	assert.Equal(t, int64(2), claims.HospitalID)
	assert.Equal(t, "hospital-a", claims.HospitalCode)
	assert.Equal(t, "10", claims.Subject)
}

func TestToken_Expired(t *testing.T) {
	m := NewTokenManager(secret, time.Minute).(*jwtManager)
	past := time.Now().Add(-time.Hour)
	m.now = func() time.Time { return past }
	token, _, err := m.Generate(1, "a", 1, "h")
	require.NoError(t, err)

	m.now = time.Now
	_, err = m.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestToken_WrongSecret(t *testing.T) {
	token, _, err := NewTokenManager(secret, time.Hour).Generate(1, "a", 1, "h")
	require.NoError(t, err)

	_, err = NewTokenManager(strings.Repeat("x", 32), time.Hour).Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestToken_Malformed(t *testing.T) {
	_, err := NewTokenManager(secret, time.Hour).Parse("not-a-jwt")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestToken_RejectsOtherAlgorithms(t *testing.T) {
	claims := Claims{StaffID: 1, HospitalID: 1, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(secret))
	require.NoError(t, err)

	_, err = NewTokenManager(secret, time.Hour).Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestToken_RejectsMissingHospital(t *testing.T) {
	claims := Claims{StaffID: 1, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)

	_, err = NewTokenManager(secret, time.Hour).Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestBcryptHasher(t *testing.T) {
	h := bcryptHasher{cost: 4} // minimum cost keeps the test fast
	hash, err := h.Hash("s3cret-pass")
	require.NoError(t, err)
	assert.NotEqual(t, "s3cret-pass", hash)
	assert.True(t, h.Compare(hash, "s3cret-pass"))
	assert.False(t, h.Compare(hash, "wrong"))
	assert.False(t, h.Compare("not-a-hash", "s3cret-pass"))
}

func TestNewBcryptHasher(t *testing.T) {
	assert.NotNil(t, NewBcryptHasher())
}
