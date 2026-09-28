package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hospital-middleware/internal/models"
)

var (
	hospitalA = &models.Hospital{ID: 1, Code: "hospital-a", Name: "Hospital A", HISBaseURL: strPtr("http://his-a")}
	hospitalB = &models.Hospital{ID: 2, Code: "hospital-b", Name: "Hospital B"}
)

func newStaffSvc(staff *fakeStaffRepo, hasher fakeHasher, tokens fakeTokens) StaffService {
	return NewStaffService(newFakeHospitalRepo(hospitalA, hospitalB), staff, hasher, tokens)
}

func TestStaffCreate_Success(t *testing.T) {
	repo := &fakeStaffRepo{}
	svc := newStaffSvc(repo, fakeHasher{}, fakeTokens{})

	staff, hospital, err := svc.Create(context.Background(), CreateStaffInput{
		Username: " alice ", Password: "password123", Hospital: " Hospital-A ",
	})
	require.NoError(t, err)
	assert.Equal(t, "alice", staff.Username)
	assert.Equal(t, hospitalA.ID, staff.HospitalID)
	assert.Equal(t, "hashed:password123", staff.PasswordHash)
	assert.Equal(t, hospitalA, hospital)
	assert.Len(t, repo.staff, 1)
}

func TestStaffCreate_SameUsernameDifferentHospitals(t *testing.T) {
	svc := newStaffSvc(&fakeStaffRepo{}, fakeHasher{}, fakeTokens{})
	ctx := context.Background()

	_, _, err := svc.Create(ctx, CreateStaffInput{Username: "alice", Password: "password123", Hospital: "hospital-a"})
	require.NoError(t, err)
	_, _, err = svc.Create(ctx, CreateStaffInput{Username: "alice", Password: "password123", Hospital: "hospital-b"})
	require.NoError(t, err)
	_, _, err = svc.Create(ctx, CreateStaffInput{Username: "alice", Password: "password123", Hospital: "hospital-a"})
	assert.ErrorIs(t, err, ErrUsernameTaken)
}

func TestStaffCreate_Errors(t *testing.T) {
	tests := []struct {
		name     string
		repo     *fakeStaffRepo
		hasher   fakeHasher
		hospital string
		hospErr  error
		wantErr  error
		wantMsg  string
	}{
		{name: "unknown hospital", repo: &fakeStaffRepo{}, hospital: "hospital-z", wantErr: ErrHospitalNotFound},
		{name: "hospital lookup fails", repo: &fakeStaffRepo{}, hospital: "hospital-a", hospErr: errDB, wantMsg: "get hospital"},
		{name: "hash fails", repo: &fakeStaffRepo{}, hasher: fakeHasher{err: errors.New("x")}, hospital: "hospital-a", wantMsg: "hash password"},
		{name: "insert fails", repo: &fakeStaffRepo{createErr: errDB}, hospital: "hospital-a", wantMsg: "create staff"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hospitals := newFakeHospitalRepo(hospitalA, hospitalB)
			hospitals.err = tc.hospErr
			svc := NewStaffService(hospitals, tc.repo, tc.hasher, fakeTokens{})

			_, _, err := svc.Create(context.Background(), CreateStaffInput{Username: "u", Password: "password123", Hospital: tc.hospital})
			require.Error(t, err)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.ErrorContains(t, err, tc.wantMsg)
			}
		})
	}
}

func seededStaffRepo() *fakeStaffRepo {
	return &fakeStaffRepo{staff: []*models.Staff{
		{ID: 5, HospitalID: hospitalA.ID, Username: "alice", PasswordHash: "hashed:password123"},
	}}
}

func TestStaffLogin_Success(t *testing.T) {
	svc := newStaffSvc(seededStaffRepo(), fakeHasher{}, fakeTokens{})

	res, err := svc.Login(context.Background(), LoginInput{Username: "alice", Password: "password123", Hospital: "HOSPITAL-A"})
	require.NoError(t, err)
	assert.Equal(t, "token", res.AccessToken)
	assert.Equal(t, int64(5), res.Staff.ID)
	assert.Equal(t, hospitalA, res.Hospital)
	assert.False(t, res.ExpiresAt.IsZero())
}

func TestStaffLogin_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name string
		in   LoginInput
	}{
		{"wrong password", LoginInput{Username: "alice", Password: "nope-nope", Hospital: "hospital-a"}},
		{"unknown user", LoginInput{Username: "bob", Password: "password123", Hospital: "hospital-a"}},
		{"unknown hospital", LoginInput{Username: "alice", Password: "password123", Hospital: "hospital-z"}},
		{"user from another hospital", LoginInput{Username: "alice", Password: "password123", Hospital: "hospital-b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newStaffSvc(seededStaffRepo(), fakeHasher{}, fakeTokens{})
			_, err := svc.Login(context.Background(), tc.in)
			assert.ErrorIs(t, err, ErrInvalidCredentials)
		})
	}
}

func TestStaffLogin_InfrastructureErrors(t *testing.T) {
	in := LoginInput{Username: "alice", Password: "password123", Hospital: "hospital-a"}

	hospitals := newFakeHospitalRepo(hospitalA)
	hospitals.err = errDB
	_, err := NewStaffService(hospitals, seededStaffRepo(), fakeHasher{}, fakeTokens{}).Login(context.Background(), in)
	assert.ErrorContains(t, err, "get hospital")

	repo := seededStaffRepo()
	repo.getErr = errDB
	_, err = newStaffSvc(repo, fakeHasher{}, fakeTokens{}).Login(context.Background(), in)
	assert.ErrorContains(t, err, "get staff")

	_, err = newStaffSvc(seededStaffRepo(), fakeHasher{}, fakeTokens{err: errors.New("x")}).Login(context.Background(), in)
	assert.ErrorContains(t, err, "generate token")
}
