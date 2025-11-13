package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Deymos01/pr-review-manager/internal/config"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/teams/add"
	"github.com/Deymos01/pr-review-manager/internal/httpserver/handlers/teams/get"
	"github.com/Deymos01/pr-review-manager/internal/repository/postgres"
	"github.com/Deymos01/pr-review-manager/internal/usecase/team"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.Load()

	log := setupLogger(cfg.Env)

	log.Info("starting application", slog.String("env", cfg.Env))

	storage, err := postgres.New(cfg.PostgresConfig)
	if err != nil {
		slog.Error("failed to initialize storage",
			slog.String("env", cfg.Env),
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	teamService := team.New(log, storage)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/team", func(r chi.Router) {
		r.Post("/add", add.New(log, teamService))
		r.Get("/get", get.New(log, teamService))
	})

	router.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", nil)
		r.Get("/getReview", nil)
	})

	router.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", nil)
		r.Post("/merge", nil)
		r.Post("/reassign", nil)
	})

	addr := cfg.HTTPServerConfig.Host + ":" + strconv.Itoa(cfg.HTTPServerConfig.Port)

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTPServerConfig.Timeout,
		WriteTimeout:      cfg.HTTPServerConfig.Timeout,
		IdleTimeout:       cfg.HTTPServerConfig.IdleTimeout,
	}

	go func() {
		log.Info("starting server", slog.String("address", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	gracefulShutdown(context.Background(), srv, log)
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

func gracefulShutdown(ctx context.Context, srv *http.Server, log *slog.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", slog.Any("err", err))
		return
	}

	log.Info("server exited gracefully")
}
