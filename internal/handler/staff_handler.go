package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/service"
)

type StaffHandler struct {
	staff  service.StaffService
	logger *slog.Logger
}

func NewStaffHandler(staff service.StaffService, logger *slog.Logger) *StaffHandler {
	return &StaffHandler{staff: staff, logger: logger}
}

type CreateStaffRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	// bcrypt only uses the first 72 bytes, so longer passwords are rejected.
	Password string `json:"password" binding:"required,min=8,max=72"`
	Hospital string `json:"hospital" binding:"required,max=50"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,max=50"`
	Password string `json:"password" binding:"required,max=72"`
	Hospital string `json:"hospital" binding:"required,max=50"`
}

type HospitalResponse struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type StaffResponse struct {
	ID        int64            `json:"id"`
	Username  string           `json:"username"`
	Hospital  HospitalResponse `json:"hospital"`
	CreatedAt time.Time        `json:"created_at"`
}

type LoginResponse struct {
	AccessToken string        `json:"access_token"`
	TokenType   string        `json:"token_type"`
	ExpiresAt   time.Time     `json:"expires_at"`
	Staff       StaffResponse `json:"staff"`
}

// Create handles POST /staff/create.
func (h *StaffHandler) Create(c *gin.Context) {
	var req CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err, &req)
		return
	}

	staff, hospital, err := h.staff.Create(c.Request.Context(), service.CreateStaffInput{
		Username: req.Username, Password: req.Password, Hospital: req.Hospital,
	})
	switch {
	case errors.Is(err, service.ErrHospitalNotFound):
		respondError(c, http.StatusNotFound, CodeHospitalNotFound, "hospital not found")
		return
	case errors.Is(err, service.ErrUsernameTaken):
		respondError(c, http.StatusConflict, CodeUsernameTaken, "username already exists in this hospital")
		return
	case err != nil:
		h.logger.ErrorContext(c.Request.Context(), "create staff failed", "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "internal server error")
		return
	}

	c.JSON(http.StatusCreated, StaffResponse{
		ID:        staff.ID,
		Username:  staff.Username,
		Hospital:  HospitalResponse{ID: hospital.ID, Code: hospital.Code, Name: hospital.Name},
		CreatedAt: staff.CreatedAt,
	})
}

// Login handles POST /staff/login.
func (h *StaffHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err, &req)
		return
	}

	res, err := h.staff.Login(c.Request.Context(), service.LoginInput{
		Username: req.Username, Password: req.Password, Hospital: req.Hospital,
	})
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, CodeInvalidCredentials, "invalid username, password or hospital")
		return
	case err != nil:
		h.logger.ErrorContext(c.Request.Context(), "login failed", "error", err)
		respondError(c, http.StatusInternalServerError, CodeInternal, "internal server error")
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken: res.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   res.ExpiresAt,
		Staff: StaffResponse{
			ID:        res.Staff.ID,
			Username:  res.Staff.Username,
			Hospital:  HospitalResponse{ID: res.Hospital.ID, Code: res.Hospital.Code, Name: res.Hospital.Name},
			CreatedAt: res.Staff.CreatedAt,
		},
	})
}
