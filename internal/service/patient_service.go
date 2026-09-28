package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"hospital-middleware/internal/his"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
)

type PatientService interface {
	// Search returns patients of hospitalID that match the filter.
	Search(ctx context.Context, hospitalID int64, filter models.PatientSearchFilter) ([]models.Patient, error)
}

type patientService struct {
	hospitals repository.HospitalRepository
	patients  repository.PatientRepository
	his       his.Client
	logger    *slog.Logger
}

func NewPatientService(
	hospitals repository.HospitalRepository,
	patients repository.PatientRepository,
	hisClient his.Client,
	logger *slog.Logger,
) PatientService {
	return &patientService{hospitals: hospitals, patients: patients, his: hisClient, logger: logger}
}

// Search works in two steps:
//
//  1. If the filter has a national_id or passport_id and the staff member's
//     hospital has an HIS API, fetch that patient from the HIS and upsert it
//     locally so the middleware always serves fresh data.
//  2. Query the local database, always scoped to hospitalID.
//
// HIS failures are logged and do not fail the request: the middleware falls
// back to the data it already has.
func (s *patientService) Search(ctx context.Context, hospitalID int64, filter models.PatientSearchFilter) ([]models.Patient, error) {
	s.syncFromHIS(ctx, hospitalID, filter)

	patients, err := s.patients.Search(ctx, hospitalID, filter)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	return patients, nil
}

func (s *patientService) syncFromHIS(ctx context.Context, hospitalID int64, filter models.PatientSearchFilter) {
	ids := make([]string, 0, 2)
	for _, id := range []string{filter.NationalID, filter.PassportID} {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}

	hospital, err := s.hospitals.GetByID(ctx, hospitalID)
	if err != nil {
		s.logger.WarnContext(ctx, "HIS sync skipped: cannot load hospital", "hospital_id", hospitalID, "error", err)
		return
	}
	if hospital.HISBaseURL == nil || *hospital.HISBaseURL == "" {
		return
	}

	for _, id := range ids {
		hisPatient, err := s.his.SearchPatient(ctx, *hospital.HISBaseURL, id)
		if errors.Is(err, his.ErrPatientNotFound) {
			continue
		}
		if err != nil {
			s.logger.WarnContext(ctx, "HIS lookup failed, using local data", "hospital", hospital.Code, "error", err)
			return
		}

		patient := hisPatient.ToModel(hospital.ID)
		if err := s.patients.Upsert(ctx, &patient); err != nil {
			s.logger.WarnContext(ctx, "failed to store HIS patient", "hospital", hospital.Code, "patient_hn", patient.PatientHN, "error", err)
		}
		return // both IDs identify the same person; one hit is enough
	}
}
