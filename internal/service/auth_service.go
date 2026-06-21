package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"github.com/your-org/ai-employee-platform/internal/domain"
	"github.com/your-org/ai-employee-platform/internal/repository"
)

// jwtSecret should come from env in production.
var jwtSecret = []byte("ai-employee-super-secret-key-change-in-prod")

// Helper to generate a random 32-character hex token
func generateVerificationToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- Email/Password Auth ----------

// Register creates a new user with hashed password and generates a verification token.
func Register(req domain.RegisterRequest) (*domain.AuthResponse, error) {
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("email and password are required")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	// Check if email already taken.
	existing, err := repository.GetUserByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	verificationToken := generateVerificationToken()

	user := &domain.User{
		Email:             strings.ToLower(strings.TrimSpace(req.Email)),
		Name:              strings.TrimSpace(req.Name),
		PasswordHash:      string(hash),
		Provider:          "email",
		IsVerified:        true,
		VerificationToken: verificationToken,
	}
	if err := repository.CreateUser(user); err != nil {
		return nil, err
	}

	token, err := generateJWT(user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: *user}, nil
}

// Login verifies credentials and returns a JWT.
func Login(req domain.LoginRequest) (*domain.AuthResponse, error) {
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("email and password are required")
	}

	user, err := repository.GetUserByEmail(strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}
	if user.Provider != "email" {
		return nil, fmt.Errorf("this account uses %s login", user.Provider)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Removed verification check to allow all users to log in
	// if !user.IsVerified {
	// 	return nil, errors.New("please verify your email to log in")
	// }

	token, err := generateJWT(user)
	if err != nil {
		return nil, err
	}
	return &domain.AuthResponse{Token: token, User: *user}, nil
}

// ---------- Google OAuth ----------

// GoogleUserInfo is the subset of fields we use from Google's tokeninfo endpoint.
type GoogleUserInfo struct {
	Sub       string `json:"sub"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Picture   string `json:"picture"`
	ExpiresIn int    `json:"expires_in"`
}

// LoginWithGoogle verifies a Google ID-token via Google's tokeninfo endpoint
// and either logs in or auto-registers the user.
func LoginWithGoogle(idToken string) (*domain.AuthResponse, error) {
	if idToken == "" {
		return nil, errors.New("id_token is required")
	}

	// Verify token with Google.
	info, err := verifyGoogleToken(idToken)
	if err != nil {
		return nil, fmt.Errorf("google auth: %w", err)
	}

	// Check if user already exists by provider+sub.
	user, err := repository.GetUserByProvider("google", info.Sub)
	if err != nil {
		return nil, err
	}

	if user == nil {
		// Also check by email — maybe they registered with email first.
		user, err = repository.GetUserByEmail(info.Email)
		if err != nil {
			return nil, err
		}
	}

	if user == nil {
		// Auto-register.
		user = &domain.User{
			Email:      info.Email,
			Name:       info.Name,
			AvatarURL:  info.Picture,
			Provider:   "google",
			ProviderID: info.Sub,
			IsVerified: true,
		}
		if err := repository.CreateUser(user); err != nil {
			return nil, err
		}
	} else if !user.IsVerified {
		// Auto-verify if they verify using Google login
		if err := repository.VerifyUser(user.ID); err != nil {
			return nil, err
		}
		user.IsVerified = true
	}

	token, err := generateJWT(user)
	if err != nil {
		return nil, err
	}
	return &domain.AuthResponse{Token: token, User: *user}, nil
}

// verifyGoogleToken verifies the Google ID token locally using cached Google public keys.
func verifyGoogleToken(idTokenStr string) (*GoogleUserInfo, error) {
	payload, err := idtoken.Validate(context.Background(), idTokenStr, "")
	if err != nil {
		return nil, fmt.Errorf("validate google token: %w", err)
	}

	sub, okSub := payload.Claims["sub"].(string)
	email, okEmail := payload.Claims["email"].(string)

	if !okSub || !okEmail {
		return nil, errors.New("invalid google token claims")
	}

	info := &GoogleUserInfo{
		Sub:   sub,
		Email: email,
	}

	if name, ok := payload.Claims["name"].(string); ok {
		info.Name = name
	}
	if picture, ok := payload.Claims["picture"].(string); ok {
		info.Picture = picture
	}
	if expFloat, ok := payload.Claims["exp"].(float64); ok {
		info.ExpiresIn = int(expFloat - float64(time.Now().Unix()))
	}

	return info, nil
}

// ---------- JWT Helpers ----------

// generateJWT creates a signed JWT for the user (valid 7 days).
func generateJWT(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken parses and validates a JWT, returning the user ID.
func ValidateToken(tokenStr string) (int64, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return 0, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token")
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, errors.New("invalid token claims")
	}
	return int64(sub), nil
}

// VerifyEmail checks if a verification token is valid, verifies the user, and clears the token.
func VerifyEmail(token string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("token is required")
	}

	user, err := repository.GetUserByVerificationToken(token)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("invalid or expired verification token")
	}

	if user.IsVerified {
		return errors.New("email already verified")
	}

	return repository.VerifyUser(user.ID)
}

// ResendVerification generates a new token and resends the verification email.
func ResendVerification(email string) error {
	if strings.TrimSpace(email) == "" {
		return errors.New("email is required")
	}

	user, err := repository.GetUserByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("no account found with this email")
	}

	if user.IsVerified {
		return errors.New("email already verified")
	}

	newToken := generateVerificationToken()
	if err := repository.UpdateVerificationToken(user.ID, newToken); err != nil {
		return err
	}

	// Send verification email asynchronously
	go func() {
		if err := SendVerificationEmail(user.Email, user.Name, newToken); err != nil {
			log.Printf("❌ Failed to resend verification email: %v", err)
		}
	}()

	return nil
}

