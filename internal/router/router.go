package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type AdminRouteRegistrar interface {
	Register(r chi.Router)
}

type Dependencies struct {
	AdminRoutes AdminRouteRegistrar
	Proxy       http.Handler
	Logger      *slog.Logger
	Middlewares []func(http.Handler) http.Handler
}

func New(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	for _, mw := range deps.Middlewares {
		r.Use(mw)
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/admin", deps.AdminRoutes.Register)
	r.NotFound(deps.Proxy.ServeHTTP)
	r.MethodNotAllowed(deps.Proxy.ServeHTTP)
	return r
}
