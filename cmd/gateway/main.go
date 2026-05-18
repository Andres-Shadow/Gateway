package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway/internal/auth"
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
	breaker := proxy.NewCircuitBreaker(cfg.Proxy.CircuitFailureLimit, cfg.Proxy.CircuitCooldownPeriod)
	routeService := services.NewRouteService(routeRepo, registry)
	if err := routeService.LoadActiveRoutes(context.Background()); err != nil {
		logger.Error("route cache warmup failed", "error", err)
		os.Exit(1)
	}

	authService := auth.NewService(cfg.AdminAuth.Username, cfg.AdminAuth.Password, cfg.AdminAuth.JWTSecret, cfg.AdminAuth.TokenTTL)
	authHandler := handlers.NewAuthHandler(authService, logger)
	adminHandler := handlers.NewRouteHandler(routeService, logger)
	proxyHandler := proxy.NewHandler(registry, breaker, logger, proxy.HandlerOptions{
		MaxRetries:   cfg.Proxy.MaxRetries,
		RetryBackoff: cfg.Proxy.RetryBackoff,
	})
	rootMiddlewares := []func(http.Handler) http.Handler{
		gatewaymiddleware.RequestLogger(logger),
		gatewaymiddleware.CORS(gatewaymiddleware.CORSOptions{
			AllowedOrigins: cfg.CORS.AllowedOrigins,
			AllowedMethods: cfg.CORS.AllowedMethods,
			AllowedHeaders: cfg.CORS.AllowedHeaders,
		}),
	}
	if cfg.RateLimit.Enabled {
		rootMiddlewares = append(rootMiddlewares, gatewaymiddleware.NewRateLimiter(cfg.RateLimit.RequestsPerMinute, time.Minute).Middleware())
	}
	httpRouter := router.New(router.Dependencies{
		AdminRoutes:      adminHandler,
		AuthTokenHandler: authHandler.Token,
		Proxy:            proxyHandler,
		Logger:           logger,
		Middlewares:      rootMiddlewares,
		AdminMiddlewares: []func(http.Handler) http.Handler{
			gatewaymiddleware.AdminJWT(authService),
		},
	})

	appCtx, stopApp := context.WithCancel(context.Background())
	defer stopApp()
	if cfg.HealthCheck.Enabled {
		healthChecker := proxy.NewHealthChecker(registry, cfg.HealthCheck.Timeout, cfg.HealthCheck.Interval, logger)
		go healthChecker.Start(appCtx)
	}

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
	stopApp()

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
