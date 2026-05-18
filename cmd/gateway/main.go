package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway/internal/config"
	"gateway/internal/database"
	"gateway/internal/handlers"
	gatewaymiddleware "gateway/internal/middleware"
	"gateway/internal/models"
	"gateway/internal/proxy"
	"gateway/internal/repositories"
	"gateway/internal/router"
	"gateway/internal/services"
)

func main() {
	cfg := config.Load()
	logger := newLogger(cfg.Debug)

	db, err := database.Open(cfg.SQLitePath)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	if err := database.AutoMigrate(db, &models.Route{}); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	routeRepo := repositories.NewGormRouteRepository(db)
	registry := proxy.NewRouteRegistry()
	routeService := services.NewRouteService(routeRepo, registry)
	if err := routeService.LoadActiveRoutes(context.Background()); err != nil {
		logger.Error("route cache warmup failed", "error", err)
		os.Exit(1)
	}

	adminHandler := handlers.NewRouteHandler(routeService, logger)
	proxyHandler := proxy.NewHandler(registry, logger)
	httpRouter := router.New(router.Dependencies{
		AdminRoutes: adminHandler,
		Proxy:       proxyHandler,
		Logger:      logger,
		Middlewares: []func(http.Handler) http.Handler{
			gatewaymiddleware.RequestLogger(logger),
		},
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httpRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("gateway started", "addr", server.Addr, "sqlite_path", cfg.SQLitePath)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("gateway stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("gateway stopped")
}

func newLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
