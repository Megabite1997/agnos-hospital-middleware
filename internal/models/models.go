// Package models holds the domain types shared by every layer.
package models

import "time"

type Hospital struct {
	ID         int64     `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	HISBaseURL *string   `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Staff struct {
	ID           int64     `json:"id"`
	HospitalID   int64     `json:"hospital_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Patient mirrors the HIS patient payload plus the hospital it belongs to.
// Optional fields are pointers so "absent" is stored as NULL, not "".
type Patient struct {
	ID           int64     `json:"id"`
	HospitalID   int64     `json:"hospital_id"`
	PatientHN    string    `json:"patient_hn"`
	FirstNameTH  *string   `json:"first_name_th"`
	MiddleNameTH *string   `json:"middle_name_th"`
	LastNameTH   *string   `json:"last_name_th"`
	FirstNameEN  *string   `json:"first_name_en"`
	MiddleNameEN *string   `json:"middle_name_en"`
	LastNameEN   *string   `json:"last_name_en"`
	DateOfBirth  *Date     `json:"date_of_birth"`
	NationalID   *string   `json:"national_id"`
	PassportID   *string   `json:"passport_id"`
	PhoneNumber  *string   `json:"phone_number"`
	Email        *string   `json:"email"`
	Gender       *string   `json:"gender"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PatientSearchFilter holds the optional search criteria. Empty fields are
// ignored; all non-empty fields must match (AND).
type PatientSearchFilter struct {
	NationalID  string
	PassportID  string
	FirstName   string
	MiddleName  string
	LastName    string
	DateOfBirth *Date
	PhoneNumber string
	Email       string
	Limit       int
	Offset      int
}
