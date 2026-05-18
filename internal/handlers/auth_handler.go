package handlers

import (
	"log/slog"
	"net/http"

	"gateway/internal/auth"
)

type AuthHandler struct {
	service *auth.Service
	logger  *slog.Logger
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(service *auth.Service, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{service: service, logger: logger}
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	var input LoginRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !h.service.Authenticate(input.Username, input.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := h.service.IssueToken(input.Username)
	if err != nil {
		h.logger.Error("token issue failed", "error", err)
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
