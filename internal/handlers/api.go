// Package handlers provides HTTP handlers for the loyalty system API.
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/max-marek-projects/custodia/internal/auth"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/models"
	"github.com/max-marek-projects/custodia/internal/service"
)

// RegisterUser handles user registration.
// Reads login/password from JSON, creates a user, and sets an auth cookie.
func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {

	var requestData models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", slog.Any("error", err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if requestData.Login == "" || requestData.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}
	userID, err := h.service.RegisterUser(r.Context(), requestData)
	if err != nil {
		if errors.Is(err, service.ErrLoginAlreadyTaken) {
			http.Error(w, "Login already taken", http.StatusConflict)
			return
		}
		logger.Log.Error("Failed to register user", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := auth.SetUserCookie(w, userID, h.secretKey); err != nil {
		logger.Log.Error("Failed to set auth cookie", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// LoginUser handles user login.
// Validates credentials and sets an auth cookie on success.
func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var requestData models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", slog.Any("error", err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if requestData.Login == "" || requestData.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}
	userID, err := h.service.LoginUser(r.Context(), requestData)
	if err != nil {
		if errors.Is(err, service.ErrWrongUsernamePassword) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		logger.Log.Error("Failed to register user", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := auth.SetUserCookie(w, userID, h.secretKey); err != nil {
		logger.Log.Error("Failed to set auth cookie", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
