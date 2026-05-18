package router_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gateway/internal/database"
	"gateway/internal/handlers"
	"gateway/internal/models"
	"gateway/internal/proxy"
	"gateway/internal/repositories"
	"gateway/internal/router"
	"gateway/internal/services"
)

func TestAdminRouteHotReloadAndProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/123" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("proxied"))
	}))
	defer upstream.Close()

	app := newTestApp(t)

	createBody := `{"path":"/users/*","target_url":"` + upstream.URL + `","methods":"GET","is_active":true}`
	createReq := httptest.NewRequest(http.MethodPost, "/admin/routes", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	app.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d: %s", http.StatusCreated, createRes.Code, createRes.Body.String())
	}

	proxyReq := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	proxyRes := httptest.NewRecorder()
	app.ServeHTTP(proxyRes, proxyReq)
	if proxyRes.Code != http.StatusAccepted {
		t.Fatalf("expected proxy status %d, got %d: %s", http.StatusAccepted, proxyRes.Code, proxyRes.Body.String())
	}
	if proxyRes.Body.String() != "proxied" {
		t.Fatalf("unexpected proxy response body: %s", proxyRes.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/admin/routes/1", bytes.NewBufferString(`{"is_active":false}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	app.ServeHTTP(updateRes, updateReq)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected update status %d, got %d: %s", http.StatusOK, updateRes.Code, updateRes.Body.String())
	}

	disabledReq := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	disabledRes := httptest.NewRecorder()
	app.ServeHTTP(disabledRes, disabledReq)
	if disabledRes.Code != http.StatusNotFound {
		t.Fatalf("expected disabled route status %d, got %d: %s", http.StatusNotFound, disabledRes.Code, disabledRes.Body.String())
	}
}

func newTestApp(t *testing.T) http.Handler {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "gateway-test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(db, &models.Route{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("database handle: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	routeRepo := repositories.NewGormRouteRepository(db)
	registry := proxy.NewRouteRegistry()
	routeService := services.NewRouteService(routeRepo, registry)
	if err := routeService.LoadActiveRoutes(context.Background()); err != nil {
		t.Fatalf("load active routes: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	adminHandler := handlers.NewRouteHandler(routeService, logger)
	proxyHandler := proxy.NewHandler(registry, logger)
	return router.New(router.Dependencies{
		AdminRoutes: adminHandler,
		Proxy:       proxyHandler,
		Logger:      logger,
	})
}
