// Command server runs the Hospital Middleware HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/handler"
	"hospital-middleware/internal/his"
	"hospital-middleware/internal/repository"
	"hospital-middleware/internal/router"
	"hospital-middleware/internal/service"
	"hospital-middleware/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	gin.SetMode(cfg.GinMode)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		return err
	}

	hospitals := repository.NewHospitalRepository(pool)
	staffRepo := repository.NewStaffRepository(pool)
	patients := repository.NewPatientRepository(pool)
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)

	staffSvc := service.NewStaffService(hospitals, staffRepo, auth.NewBcryptHasher(), tokens)
	patientSvc := service.NewPatientService(hospitals, patients, his.NewClient(cfg.HISTimeout), logger)

	engine := router.New(router.Deps{
		Staff:    handler.NewStaffHandler(staffSvc, logger),
		Patients: handler.NewPatientHandler(patientSvc, logger),
		Tokens:   tokens,
		DB:       pool,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
