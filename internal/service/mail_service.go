package service

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
)

// SendVerificationEmail sends a verification email to the user.
// If SMTP environment variables are not set, it prints the verification link to the console logs.
func SendVerificationEmail(toEmail, toName, token string) error {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		if os.Getenv("RENDER") == "true" {
			frontendURL = "https://www.bdaiemployee.com"
		} else {
			frontendURL = "http://localhost:3000"
		}
	}
	frontendURL = strings.TrimSuffix(frontendURL, "/")
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", frontendURL, token)

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	smtpFrom := os.Getenv("SMTP_FROM")

	if smtpFrom == "" {
		smtpFrom = smtpUser
	}
	if smtpFrom == "" {
		smtpFrom = "no-reply@aiemployee.com"
	}

	subject := "Verify Your Email - AI Employee Platform"
	body := fmt.Sprintf("Hello %s,\n\nThank you for registering at AI Employee Platform!\n\nPlease verify your email address by clicking the link below:\n%s\n\nThis link will expire in 24 hours.\n\nBest regards,\nAI Employee Platform Team", toName, verifyURL)

	// If SMTP_HOST is not configured, fall back to console logging
	if smtpHost == "" {
		border := strings.Repeat("═", 70)
		log.Printf("\n\n%s\n  📧  EMAIL VERIFICATION (SMTP not configured, logging to console)\n%s\n  To:   %s (%s)\n  Subj: %s\n  Link: %s\n%s\n\n",
			border, border, toName, toEmail, subject, verifyURL, border)
		return nil
	}

	// Send real email via SMTP
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	msg := []byte("To: " + toEmail + "\r\n" +
		"From: " + smtpFrom + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	if err := smtp.SendMail(addr, auth, smtpFrom, []string{toEmail}, msg); err != nil {
		// If SMTP send fails (e.g. unverified sender 'short write' on Brevo),
		// retry using smtpUser which is guaranteed to be a verified sender for the authenticated user.
		if smtpFrom != smtpUser && smtpUser != "" {
			log.Printf("⚠️ SMTP sending failed with sender %s (%v). Retrying with authenticated user %s...\n", smtpFrom, err, smtpUser)
			msgRetry := []byte("To: " + toEmail + "\r\n" +
				"From: " + smtpUser + "\r\n" +
				"Subject: " + subject + "\r\n" +
				"\r\n" +
				body + "\r\n")
			if retryErr := smtp.SendMail(addr, auth, smtpUser, []string{toEmail}, msgRetry); retryErr == nil {
				log.Printf("📧 Verification email successfully sent to %s via SMTP using fallback authenticated user sender\n", toEmail)
				return nil
			} else {
				err = fmt.Errorf("retry with smtpUser failed: %w (original error: %v)", retryErr, err)
			}
		}
		return fmt.Errorf("failed to send smtp mail: %w", err)
	}

	log.Printf("📧 Verification email successfully sent to %s via SMTP\n", toEmail)
	return nil
}
