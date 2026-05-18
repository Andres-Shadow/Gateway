package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gateway/internal/repositories"
	"gateway/internal/services"
)

type RouteHandler struct {
	service *services.RouteService
	logger  *slog.Logger
}

func NewRouteHandler(service *services.RouteService, logger *slog.Logger) *RouteHandler {
	return &RouteHandler{service: service, logger: logger}
}

func (h *RouteHandler) Register(r chi.Router) {
	r.Post("/routes", h.Create)
	r.Get("/routes", h.List)
	r.Get("/routes/{id}", h.Get)
	r.Put("/routes/{id}", h.Update)
	r.Delete("/routes/{id}", h.Delete)
}

func (h *RouteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input services.CreateRouteInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	route, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, route)
}

func (h *RouteHandler) List(w http.ResponseWriter, r *http.Request) {
	routes, err := h.service.List(r.Context())
	if err != nil {
		h.logger.Error("list routes failed", "error", err)
		writeError(w, http.StatusInternalServerError, "could not list routes")
		return
	}
	writeJSON(w, http.StatusOK, routes)
}

func (h *RouteHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := routeIDParam(w, r)
	if !ok {
		return
	}
	route, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, route)
}

func (h *RouteHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := routeIDParam(w, r)
	if !ok {
		return
	}

	var input services.UpdateRouteInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	route, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, route)
}

func (h *RouteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := routeIDParam(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RouteHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidRoute):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repositories.ErrRouteNotFound):
		writeError(w, http.StatusNotFound, "route not found")
	default:
		h.logger.Error("route request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "route request failed")
	}
}

func routeIDParam(w http.ResponseWriter, r *http.Request) (uint, bool) {
	id, err := services.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid route id")
		return 0, false
	}
	return id, true
}

func decodeJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
