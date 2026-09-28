package repository

import (
	"context"

	"hospital-middleware/internal/models"
)

type hospitalRepository struct {
	db DB
}

func NewHospitalRepository(db DB) HospitalRepository {
	return &hospitalRepository{db: db}
}

const hospitalColumns = `id, code, name, his_base_url, created_at, updated_at`

func (r *hospitalRepository) GetByCode(ctx context.Context, code string) (*models.Hospital, error) {
	return r.getOne(ctx, `SELECT `+hospitalColumns+` FROM hospitals WHERE code = $1`, code)
}

func (r *hospitalRepository) GetByID(ctx context.Context, id int64) (*models.Hospital, error) {
	return r.getOne(ctx, `SELECT `+hospitalColumns+` FROM hospitals WHERE id = $1`, id)
}

func (r *hospitalRepository) getOne(ctx context.Context, sql string, arg any) (*models.Hospital, error) {
	var h models.Hospital
	err := r.db.QueryRow(ctx, sql, arg).Scan(&h.ID, &h.Code, &h.Name, &h.HISBaseURL, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, mapError(err)
	}
	return &h, nil
}
