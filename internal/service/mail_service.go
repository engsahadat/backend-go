package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
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
	if smtpHost == "" && smtpPass == "" {
		border := strings.Repeat("═", 70)
		log.Printf("\n\n%s\n  📧  EMAIL VERIFICATION (SMTP not configured, logging to console)\n%s\n  To:   %s (%s)\n  Subj: %s\n  Link: %s\n%s\n\n",
			border, border, toName, toEmail, subject, verifyURL, border)
		return nil
	}

	// 1. We use Brevo's HTTP API directly if available, which is extremely fast and reliable.
	isBrevoKey := strings.HasPrefix(smtpPass, "xsmtpsib-")

	if isBrevoKey {
		log.Println("🚀 Brevo API Key detected. Sending email via Brevo HTTP API...")
		if err := sendViaBrevoAPI(toEmail, toName, subject, body, smtpFrom, smtpPass); err == nil {
			log.Printf("📧 Verification email successfully sent to %s via Brevo HTTP API\n", toEmail)
			return nil
		} else {
			log.Printf("⚠️ Brevo HTTP API failed (%v). Falling back to standard SMTP...", err)
		}
	}

	// 2. Standard SMTP delivery
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	msg := []byte("To: " + toEmail + "\r\n" +
		"From: " + smtpFrom + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	if err := smtp.SendMail(addr, auth, smtpFrom, []string{toEmail}, msg); err != nil {
		// If SMTP send fails (e.g. unverified sender 'short write' on Brevo, or port block locally),
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

		// If SMTP completely failed and we haven't tried Brevo API yet (e.g. running locally but port blocked),
		// try Brevo API as the final fallback
		if isBrevoKey {
			log.Println("🔄 SMTP failed. Attempting final fallback via Brevo HTTP API...")
			if apiErr := sendViaBrevoAPI(toEmail, toName, subject, body, smtpFrom, smtpPass); apiErr == nil {
				log.Printf("📧 Verification email successfully sent to %s via Brevo HTTP API fallback\n", toEmail)
				return nil
			} else {
				return fmt.Errorf("smtp failed: %v; brevo api fallback failed: %w", err, apiErr)
			}
		}

		return fmt.Errorf("failed to send smtp mail: %w", err)
	}

	log.Printf("📧 Verification email successfully sent to %s via SMTP\n", toEmail)
	return nil
}

func sendViaBrevoAPI(toEmail, toName, subject, body, smtpFrom, apiKey string) error {
	url := "https://api.brevo.com/v3/smtp/email"

	type EmailPerson struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	type BrevoPayload struct {
		Sender      EmailPerson   `json:"sender"`
		To          []EmailPerson `json:"to"`
		Subject     string        `json:"subject"`
		TextContent string        `json:"textContent"`
	}

	payload := BrevoPayload{
		Sender: EmailPerson{
			Name:  "AI Employee Platform",
			Email: smtpFrom,
		},
		To: []EmailPerson{
			{
				Name:  toName,
				Email: toEmail,
			},
		},
		Subject:     subject,
		TextContent: body,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}

	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("brevo api error (status %d): %s", resp.StatusCode, buf.String())
	}

	return nil
}
