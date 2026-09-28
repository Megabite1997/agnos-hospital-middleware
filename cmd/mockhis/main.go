// Command mockhis is a stand-in for Hospital A's HIS API
// (GET https://hospital-a.api.co.th/patient/search/{id}) so the whole flow
// can be demonstrated locally with docker compose.
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

type patient struct {
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

var patients = []patient{
	{
		FirstNameTH: "สมชาย", LastNameTH: "ใจดี", FirstNameEN: "Somchai", LastNameEN: "Jaidee",
		DateOfBirth: "1985-04-12", PatientHN: "HN-A-0001", NationalID: "1103700000011",
		PhoneNumber: "0811111111", Email: "somchai@example.com", Gender: "M",
	},
	{
		// Only exists in the HIS: the middleware learns about this patient on first search.
		FirstNameTH: "มานี", MiddleNameTH: "", LastNameTH: "มีนา", FirstNameEN: "Manee", LastNameEN: "Meena",
		DateOfBirth: "1995-12-25", PatientHN: "HN-A-0003", NationalID: "1103700000033",
		PhoneNumber: "0844444444", Email: "manee@example.com", Gender: "F",
	},
	{
		FirstNameEN: "Emma", MiddleNameEN: "Rose", LastNameEN: "Brown",
		DateOfBirth: "1992-07-15", PatientHN: "HN-A-0004", PassportID: "GB9876543",
		PhoneNumber: "0855555555", Email: "emma@example.com", Gender: "F",
	},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /patient/search/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		w.Header().Set("Content-Type", "application/json")
		for _, p := range patients {
			if id != "" && (p.NationalID == id || p.PassportID == id) {
				_ = json.NewEncoder(w).Encode(p)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"patient not found"}`))
	})

	addr := ":" + envOr("PORT", "8081")
	slog.Info("mock HIS listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("mock HIS stopped", "error", err)
		os.Exit(1)
	}
}

func envOr(k, v string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return v
}
