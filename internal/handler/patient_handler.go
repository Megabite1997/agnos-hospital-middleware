package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
	"hospital-middleware/internal/service"
)

type PatientHandler struct {
	patients service.PatientService
	logger   *slog.Logger
}

func NewPatientHandler(patients service.PatientService, logger *slog.Logger) *PatientHandler {
	return &PatientHandler{patients: patients, logger: logger}
}

// SearchPatientRequest is read from the query string on GET and from a JSON
// body on POST. Every field is optional.
type SearchPatientRequest struct {
	NationalID  string `form:"national_id" json:"national_id" binding:"max=20"`
	PassportID  string `form:"passport_id" json:"passport_id" binding:"max=20"`
	FirstName   string `form:"first_name" json:"first_name" binding:"max=100"`
	MiddleName  string `form:"middle_name" json:"middle_name" binding:"max=100"`
	LastName    string `form:"last_name" json:"last_name" binding:"max=100"`
	DateOfBirth string `form:"date_of_birth" json:"date_of_birth"`
	PhoneNumber string `form:"phone_number" json:"phone_number" binding:"max=20"`
	Email       string `form:"email" json:"email" binding:"omitempty,email,max=255"`
	Limit       int    `form:"limit" json:"limit" binding:"min=0,max=100"`
	Offset      int    `form:"offset" json:"offset" binding:"min=0"`
}

type SearchPatientResponse struct {
	Data   []models.Patient `json:"data"`
	Count  int              `json:"count"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// Search handles GET and POST /patient/search. The hospital comes from the
// caller's token, never from the request, so staff can only see patients of
// their own hospital.
func (h *PatientHandler) Search(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
		return
	}

	var req SearchPatientRequest
	if err := c.ShouldBind(&req); err != nil {
		respondBindError(c, err, &req)
		return
	}

	filter := models.PatientSearchFilter{
		NationalID:  req.NationalID,
		PassportID:  req.PassportID,
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
		Limit:       req.Limit,
		Offset:      req.Offset,
	}
	if dob := strings.TrimSpace(req.DateOfBirth); dob != "" {
		d, err := models.ParseDate(dob)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": errorBody{
				Code:    CodeValidation,
				Message: "request is invalid",
				Fields:  map[string]string{"date_of_birth": "must be a date in YYYY-MM-DD format"},
			}})
			return
		}
		filter.DateOfBirth = &d
	}
	if filter.Limit == 0 {
		filter.Limit = repository.DefaultSearchLimit
	}

	patients, err := h.patients.Search(c.Request.Context(), claims.HospitalID, filter)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "patient search failed", "error", err, "hospital_id", claims.HospitalID)
		respondError(c, http.StatusInternalServerError, CodeInternal, "internal server error")
		return
	}

	c.JSON(http.StatusOK, SearchPatientResponse{
		Data:   patients,
		Count:  len(patients),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}
