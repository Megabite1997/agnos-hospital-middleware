package repository

import (
	"context"
	"fmt"
	"strings"

	"hospital-middleware/internal/models"
)

const (
	DefaultSearchLimit = 20
	MaxSearchLimit     = 100
)

type patientRepository struct {
	db DB
}

func NewPatientRepository(db DB) PatientRepository {
	return &patientRepository{db: db}
}

const patientColumns = `id, hospital_id, patient_hn,
	first_name_th, middle_name_th, last_name_th,
	first_name_en, middle_name_en, last_name_en,
	date_of_birth, national_id, passport_id, phone_number, email, gender,
	created_at, updated_at`

func (r *patientRepository) Search(ctx context.Context, hospitalID int64, f models.PatientSearchFilter) ([]models.Patient, error) {
	sql, args := buildSearchQuery(hospitalID, f)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	defer rows.Close()

	patients := make([]models.Patient, 0)
	for rows.Next() {
		var p models.Patient
		if err := rows.Scan(patientScanTargets(&p)...); err != nil {
			return nil, fmt.Errorf("scan patient: %w", err)
		}
		patients = append(patients, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patients: %w", err)
	}
	return patients, nil
}

func (r *patientRepository) Upsert(ctx context.Context, p *models.Patient) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO patients (hospital_id, patient_hn,
			first_name_th, middle_name_th, last_name_th,
			first_name_en, middle_name_en, last_name_en,
			date_of_birth, national_id, passport_id, phone_number, email, gender)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (hospital_id, patient_hn) DO UPDATE SET
			first_name_th  = EXCLUDED.first_name_th,
			middle_name_th = EXCLUDED.middle_name_th,
			last_name_th   = EXCLUDED.last_name_th,
			first_name_en  = EXCLUDED.first_name_en,
			middle_name_en = EXCLUDED.middle_name_en,
			last_name_en   = EXCLUDED.last_name_en,
			date_of_birth  = EXCLUDED.date_of_birth,
			national_id    = EXCLUDED.national_id,
			passport_id    = EXCLUDED.passport_id,
			phone_number   = EXCLUDED.phone_number,
			email          = EXCLUDED.email,
			gender         = EXCLUDED.gender,
			updated_at     = NOW()
		RETURNING id, created_at, updated_at`,
		p.HospitalID, p.PatientHN,
		p.FirstNameTH, p.MiddleNameTH, p.LastNameTH,
		p.FirstNameEN, p.MiddleNameEN, p.LastNameEN,
		p.DateOfBirth, p.NationalID, p.PassportID, p.PhoneNumber, p.Email, p.Gender,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	return mapError(err)
}

func patientScanTargets(p *models.Patient) []any {
	return []any{
		&p.ID, &p.HospitalID, &p.PatientHN,
		&p.FirstNameTH, &p.MiddleNameTH, &p.LastNameTH,
		&p.FirstNameEN, &p.MiddleNameEN, &p.LastNameEN,
		&p.DateOfBirth, &p.NationalID, &p.PassportID, &p.PhoneNumber, &p.Email, &p.Gender,
		&p.CreatedAt, &p.UpdatedAt,
	}
}

// buildSearchQuery builds a parameterised query. The hospital condition is
// always present, so a staff member can never see another hospital's patients.
//
// Matching rules:
//   - national_id, passport_id, phone_number, date_of_birth: exact match
//   - email: case-insensitive exact match
//   - first/middle/last name: case-insensitive partial match against both the
//     Thai and the English name columns
func buildSearchQuery(hospitalID int64, f models.PatientSearchFilter) (string, []any) {
	conds := []string{"hospital_id = $1"}
	args := []any{hospitalID}

	add := func(format string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(format, len(args)))
	}

	if v := strings.TrimSpace(f.NationalID); v != "" {
		add("national_id = $%d", v)
	}
	if v := strings.TrimSpace(f.PassportID); v != "" {
		add("passport_id = $%d", v)
	}
	if v := strings.TrimSpace(f.FirstName); v != "" {
		add("(first_name_th ILIKE $%[1]d OR first_name_en ILIKE $%[1]d)", containsPattern(v))
	}
	if v := strings.TrimSpace(f.MiddleName); v != "" {
		add("(middle_name_th ILIKE $%[1]d OR middle_name_en ILIKE $%[1]d)", containsPattern(v))
	}
	if v := strings.TrimSpace(f.LastName); v != "" {
		add("(last_name_th ILIKE $%[1]d OR last_name_en ILIKE $%[1]d)", containsPattern(v))
	}
	if f.DateOfBirth != nil {
		add("date_of_birth = $%d", *f.DateOfBirth)
	}
	if v := strings.TrimSpace(f.PhoneNumber); v != "" {
		add("phone_number = $%d", v)
	}
	if v := strings.TrimSpace(f.Email); v != "" {
		add("LOWER(email) = LOWER($%d)", v)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	offset := max(f.Offset, 0)
	args = append(args, limit, offset)

	sql := fmt.Sprintf(`SELECT %s FROM patients WHERE %s ORDER BY id LIMIT $%d OFFSET $%d`,
		patientColumns, strings.Join(conds, " AND "), len(args)-1, len(args))
	return sql, args
}

// containsPattern wraps v in % for ILIKE, escaping LIKE metacharacters so
// user input is matched literally.
func containsPattern(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(v) + "%"
}
