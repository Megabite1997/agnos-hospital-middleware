package repository

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	"hospital-middleware/internal/models"
)

func whereClause(sql string) string {
	start := strings.Index(sql, "WHERE ")
	end := strings.Index(sql, " ORDER BY")
	return sql[start+len("WHERE ") : end]
}

func TestBuildSearchQuery_NoFilters_ScopesToHospitalWithDefaultPaging(t *testing.T) {
	sql, args := buildSearchQuery(7, models.PatientSearchFilter{})

	assert.Equal(t, "hospital_id = $1", whereClause(sql))
	assert.Contains(t, sql, "ORDER BY id LIMIT $2 OFFSET $3")
	assert.Equal(t, []any{int64(7), DefaultSearchLimit, 0}, args)
}

func TestBuildSearchQuery_AllFilters(t *testing.T) {
	dob := models.NewDate(1990, time.September, 1)
	sql, args := buildSearchQuery(1, models.PatientSearchFilter{
		NationalID:  " 1103700000022 ",
		PassportID:  "AA1234567",
		FirstName:   "som",
		MiddleName:  "k",
		LastName:    "rak",
		DateOfBirth: &dob,
		PhoneNumber: "0822222222",
		Email:       "Somying@Example.com",
		Limit:       10,
		Offset:      5,
	})

	assert.Equal(t, strings.Join([]string{
		"hospital_id = $1",
		"national_id = $2",
		"passport_id = $3",
		"(first_name_th ILIKE $4 OR first_name_en ILIKE $4)",
		"(middle_name_th ILIKE $5 OR middle_name_en ILIKE $5)",
		"(last_name_th ILIKE $6 OR last_name_en ILIKE $6)",
		"date_of_birth = $7",
		"phone_number = $8",
		"LOWER(email) = LOWER($9)",
	}, " AND "), whereClause(sql))
	assert.Contains(t, sql, "LIMIT $10 OFFSET $11")
	assert.Equal(t, []any{
		int64(1), "1103700000022", "AA1234567", "%som%", "%k%", "%rak%",
		dob, "0822222222", "Somying@Example.com", 10, 5,
	}, args)
}

func TestBuildSearchQuery_BlankStringsIgnored(t *testing.T) {
	sql, args := buildSearchQuery(1, models.PatientSearchFilter{FirstName: "   ", Email: ""})
	assert.Equal(t, "hospital_id = $1", whereClause(sql))
	assert.Len(t, args, 3)
}

func TestBuildSearchQuery_Paging(t *testing.T) {
	tests := []struct {
		limit, offset         int
		wantLimit, wantOffset int
	}{
		{0, 0, DefaultSearchLimit, 0},
		{-5, -1, DefaultSearchLimit, 0},
		{50, 10, 50, 10},
		{1000, 0, MaxSearchLimit, 0},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("limit=%d offset=%d", tc.limit, tc.offset), func(t *testing.T) {
			_, args := buildSearchQuery(1, models.PatientSearchFilter{Limit: tc.limit, Offset: tc.offset})
			assert.Equal(t, tc.wantLimit, args[len(args)-2])
			assert.Equal(t, tc.wantOffset, args[len(args)-1])
		})
	}
}

func TestContainsPattern_EscapesWildcards(t *testing.T) {
	assert.Equal(t, `%50\%\_off\\%`, containsPattern(`50%_off\`))
}

func TestMapError(t *testing.T) {
	assert.ErrorIs(t, mapError(pgx.ErrNoRows), ErrNotFound)
	assert.ErrorIs(t, mapError(&pgconn.PgError{Code: "23505"}), ErrDuplicate)

	other := errors.New("boom")
	assert.Equal(t, other, mapError(other))
	assert.NoError(t, mapError(nil))
}
