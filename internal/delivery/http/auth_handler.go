package http

import (
	"encoding/json"
	"net/http"

	"github.com/your-org/ai-employee-platform/internal/domain"
	"github.com/your-org/ai-employee-platform/internal/repository"
	"github.com/your-org/ai-employee-platform/internal/service"
)

// RegisterAuthRoutes registers all auth-related HTTP handlers on the mux.
func RegisterAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/register", handleRegister)
	mux.HandleFunc("/api/auth/login", handleLogin)
	mux.HandleFunc("/api/auth/google", handleGoogleLogin)
	mux.HandleFunc("/api/auth/verify-email", handleVerifyEmail)
	mux.HandleFunc("/api/auth/resend-verification", handleResendVerification)
	mux.Handle("/api/auth/me", AuthMiddleware(http.HandlerFunc(handleMe)))
}

// POST /api/auth/register
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, domain.ErrorResponse{Error: "method not allowed"})
		return
	}

	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: "invalid request body"})
		return
	}

	resp, err := service.Register(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// POST /api/auth/login
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, domain.ErrorResponse{Error: "method not allowed"})
		return
	}

	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: "invalid request body"})
		return
	}

	resp, err := service.Login(req)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, domain.ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// POST /api/auth/google
func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, domain.ErrorResponse{Error: "method not allowed"})
		return
	}

	var req domain.GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: "invalid request body"})
		return
	}

	resp, err := service.LoginWithGoogle(req.IDToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, domain.ErrorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GET /api/auth/me  (protected)
func handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, domain.ErrorResponse{Error: "method not allowed"})
		return
	}

	userID := GetUserIDFromContext(r)
	user, err := repository.GetUserByID(userID)
	if err != nil || user == nil {
		writeJSON(w, http.StatusUnauthorized, domain.ErrorResponse{Error: "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// writeJSON is a helper that sets Content-Type and encodes the response.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// POST /api/auth/verify-email
func handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, domain.ErrorResponse{Error: "method not allowed"})
		return
	}

	var req domain.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := service.VerifyEmail(req.Token); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Email verified successfully"})
}

// POST /api/auth/resend-verification
func handleResendVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, domain.ErrorResponse{Error: "method not allowed"})
		return
	}

	var req domain.ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := service.ResendVerification(req.Email); err != nil {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Verification link sent successfully"})
}
