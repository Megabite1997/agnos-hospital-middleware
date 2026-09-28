package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/service"
)

func init() { gin.SetMode(gin.TestMode) }

var (
	discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
	errBoom       = errors.New("boom")
	testHospital  = &models.Hospital{ID: 1, Code: "hospital-a", Name: "Hospital A"}
)

// ---- fakes -----------------------------------------------------------------

type fakeStaffService struct {
	createErr error
	loginErr  error
	lastInput service.CreateStaffInput
}

func (f *fakeStaffService) Create(_ context.Context, in service.CreateStaffInput) (*models.Staff, *models.Hospital, error) {
	f.lastInput = in
	if f.createErr != nil {
		return nil, nil, f.createErr
	}
	return &models.Staff{ID: 10, HospitalID: 1, Username: in.Username}, testHospital, nil
}

func (f *fakeStaffService) Login(_ context.Context, in service.LoginInput) (*service.LoginResult, error) {
	f.lastInput = in
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return &service.LoginResult{
		AccessToken: "jwt-token",
		ExpiresAt:   time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Staff:       &models.Staff{ID: 10, HospitalID: 1, Username: in.Username},
		Hospital:    testHospital,
	}, nil
}

type fakePatientService struct {
	result     []models.Patient
	err        error
	hospitalID int64
	filter     models.PatientSearchFilter
	called     bool
}

func (f *fakePatientService) Search(_ context.Context, hospitalID int64, filter models.PatientSearchFilter) ([]models.Patient, error) {
	f.called, f.hospitalID, f.filter = true, hospitalID, filter
	return f.result, f.err
}

// ---- helpers ---------------------------------------------------------------

var tokens = auth.NewTokenManager(strings.Repeat("k", 32), time.Hour)

func newTestRouter(staff service.StaffService, patients service.PatientService) *gin.Engine {
	r := gin.New()
	sh := NewStaffHandler(staff, discardLogger)
	ph := NewPatientHandler(patients, discardLogger)
	r.POST("/staff/create", sh.Create)
	r.POST("/staff/login", sh.Login)
	authed := r.Group("/", middleware.RequireAuth(tokens))
	authed.GET("/patient/search", ph.Search)
	authed.POST("/patient/search", ph.Search)
	return r
}

func doJSON(t *testing.T, r http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type errResp struct {
	Error errorBody `json:"error"`
}

func decodeErr(t *testing.T, w *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var e errResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &e), w.Body.String())
	return e.Error
}

func staffToken(t *testing.T, hospitalID int64) string {
	t.Helper()
	tok, _, err := tokens.Generate(10, "alice", hospitalID, "hospital-a")
	require.NoError(t, err)
	return tok
}

// ---- POST /staff/create ------------------------------------------------------

func TestCreateStaff_Success(t *testing.T) {
	svc := &fakeStaffService{}
	w := doJSON(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/create",
		`{"username":"alice","password":"password123","hospital":"hospital-a"}`, "")

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var resp StaffResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(10), resp.ID)
	assert.Equal(t, "alice", resp.Username)
	assert.Equal(t, "hospital-a", resp.Hospital.Code)
	assert.NotContains(t, w.Body.String(), "password")
	assert.Equal(t, service.CreateStaffInput{Username: "alice", Password: "password123", Hospital: "hospital-a"}, svc.lastInput)
}

func TestCreateStaff_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
		wantMsg   string
	}{
		{"missing username", `{"password":"password123","hospital":"hospital-a"}`, "username", "is required"},
		{"short username", `{"username":"al","password":"password123","hospital":"hospital-a"}`, "username", "at least 3"},
		{"short password", `{"username":"alice","password":"short","hospital":"hospital-a"}`, "password", "at least 8"},
		{"long password", `{"username":"alice","password":"` + strings.Repeat("p", 73) + `","hospital":"hospital-a"}`, "password", "at most 72"},
		{"missing hospital", `{"username":"alice","password":"password123"}`, "hospital", "is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeStaffService{}
			w := doJSON(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/create", tc.body, "")
			require.Equal(t, http.StatusBadRequest, w.Code)
			e := decodeErr(t, w)
			assert.Equal(t, CodeValidation, e.Code)
			assert.Contains(t, e.Fields[tc.wantField], tc.wantMsg)
			assert.Empty(t, svc.lastInput.Username, "service must not be called")
		})
	}
}

func TestCreateStaff_MalformedJSON(t *testing.T) {
	w := doJSON(t, newTestRouter(&fakeStaffService{}, &fakePatientService{}), http.MethodPost, "/staff/create", `{"username":`, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "request body is malformed", decodeErr(t, w).Message)
}

func TestCreateStaff_ServiceErrors(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{service.ErrHospitalNotFound, http.StatusNotFound, CodeHospitalNotFound},
		{service.ErrUsernameTaken, http.StatusConflict, CodeUsernameTaken},
		{errBoom, http.StatusInternalServerError, CodeInternal},
	}
	for _, tc := range tests {
		t.Run(tc.wantCode, func(t *testing.T) {
			r := newTestRouter(&fakeStaffService{createErr: tc.err}, &fakePatientService{})
			w := doJSON(t, r, http.MethodPost, "/staff/create", `{"username":"alice","password":"password123","hospital":"x"}`, "")
			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Equal(t, tc.wantCode, decodeErr(t, w).Code)
			assert.NotContains(t, w.Body.String(), "boom", "internal errors must not leak")
		})
	}
}

