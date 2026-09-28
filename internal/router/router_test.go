package router

import (
	"context"
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

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/handler"
)

func init() { gin.SetMode(gin.TestMode) }

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func newEngine(dbErr error) *gin.Engine {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(Deps{
		Staff:    handler.NewStaffHandler(nil, logger),
		Patients: handler.NewPatientHandler(nil, logger),
		Tokens:   auth.NewTokenManager(strings.Repeat("k", 32), time.Hour),
		DB:       fakePinger{err: dbErr},
	})
}

func serve(r http.Handler, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestHealth(t *testing.T) {
	assert.Equal(t, http.StatusOK, serve(newEngine(nil), http.MethodGet, "/health").Code)
	assert.Equal(t, http.StatusServiceUnavailable, serve(newEngine(errors.New("down")), http.MethodGet, "/health").Code)
}

func TestPatientRoutesAreProtected(t *testing.T) {
	r := newEngine(nil)
	assert.Equal(t, http.StatusUnauthorized, serve(r, http.MethodGet, "/patient/search").Code)
	assert.Equal(t, http.StatusUnauthorized, serve(r, http.MethodPost, "/patient/search").Code)
}

func TestNoRoute(t *testing.T) {
	w := serve(newEngine(nil), http.MethodGet, "/nope")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "NOT_FOUND")
}
