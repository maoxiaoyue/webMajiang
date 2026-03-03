package utils

import (
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPConfig represents the configuration needed to send emails.
type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

var globalSMTPConfig *SMTPConfig

// InitEmail sets up the global SMTP configuration.
func InitEmail(cfg *SMTPConfig) {
	globalSMTPConfig = cfg
}

// SendVerificationEmail sends a verification email to the user.
// Returns the token for logging/testing purposes.
func SendVerificationEmail(to, token, serverAddr string) error {
	link := fmt.Sprintf("http://%s/api/auth/verify?token=%s", serverAddr, token)

	// Format the message
	subject := "Subject: Web Majiang Game Verification Email\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf("Hello,\n\nPlease verify your email by clicking the link: %s\n\nIf you did not request this, please ignore.", link)

	msg := []byte(subject + mime + body)

	// If SMTP is not fully configured, just print to console for development testing
	if globalSMTPConfig == nil || globalSMTPConfig.Host == "" {
		fmt.Printf("=========================================\n")
		fmt.Printf("[TEST] Verification Email Details:\n")
		fmt.Printf("To: %s\n", to)
		fmt.Printf("Link: %s\n", link)
		fmt.Printf("=========================================\n")
		return nil
	}

	addr := fmt.Sprintf("%s:%d", globalSMTPConfig.Host, globalSMTPConfig.Port)

	// In development environments with missing auth, we might bypass CRAM-MD5/PLAIN
	// But standard auth is preferred if username is provided.
	var auth smtp.Auth
	if globalSMTPConfig.Username != "" && globalSMTPConfig.Password != "" && !strings.HasPrefix(globalSMTPConfig.Host, "localhost") {
		auth = smtp.PlainAuth("", globalSMTPConfig.Username, globalSMTPConfig.Password, globalSMTPConfig.Host)
	}

	err := smtp.SendMail(addr, auth, globalSMTPConfig.From, []string{to}, msg)
	if err != nil {
		fmt.Printf("Failed to send email to %s: %v\n", to, err)
		return err
	}

	return nil
}
