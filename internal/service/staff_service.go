// Package service holds the business logic between HTTP handlers and
// repositories.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
)

var (
	ErrHospitalNotFound   = errors.New("hospital not found")
	ErrUsernameTaken      = errors.New("username already exists in this hospital")
	ErrInvalidCredentials = errors.New("invalid username, password or hospital")
)

type CreateStaffInput struct {
	Username string
	Password string
	Hospital string // hospital code, e.g. "hospital-a"
}

type LoginInput = CreateStaffInput

type LoginResult struct {
	AccessToken string
	ExpiresAt   time.Time
	Staff       *models.Staff
	Hospital    *models.Hospital
}

type StaffService interface {
	Create(ctx context.Context, in CreateStaffInput) (*models.Staff, *models.Hospital, error)
	Login(ctx context.Context, in LoginInput) (*LoginResult, error)
}

type staffService struct {
	hospitals repository.HospitalRepository
	staff     repository.StaffRepository
	hasher    auth.PasswordHasher
	tokens    auth.TokenManager
}

func NewStaffService(
	hospitals repository.HospitalRepository,
	staff repository.StaffRepository,
	hasher auth.PasswordHasher,
	tokens auth.TokenManager,
) StaffService {
	return &staffService{hospitals: hospitals, staff: staff, hasher: hasher, tokens: tokens}
}

func (s *staffService) Create(ctx context.Context, in CreateStaffInput) (*models.Staff, *models.Hospital, error) {
	hospital, err := s.hospitals.GetByCode(ctx, normalizeHospital(in.Hospital))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrHospitalNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get hospital: %w", err)
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	staff := &models.Staff{
		HospitalID:   hospital.ID,
		Username:     strings.TrimSpace(in.Username),
		PasswordHash: hash,
	}
	if err := s.staff.Create(ctx, staff); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, nil, ErrUsernameTaken
		}
		return nil, nil, fmt.Errorf("create staff: %w", err)
	}
	return staff, hospital, nil
}

// Login returns the same ErrInvalidCredentials for an unknown hospital,
// unknown user or wrong password so callers cannot probe which one it was.
func (s *staffService) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	hospital, err := s.hospitals.GetByCode(ctx, normalizeHospital(in.Hospital))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("get hospital: %w", err)
	}

	staff, err := s.staff.GetByUsername(ctx, hospital.ID, strings.TrimSpace(in.Username))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("get staff: %w", err)
	}

	if !s.hasher.Compare(staff.PasswordHash, in.Password) {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.tokens.Generate(staff.ID, staff.Username, hospital.ID, hospital.Code)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	return &LoginResult{AccessToken: token, ExpiresAt: expiresAt, Staff: staff, Hospital: hospital}, nil
}

func normalizeHospital(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}
