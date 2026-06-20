package domain

import "time"

// User represents a registered user.
type User struct {
	ID                int64     `json:"id"`
	Email             string    `json:"email"`
	Name              string    `json:"name"`
	PasswordHash      string    `json:"-"` // never expose
	AvatarURL         string    `json:"avatar_url"`
	Provider          string    `json:"provider"`     // "email" or "google"
	ProviderID        string    `json:"provider_id"`  // Google sub ID
	IsVerified        bool      `json:"is_verified"`
	VerificationToken string    `json:"-"` // never expose
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// --- Request / Response DTOs ---

// RegisterRequest is the body for POST /api/auth/register.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the body for POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// GoogleLoginRequest carries the Google ID-token from the frontend.
type GoogleLoginRequest struct {
	IDToken string `json:"id_token"`
}

// VerifyEmailRequest is the body for POST /api/auth/verify-email.
type VerifyEmailRequest struct {
	Token string `json:"token"`
}

// ResendVerificationRequest is the body for POST /api/auth/resend-verification.
type ResendVerificationRequest struct {
	Email string `json:"email"`
}

// AuthResponse is returned on successful login/register.
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// ErrorResponse is a generic error payload.
type ErrorResponse struct {
	Error string `json:"error"`
}
