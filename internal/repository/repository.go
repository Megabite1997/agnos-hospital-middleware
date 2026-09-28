// Package repository is the Postgres data-access layer. Each repository is
// defined by an interface so services can be unit tested with fakes.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"hospital-middleware/internal/models"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("record already exists")
)

type HospitalRepository interface {
	GetByCode(ctx context.Context, code string) (*models.Hospital, error)
	GetByID(ctx context.Context, id int64) (*models.Hospital, error)
}

type StaffRepository interface {
	Create(ctx context.Context, staff *models.Staff) error
	GetByUsername(ctx context.Context, hospitalID int64, username string) (*models.Staff, error)
}

type PatientRepository interface {
	// Search returns patients of hospitalID matching every non-empty filter.
	Search(ctx context.Context, hospitalID int64, filter models.PatientSearchFilter) ([]models.Patient, error)
	// Upsert inserts the patient or updates the row with the same
	// (hospital_id, patient_hn), filling in ID and timestamps.
	Upsert(ctx context.Context, patient *models.Patient) error
}

// DB is the subset of pgxpool.Pool the repositories use.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var _ DB = (*pgxpool.Pool)(nil)

// mapError translates driver errors into repository sentinel errors.
func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return ErrDuplicate
	}
	return err
}
