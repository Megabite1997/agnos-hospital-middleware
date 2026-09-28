package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hospital-middleware/internal/database"
	"hospital-middleware/internal/models"
	"hospital-middleware/migrations"
)

// These tests run against a real Postgres and are skipped unless
// TEST_DATABASE_URL is set (see `make test-integration`).
func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `DROP TABLE IF EXISTS patients, staff, hospitals, schema_migrations CASCADE`)
	require.NoError(t, err)
	require.NoError(t, database.Migrate(ctx, pool, migrations.FS))
	// Running twice must be a no-op.
	require.NoError(t, database.Migrate(ctx, pool, migrations.FS))
	return pool
}

func ptr(s string) *string { return &s }

func TestIntegration_Repositories(t *testing.T) {
	pool := setupDB(t)
	ctx := context.Background()

	hospitals := NewHospitalRepository(pool)
	staffRepo := NewStaffRepository(pool)
	patients := NewPatientRepository(pool)

	// Hospitals come from the seed migration.
	hospA, err := hospitals.GetByCode(ctx, "hospital-a")
	require.NoError(t, err)
	require.NotNil(t, hospA.HISBaseURL)
	hospB, err := hospitals.GetByCode(ctx, "hospital-b")
	require.NoError(t, err)
	assert.Nil(t, hospB.HISBaseURL)

	byID, err := hospitals.GetByID(ctx, hospA.ID)
	require.NoError(t, err)
	assert.Equal(t, "hospital-a", byID.Code)

	_, err = hospitals.GetByCode(ctx, "nope")
	assert.ErrorIs(t, err, ErrNotFound)

	t.Run("staff", func(t *testing.T) {
		s := &models.Staff{HospitalID: hospA.ID, Username: "alice", PasswordHash: "hash"}
		require.NoError(t, staffRepo.Create(ctx, s))
		assert.NotZero(t, s.ID)

		// Same username in the same hospital is rejected ...
		err := staffRepo.Create(ctx, &models.Staff{HospitalID: hospA.ID, Username: "alice", PasswordHash: "x"})
		assert.ErrorIs(t, err, ErrDuplicate)
		// ... but allowed in a different hospital.
		require.NoError(t, staffRepo.Create(ctx, &models.Staff{HospitalID: hospB.ID, Username: "alice", PasswordHash: "y"}))

		got, err := staffRepo.GetByUsername(ctx, hospA.ID, "alice")
		require.NoError(t, err)
		assert.Equal(t, "hash", got.PasswordHash)

		_, err = staffRepo.GetByUsername(ctx, hospA.ID, "bob")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("patient search is scoped to hospital", func(t *testing.T) {
		// Seed: national id 1103700000011 exists in both hospitals.
		res, err := patients.Search(ctx, hospA.ID, models.PatientSearchFilter{NationalID: "1103700000011"})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "HN-A-0001", res[0].PatientHN)
		assert.Equal(t, hospA.ID, res[0].HospitalID)
		assert.Equal(t, "1985-04-12", res[0].DateOfBirth.String())

		// Passport-only patient of hospital B is invisible to hospital A.
		res, err = patients.Search(ctx, hospA.ID, models.PatientSearchFilter{PassportID: "AA1234567"})
		require.NoError(t, err)
		assert.Empty(t, res)

		res, err = patients.Search(ctx, hospB.ID, models.PatientSearchFilter{PassportID: "AA1234567"})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Nil(t, res[0].FirstNameTH)
	})

	t.Run("patient search matches names in Thai and English", func(t *testing.T) {
		res, err := patients.Search(ctx, hospA.ID, models.PatientSearchFilter{FirstName: "SOM"})
		require.NoError(t, err)
		assert.Len(t, res, 2)

		res, err = patients.Search(ctx, hospA.ID, models.PatientSearchFilter{FirstName: "สมหญิง", LastName: "rak"})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "HN-A-0002", res[0].PatientHN)

		dob := models.NewDate(1990, time.September, 1)
		res, err = patients.Search(ctx, hospA.ID, models.PatientSearchFilter{DateOfBirth: &dob, Email: "SOMYING@example.com"})
		require.NoError(t, err)
		assert.Len(t, res, 1)

		res, err = patients.Search(ctx, hospA.ID, models.PatientSearchFilter{Limit: 1, Offset: 1})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "HN-A-0002", res[0].PatientHN)
	})

	t.Run("patient upsert inserts then updates", func(t *testing.T) {
		dob := models.NewDate(2000, time.January, 2)
		p := &models.Patient{
			HospitalID: hospA.ID, PatientHN: "HN-A-9999",
			FirstNameEN: ptr("Jane"), LastNameEN: ptr("Doe"),
			DateOfBirth: &dob, PassportID: ptr("P999"), Gender: ptr("F"),
		}
		require.NoError(t, patients.Upsert(ctx, p))
		firstID := p.ID
		assert.NotZero(t, firstID)

		p.PhoneNumber = ptr("0899999999")
		require.NoError(t, patients.Upsert(ctx, p))
		assert.Equal(t, firstID, p.ID)

		res, err := patients.Search(ctx, hospA.ID, models.PatientSearchFilter{PassportID: "P999"})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "0899999999", *res[0].PhoneNumber)
		assert.Nil(t, res[0].NationalID)

		// A second HN claiming the same passport in the same hospital is a duplicate.
		dup := &models.Patient{HospitalID: hospA.ID, PatientHN: "HN-A-8888", PassportID: ptr("P999")}
		assert.ErrorIs(t, patients.Upsert(ctx, dup), ErrDuplicate)
	})
}
