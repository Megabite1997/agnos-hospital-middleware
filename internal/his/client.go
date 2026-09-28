// Package his is the client for hospital HIS (Hospital Information System)
// APIs, e.g. GET https://hospital-a.api.co.th/patient/search/{id}.
package his

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hospital-middleware/internal/models"
)

var ErrPatientNotFound = errors.New("patient not found in HIS")

// Patient is the HIS response body, field for field.
type Patient struct {
	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`
	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`
	DateOfBirth  string `json:"date_of_birth"`
	PatientHN    string `json:"patient_hn"`
	NationalID   string `json:"national_id"`
	PassportID   string `json:"passport_id"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
}

type Client interface {
	// SearchPatient looks up a patient by national ID or passport ID.
	SearchPatient(ctx context.Context, baseURL, id string) (*Patient, error)
}

type httpClient struct {
	http *http.Client
}

func NewClient(timeout time.Duration) Client {
	return &httpClient{http: &http.Client{Timeout: timeout}}
}

func (c *httpClient) SearchPatient(ctx context.Context, baseURL, id string) (*Patient, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/patient/search/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build HIS request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call HIS: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrPatientNotFound
	case resp.StatusCode != http.StatusOK:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HIS returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var p Patient
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&p); err != nil {
		return nil, fmt.Errorf("decode HIS response: %w", err)
	}
	if strings.TrimSpace(p.PatientHN) == "" {
		return nil, errors.New("HIS response is missing patient_hn")
	}
	return &p, nil
}

// ToModel converts the HIS payload into a Patient row for hospitalID.
// Blank strings become NULL; an unparsable date or gender is dropped rather
// than failing the whole record.
func (p Patient) ToModel(hospitalID int64) models.Patient {
	m := models.Patient{
		HospitalID:   hospitalID,
		PatientHN:    strings.TrimSpace(p.PatientHN),
		FirstNameTH:  optional(p.FirstNameTH),
		MiddleNameTH: optional(p.MiddleNameTH),
		LastNameTH:   optional(p.LastNameTH),
		FirstNameEN:  optional(p.FirstNameEN),
		MiddleNameEN: optional(p.MiddleNameEN),
		LastNameEN:   optional(p.LastNameEN),
		NationalID:   optional(p.NationalID),
		PassportID:   optional(p.PassportID),
		PhoneNumber:  optional(p.PhoneNumber),
		Email:        optional(p.Email),
	}
	if dob, err := models.ParseDate(p.DateOfBirth); err == nil {
		m.DateOfBirth = &dob
	}
	if g := strings.ToUpper(strings.TrimSpace(p.Gender)); g == "M" || g == "F" {
		m.Gender = &g
	}
	return m
}

func optional(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
