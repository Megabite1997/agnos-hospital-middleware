// Package router wires handlers and middleware into a Gin engine.
package router

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/handler"
	"hospital-middleware/internal/middleware"
)

// Pinger reports whether a dependency (the database) is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	Staff    *handler.StaffHandler
	Patients *handler.PatientHandler
	Tokens   auth.TokenManager
	DB       Pinger
}

func New(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	// Only trust X-Forwarded-For from the Nginx container network.
	_ = r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	r.GET("/health", func(c *gin.Context) {
		if err := d.DB.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	staff := r.Group("/staff")
	staff.POST("/create", d.Staff.Create)
	staff.POST("/login", d.Staff.Login)

	patient := r.Group("/patient", middleware.RequireAuth(d.Tokens))
	patient.GET("/search", d.Patients.Search)
	patient.POST("/search", d.Patients.Search)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "route not found"}})
	})
	return r
}