// ---- POST /staff/login -------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	w := doJSON(t, newTestRouter(&fakeStaffService{}, &fakePatientService{}), http.MethodPost, "/staff/login",
		`{"username":"alice","password":"password123","hospital":"hospital-a"}`, "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "jwt-token", resp.AccessToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, "hospital-a", resp.Staff.Hospital.Code)
}

func TestLogin_Errors(t *testing.T) {
	tests := []struct {
		name       string
		svcErr     error
		body       string
		wantStatus int
		wantCode   string
	}{
		{"missing fields", nil, `{}`, http.StatusBadRequest, CodeValidation},
		{"invalid credentials", service.ErrInvalidCredentials, `{"username":"a","password":"b","hospital":"c"}`, http.StatusUnauthorized, CodeInvalidCredentials},
		{"internal error", errBoom, `{"username":"a","password":"b","hospital":"c"}`, http.StatusInternalServerError, CodeInternal},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRouter(&fakeStaffService{loginErr: tc.svcErr}, &fakePatientService{})
			w := doJSON(t, r, http.MethodPost, "/staff/login", tc.body, "")
			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Equal(t, tc.wantCode, decodeErr(t, w).Code)
		})
	}
}

// ---- GET/POST /patient/search -----------------------------------------------

func TestSearchPatient_RequiresLogin(t *testing.T) {
	svc := &fakePatientService{}
	w := doJSON(t, newTestRouter(&fakeStaffService{}, svc), http.MethodGet, "/patient/search", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, svc.called)

	w = doJSON(t, newTestRouter(&fakeStaffService{}, svc), http.MethodGet, "/patient/search", "", "garbage")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, svc.called)
}

func TestSearchPatient_GET_UsesHospitalFromToken(t *testing.T) {
	hn := "HN-A-0001"
	svc := &fakePatientService{result: []models.Patient{{ID: 1, HospitalID: 7, PatientHN: hn}}}
	r := newTestRouter(&fakeStaffService{}, svc)

	w := doJSON(t, r, http.MethodGet,
		"/patient/search?national_id=1103700000011&first_name=som&date_of_birth=1985-04-12&email=a@b.co&limit=5&offset=10&hospital_id=999",
		"", staffToken(t, 7))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, int64(7), svc.hospitalID, "hospital must come from the token, not the query")
	assert.Equal(t, "1103700000011", svc.filter.NationalID)
	assert.Equal(t, "som", svc.filter.FirstName)
	assert.Equal(t, "a@b.co", svc.filter.Email)
	assert.Equal(t, "1985-04-12", svc.filter.DateOfBirth.String())
	assert.Equal(t, 5, svc.filter.Limit)
	assert.Equal(t, 10, svc.filter.Offset)

	var resp SearchPatientResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Count)
	assert.Equal(t, hn, resp.Data[0].PatientHN)
}

func TestSearchPatient_POST_JSONBody(t *testing.T) {
	svc := &fakePatientService{result: []models.Patient{}}
	w := doJSON(t, newTestRouter(&fakeStaffService{}, svc), http.MethodPost, "/patient/search",
		`{"passport_id":"AA1234567","last_name":"smith"}`, staffToken(t, 2))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, int64(2), svc.hospitalID)
	assert.Equal(t, "AA1234567", svc.filter.PassportID)
	assert.Equal(t, "smith", svc.filter.LastName)
	assert.JSONEq(t, `{"data":[],"count":0,"limit":20,"offset":0}`, w.Body.String())
}

func TestSearchPatient_NoFiltersUsesDefaults(t *testing.T) {
	svc := &fakePatientService{result: []models.Patient{}}
	w := doJSON(t, newTestRouter(&fakeStaffService{}, svc), http.MethodGet, "/patient/search", "", staffToken(t, 1))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, models.PatientSearchFilter{Limit: 20}, svc.filter)
}

func TestSearchPatient_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantField string
	}{
		{"bad date", "date_of_birth=12-04-1985", "date_of_birth"},
		{"bad email", "email=not-an-email", "email"},
		{"limit too large", "limit=101", "limit"},
		{"negative offset", "offset=-1", "offset"},
		{"national id too long", "national_id=" + strings.Repeat("1", 21), "national_id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakePatientService{}
			w := doJSON(t, newTestRouter(&fakeStaffService{}, svc), http.MethodGet, "/patient/search?"+tc.query, "", staffToken(t, 1))
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			e := decodeErr(t, w)
			assert.Equal(t, CodeValidation, e.Code)
			assert.Contains(t, e.Fields, tc.wantField)
			assert.False(t, svc.called)
		})
	}
}

func TestSearchPatient_NonNumericLimit(t *testing.T) {
	w := doJSON(t, newTestRouter(&fakeStaffService{}, &fakePatientService{}), http.MethodGet, "/patient/search?limit=abc", "", staffToken(t, 1))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchPatient_ServiceError(t *testing.T) {
	w := doJSON(t, newTestRouter(&fakeStaffService{}, &fakePatientService{err: errBoom}), http.MethodGet, "/patient/search", "", staffToken(t, 1))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, CodeInternal, decodeErr(t, w).Code)
}

func TestSearchPatient_WithoutAuthMiddleware(t *testing.T) {
	// Defensive check: if the route were ever mounted without RequireAuth.
	r := gin.New()
	r.GET("/patient/search", NewPatientHandler(&fakePatientService{}, discardLogger).Search)
	w := doJSON(t, r, http.MethodGet, "/patient/search", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJSONFieldName_FallsBackToStructName(t *testing.T) {
	type noTag struct{ Foo string }
	assert.Equal(t, "Foo", jsonFieldName(noTag{}, "Foo"))
	assert.Equal(t, "username", jsonFieldName(&CreateStaffRequest{}, "Username"))
}
