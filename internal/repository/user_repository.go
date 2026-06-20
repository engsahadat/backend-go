package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/your-org/ai-employee-platform/internal/database"
	"github.com/your-org/ai-employee-platform/internal/domain"
)

// CreateUser inserts a new user and returns it with the generated ID.
func CreateUser(u *domain.User) error {
	isVerifiedInt := 0
	if u.IsVerified {
		isVerifiedInt = 1
	}
	res, err := database.DB.Exec(
		`INSERT INTO users (email, name, password_hash, avatar_url, provider, provider_id, is_verified, verification_token, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Email, u.Name, u.PasswordHash, u.AvatarURL, u.Provider, u.ProviderID, isVerifiedInt, u.VerificationToken, time.Now(), time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	id, _ := res.LastInsertId()
	u.ID = id
	return nil
}

// GetUserByEmail returns a user by email or nil if not found.
func GetUserByEmail(email string) (*domain.User, error) {
	u := &domain.User{}
	var isVerifiedInt int
	err := database.DB.QueryRow(
		`SELECT id, email, name, password_hash, avatar_url, provider, provider_id, is_verified, verification_token, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AvatarURL, &u.Provider, &u.ProviderID, &isVerifiedInt, &u.VerificationToken, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	u.IsVerified = isVerifiedInt == 1
	return u, nil
}

// GetUserByID returns a user by ID.
func GetUserByID(id int64) (*domain.User, error) {
	u := &domain.User{}
	var isVerifiedInt int
	err := database.DB.QueryRow(
		`SELECT id, email, name, password_hash, avatar_url, provider, provider_id, is_verified, verification_token, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AvatarURL, &u.Provider, &u.ProviderID, &isVerifiedInt, &u.VerificationToken, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	u.IsVerified = isVerifiedInt == 1
	return u, nil
}

// GetUserByProvider returns a user by OAuth provider + provider-specific ID.
func GetUserByProvider(provider, providerID string) (*domain.User, error) {
	u := &domain.User{}
	var isVerifiedInt int
	err := database.DB.QueryRow(
		`SELECT id, email, name, password_hash, avatar_url, provider, provider_id, is_verified, verification_token, created_at, updated_at
		 FROM users WHERE provider = ? AND provider_id = ?`, provider, providerID,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AvatarURL, &u.Provider, &u.ProviderID, &isVerifiedInt, &u.VerificationToken, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by provider: %w", err)
	}
	u.IsVerified = isVerifiedInt == 1
	return u, nil
}

// GetUserByVerificationToken returns a user by verification token or nil if not found.
func GetUserByVerificationToken(token string) (*domain.User, error) {
	u := &domain.User{}
	var isVerifiedInt int
	err := database.DB.QueryRow(
		`SELECT id, email, name, password_hash, avatar_url, provider, provider_id, is_verified, verification_token, created_at, updated_at
		 FROM users WHERE verification_token = ?`, token,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AvatarURL, &u.Provider, &u.ProviderID, &isVerifiedInt, &u.VerificationToken, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by verification token: %w", err)
	}
	u.IsVerified = isVerifiedInt == 1
	return u, nil
}

// VerifyUser marks a user's email as verified and clears the token.
func VerifyUser(userID int64) error {
	_, err := database.DB.Exec(
		`UPDATE users SET is_verified = 1, verification_token = '', updated_at = ? WHERE id = ?`,
		time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("verify user in db: %w", err)
	}
	return nil
}

// UpdateVerificationToken updates a user's verification token.
func UpdateVerificationToken(userID int64, token string) error {
	_, err := database.DB.Exec(
		`UPDATE users SET verification_token = ?, updated_at = ? WHERE id = ?`,
		token, time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("update verification token in db: %w", err)
	}
	return nil
}
