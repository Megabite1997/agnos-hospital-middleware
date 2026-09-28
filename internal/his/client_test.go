package his

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hospital-middleware/internal/models"
)

func TestSearchPatient_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/patient/search/AB 123", r.URL.Path)
		assert.Equal(t, "/patient/search/AB%20123", r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"first_name_en":"Jane","patient_hn":"HN1","passport_id":"AB 123","gender":"F","date_of_birth":"1990-01-02"}`))
	}))
	defer srv.Close()

	p, err := NewClient(time.Second).SearchPatient(context.Background(), srv.URL+"/", "AB 123")
	require.NoError(t, err)
	assert.Equal(t, "Jane", p.FirstNameEN)
	assert.Equal(t, "HN1", p.PatientHN)
}

func TestSearchPatient_Errors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
		wantMsg string
	}{
		{"not found", http.StatusNotFound, `{}`, ErrPatientNotFound, ""},
		{"server error", http.StatusInternalServerError, "boom", nil, "status 500: boom"},
		{"invalid json", http.StatusOK, "{", nil, "decode HIS response"},
		{"missing hn", http.StatusOK, `{"first_name_en":"x"}`, nil, "missing patient_hn"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			_, err := NewClient(time.Second).SearchPatient(context.Background(), srv.URL, "1")
			require.Error(t, err)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.ErrorContains(t, err, tc.wantMsg)
			}
		})
	}
}

func TestSearchPatient_Unreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	_, err := NewClient(time.Second).SearchPatient(context.Background(), url, "1")
	assert.ErrorContains(t, err, "call HIS")
}

func TestSearchPatient_BadURL(t *testing.T) {
	_, err := NewClient(time.Second).SearchPatient(context.Background(), "://bad", "1")
	assert.ErrorContains(t, err, "build HIS request")
}

func TestSearchPatient_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(time.Second):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	_, err := NewClient(50*time.Millisecond).SearchPatient(context.Background(), srv.URL, "1")
	assert.Error(t, err)
}

func TestToModel(t *testing.T) {
	p := Patient{
		FirstNameTH: "สมชาย", LastNameTH: "ใจดี", FirstNameEN: " Somchai ", LastNameEN: "Jaidee",
		MiddleNameEN: "  ", DateOfBirth: "1985-04-12", PatientHN: " HN-1 ",
		NationalID: "1103700000011", PhoneNumber: "0811111111", Email: "a@b.c", Gender: "m",
	}
	m := p.ToModel(3)

	assert.Equal(t, int64(3), m.HospitalID)
	assert.Equal(t, "HN-1", m.PatientHN)
	assert.Equal(t, "Somchai", *m.FirstNameEN)
	assert.Nil(t, m.MiddleNameEN)
	assert.Nil(t, m.PassportID)
	assert.Equal(t, models.NewDate(1985, time.April, 12), *m.DateOfBirth)
	assert.Equal(t, "M", *m.Gender)
}

func TestToModel_DropsInvalidDateAndGender(t *testing.T) {
	m := Patient{PatientHN: "HN", DateOfBirth: "12/04/1985", Gender: "X"}.ToModel(1)
	assert.Nil(t, m.DateOfBirth)
	assert.Nil(t, m.Gender)
}
