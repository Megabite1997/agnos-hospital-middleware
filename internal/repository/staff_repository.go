package repository

import (
	"context"

	"hospital-middleware/internal/models"
)

type staffRepository struct {
	db DB
}

func NewStaffRepository(db DB) StaffRepository {
	return &staffRepository{db: db}
}

func (r *staffRepository) Create(ctx context.Context, s *models.Staff) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO staff (hospital_id, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`,
		s.HospitalID, s.Username, s.PasswordHash,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	return mapError(err)
}

func (r *staffRepository) GetByUsername(ctx context.Context, hospitalID int64, username string) (*models.Staff, error) {
	var s models.Staff
	err := r.db.QueryRow(ctx, `
		SELECT id, hospital_id, username, password_hash, created_at, updated_at
		FROM staff
		WHERE hospital_id = $1 AND username = $2`,
		hospitalID, username,
	).Scan(&s.ID, &s.HospitalID, &s.Username, &s.PasswordHash, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &s, nil
}
