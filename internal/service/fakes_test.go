package service

import (
	"context"
	"errors"
	"time"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/his"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
)

var errDB = errors.New("db down")

func strPtr(s string) *string { return &s }

type fakeHospitalRepo struct {
	byCode map[string]*models.Hospital
	err    error
}

func newFakeHospitalRepo(hs ...*models.Hospital) *fakeHospitalRepo {
	r := &fakeHospitalRepo{byCode: map[string]*models.Hospital{}}
	for _, h := range hs {
		r.byCode[h.Code] = h
	}
	return r
}

func (r *fakeHospitalRepo) GetByCode(_ context.Context, code string) (*models.Hospital, error) {
	if r.err != nil {
		return nil, r.err
	}
	if h, ok := r.byCode[code]; ok {
		return h, nil
	}
	return nil, repository.ErrNotFound
}

func (r *fakeHospitalRepo) GetByID(_ context.Context, id int64) (*models.Hospital, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, h := range r.byCode {
		if h.ID == id {
			return h, nil
		}
	}
	return nil, repository.ErrNotFound
}

type fakeStaffRepo struct {
	staff     []*models.Staff
	createErr error
	getErr    error
}

func (r *fakeStaffRepo) Create(_ context.Context, s *models.Staff) error {
	if r.createErr != nil {
		return r.createErr
	}
	for _, existing := range r.staff {
		if existing.HospitalID == s.HospitalID && existing.Username == s.Username {
			return repository.ErrDuplicate
		}
	}
	s.ID = int64(len(r.staff) + 1)
	r.staff = append(r.staff, s)
	return nil
}

func (r *fakeStaffRepo) GetByUsername(_ context.Context, hospitalID int64, username string) (*models.Staff, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	for _, s := range r.staff {
		if s.HospitalID == hospitalID && s.Username == username {
			return s, nil
		}
	}
	return nil, repository.ErrNotFound
}

type fakePatientRepo struct {
	result    []models.Patient
	searchErr error
	upsertErr error

	searchedHospital int64
	searchedFilter   models.PatientSearchFilter
	upserted         []models.Patient
}

func (r *fakePatientRepo) Search(_ context.Context, hospitalID int64, f models.PatientSearchFilter) ([]models.Patient, error) {
	r.searchedHospital, r.searchedFilter = hospitalID, f
	return r.result, r.searchErr
}

func (r *fakePatientRepo) Upsert(_ context.Context, p *models.Patient) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.upserted = append(r.upserted, *p)
	return nil
}

// fakeHasher "hashes" by prefixing, which is enough to test the flow.
type fakeHasher struct{ err error }

func (h fakeHasher) Hash(p string) (string, error) {
	if h.err != nil {
		return "", h.err
	}
	return "hashed:" + p, nil
}
func (fakeHasher) Compare(hash, p string) bool { return hash == "hashed:"+p }

type fakeTokens struct{ err error }

func (t fakeTokens) Generate(staffID int64, _ string, hospitalID int64, _ string) (string, time.Time, error) {
	if t.err != nil {
		return "", time.Time{}, t.err
	}
	return "token", time.Unix(1_700_000_000, 0), nil
}

func (fakeTokens) Parse(string) (*auth.Claims, error) { return nil, nil }

type fakeHIS struct {
	patients map[string]*his.Patient // keyed by id
	err      error
	calls    []string
}

func (f *fakeHIS) SearchPatient(_ context.Context, baseURL, id string) (*his.Patient, error) {
	f.calls = append(f.calls, baseURL+"|"+id)
	if f.err != nil {
		return nil, f.err
	}
	if p, ok := f.patients[id]; ok {
		return p, nil
	}
	return nil, his.ErrPatientNotFound
}
