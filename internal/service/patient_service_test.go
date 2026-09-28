package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hospital-middleware/internal/his"
	"hospital-middleware/internal/models"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func newPatientSvc(hospitals *fakeHospitalRepo, patients *fakePatientRepo, hisClient *fakeHIS) PatientService {
	return NewPatientService(hospitals, patients, hisClient, discardLogger)
}

func TestPatientSearch_NoIDs_QueriesLocalOnly(t *testing.T) {
	hisClient := &fakeHIS{}
	patients := &fakePatientRepo{result: []models.Patient{{ID: 1, PatientHN: "HN-A-0001"}}}
	svc := newPatientSvc(newFakeHospitalRepo(hospitalA), patients, hisClient)

	filter := models.PatientSearchFilter{FirstName: "som"}
	res, err := svc.Search(context.Background(), hospitalA.ID, filter)
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Empty(t, hisClient.calls, "HIS is only queried by national/passport id")
	assert.Equal(t, hospitalA.ID, patients.searchedHospital)
	assert.Equal(t, filter, patients.searchedFilter)
}

func TestPatientSearch_FetchesFromHISAndUpserts(t *testing.T) {
	hisClient := &fakeHIS{patients: map[string]*his.Patient{
		"1103700000033": {PatientHN: "HN-A-0003", FirstNameEN: "New", NationalID: "1103700000033", Gender: "F"},
	}}
	patients := &fakePatientRepo{}
	svc := newPatientSvc(newFakeHospitalRepo(hospitalA), patients, hisClient)

	_, err := svc.Search(context.Background(), hospitalA.ID, models.PatientSearchFilter{NationalID: " 1103700000033 "})
	require.NoError(t, err)

	assert.Equal(t, []string{"http://his-a|1103700000033"}, hisClient.calls)
	require.Len(t, patients.upserted, 1)
	assert.Equal(t, hospitalA.ID, patients.upserted[0].HospitalID)
	assert.Equal(t, "HN-A-0003", patients.upserted[0].PatientHN)
}

func TestPatientSearch_TriesPassportWhenNationalIDNotInHIS(t *testing.T) {
	hisClient := &fakeHIS{patients: map[string]*his.Patient{"P1": {PatientHN: "HN-9", PassportID: "P1"}}}
	patients := &fakePatientRepo{}
	svc := newPatientSvc(newFakeHospitalRepo(hospitalA), patients, hisClient)

	_, err := svc.Search(context.Background(), hospitalA.ID, models.PatientSearchFilter{NationalID: "missing", PassportID: "P1"})
	require.NoError(t, err)
	assert.Equal(t, []string{"http://his-a|missing", "http://his-a|P1"}, hisClient.calls)
	assert.Len(t, patients.upserted, 1)
}

func TestPatientSearch_HospitalWithoutHIS(t *testing.T) {
	hisClient := &fakeHIS{}
	svc := newPatientSvc(newFakeHospitalRepo(hospitalB), &fakePatientRepo{}, hisClient)

	_, err := svc.Search(context.Background(), hospitalB.ID, models.PatientSearchFilter{PassportID: "AA1234567"})
	require.NoError(t, err)
	assert.Empty(t, hisClient.calls)
}

func TestPatientSearch_HISFailuresFallBackToLocalData(t *testing.T) {
	local := []models.Patient{{ID: 1, PatientHN: "HN-A-0001"}}

	t.Run("HIS error", func(t *testing.T) {
		patients := &fakePatientRepo{result: local}
		svc := newPatientSvc(newFakeHospitalRepo(hospitalA), patients, &fakeHIS{err: errDB})
		res, err := svc.Search(context.Background(), hospitalA.ID, models.PatientSearchFilter{NationalID: "1"})
		require.NoError(t, err)
		assert.Equal(t, local, res)
		assert.Empty(t, patients.upserted)
	})

	t.Run("upsert error", func(t *testing.T) {
		patients := &fakePatientRepo{result: local, upsertErr: errDB}
		hisClient := &fakeHIS{patients: map[string]*his.Patient{"1": {PatientHN: "HN"}}}
		svc := newPatientSvc(newFakeHospitalRepo(hospitalA), patients, hisClient)
		res, err := svc.Search(context.Background(), hospitalA.ID, models.PatientSearchFilter{NationalID: "1"})
		require.NoError(t, err)
		assert.Equal(t, local, res)
	})

	t.Run("hospital lookup error", func(t *testing.T) {
		hospitals := newFakeHospitalRepo(hospitalA)
		hospitals.err = errDB
		hisClient := &fakeHIS{}
		svc := newPatientSvc(hospitals, &fakePatientRepo{result: local}, hisClient)
		res, err := svc.Search(context.Background(), hospitalA.ID, models.PatientSearchFilter{NationalID: "1"})
		require.NoError(t, err)
		assert.Equal(t, local, res)
		assert.Empty(t, hisClient.calls)
	})
}

func TestPatientSearch_LocalSearchError(t *testing.T) {
	svc := newPatientSvc(newFakeHospitalRepo(hospitalA), &fakePatientRepo{searchErr: errDB}, &fakeHIS{})
	_, err := svc.Search(context.Background(), hospitalA.ID, models.PatientSearchFilter{})
	assert.ErrorIs(t, err, errDB)
}
