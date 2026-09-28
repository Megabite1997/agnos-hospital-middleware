// Package auth issues and verifies staff access tokens (JWT, HS256).
package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid or expired token")

const issuer = "hospital-middleware"

// Claims identifies the staff member and, crucially, the hospital whose
// patients they are allowed to see.
type Claims struct {
	StaffID      int64  `json:"staff_id"`
	Username     string `json:"username"`
	HospitalID   int64  `json:"hospital_id"`
	HospitalCode string `json:"hospital_code"`
	jwt.RegisteredClaims
}

type TokenManager interface {
	Generate(staffID int64, username string, hospitalID int64, hospitalCode string) (token string, expiresAt time.Time, err error)
	Parse(token string) (*Claims, error)
}

type jwtManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenManager(secret string, ttl time.Duration) TokenManager {
	return &jwtManager{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (m *jwtManager) Generate(staffID int64, username string, hospitalID int64, hospitalCode string) (string, time.Time, error) {
	now := m.now()
	expiresAt := now.Add(m.ttl)
	claims := Claims{
		StaffID:      staffID,
		Username:     username,
		HospitalID:   hospitalID,
		HospitalCode: hospitalCode,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatInt(staffID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

func (m *jwtManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || claims.StaffID == 0 || claims.HospitalID == 0 {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
