package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

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
	mux.HandleFunc("/api/auth/test-smtp", handleTestSMTP)
	mux.HandleFunc("/api/auth/test-gemini", handleTestGemini)
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

// GET /api/auth/test-smtp
func handleTestSMTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, domain.ErrorResponse{Error: "method not allowed"})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		writeJSON(w, http.StatusBadRequest, domain.ErrorResponse{Error: "email query parameter is required"})
		return
	}

	err := service.SendVerificationEmail(email, "Test User", "test-token-12345")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Test email sent successfully to " + email,
	})
}

// GET /api/auth/test-gemini
func handleTestGemini(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, domain.ErrorResponse{Error: "method not allowed"})
		return
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "failed",
			"error":  "GEMINI_API_KEY environment variable is empty",
		})
		return
	}

	// Simple request payload
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": "Hello, respond in one word: OK"},
				},
			},
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("marshal error: %v", err),
		})
		return
	}

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", apiKey)
	
	var resp *http.Response
	var lastErr error
	maxRetries := 3
	backoff := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		bodyReader := bytes.NewReader(jsonData)
		httpReq, err := http.NewRequest("POST", apiURL, bodyReader)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"status": "failed",
				"error":  fmt.Sprintf("new request error on attempt %d: %v", attempt, err),
			})
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 60 * time.Second}
		resp, lastErr = client.Do(httpReq)
		if lastErr != nil {
			log.Printf("⚠️ [test-gemini] Gemini API call attempt %d failed: %v", attempt, lastErr)
			if attempt < maxRetries {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			break
		}

		if resp.StatusCode == http.StatusOK {
			break
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			log.Printf("⚠️ [test-gemini] Gemini API returned status %d on attempt %d: %s", resp.StatusCode, attempt, string(bodyBytes))
			lastErr = fmt.Errorf("Gemini API returned status %d", resp.StatusCode)
			if attempt < maxRetries {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			resp = nil
			break
		}

		break
	}

	if lastErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("client Do error after %d attempts: %v", maxRetries, lastErr),
		})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	
	// Set status code to match the API response status code for debugging
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	
	writeJSON(w, resp.StatusCode, map[string]interface{}{
		"status":      "response_received",
		"status_code": resp.StatusCode,
		"body":        string(bodyBytes),
	})
}
