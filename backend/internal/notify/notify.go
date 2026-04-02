package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"
)

// Config holds all email delivery configuration.
type Config struct {
	// Provider: "smtp", "mailgun", or "" (disabled)
	Provider string

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	MailgunAPIKey string
	MailgunDomain string
	MailgunFrom   string
	MailgunRegion string // "us" or "eu"; defaults to "us"
}

var (
	cfgMu     sync.RWMutex
	globalCfg Config
)

// SetConfig replaces the active email config. Called at startup and when settings are saved.
func SetConfig(c Config) {
	cfgMu.Lock()
	globalCfg = c
	cfgMu.Unlock()
}

// GetConfig returns the current active email config.
func GetConfig() Config {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return globalCfg
}

// ConfigFromEnv builds a Config from SMTP_* environment variables (legacy fallback).
func ConfigFromEnv() Config {
	host := os.Getenv("SMTP_HOST")
	provider := ""
	if host != "" {
		provider = "smtp"
	}
	return Config{
		Provider:     provider,
		SMTPHost:     host,
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
	}
}

// ConfigFromSettings builds a Config from a DB settings map, falling back to env vars for SMTP fields.
func ConfigFromSettings(settings map[string]string) Config {
	orEnv := func(val, envKey string) string {
		if val != "" {
			return val
		}
		return os.Getenv(envKey)
	}
	c := Config{
		Provider:      settings["email_provider"],
		SMTPHost:      orEnv(settings["smtp_host"], "SMTP_HOST"),
		SMTPPort:      orEnv(settings["smtp_port"], "SMTP_PORT"),
		SMTPUsername:  orEnv(settings["smtp_username"], "SMTP_USERNAME"),
		SMTPPassword:  orEnv(settings["smtp_password"], "SMTP_PASSWORD"),
		SMTPFrom:      orEnv(settings["smtp_from"], "SMTP_FROM"),
		MailgunAPIKey: settings["mailgun_api_key"],
		MailgunDomain: settings["mailgun_domain"],
		MailgunFrom:   settings["mailgun_from"],
		MailgunRegion: settings["mailgun_region"],
	}
	// Infer provider if not explicitly set in DB
	if c.Provider == "" {
		if c.SMTPHost != "" {
			c.Provider = "smtp"
		} else if c.MailgunAPIKey != "" {
			c.Provider = "mailgun"
		}
	}
	return c
}

type webhookPayload struct {
	RunID    string `json:"run_id"`
	Status   string `json:"status"`
	FormName string `json:"form_name"`
	Time     string `json:"time"`
}

// Send fires a webhook POST and/or email for a completed run.
// Safe to call concurrently; call in a goroutine for non-blocking behaviour.
func Send(webhookURL, emailTo, runID, status, formName string) {
	if webhookURL != "" {
		sendWebhook(webhookURL, runID, status, formName)
	}
	if emailTo != "" {
		cfg := GetConfig()
		subject := fmt.Sprintf("[Automation Hub] %s: %s", formName, status)
		body := fmt.Sprintf("Run ID: %s\nForm: %s\nStatus: %s\nTime: %s",
			runID, formName, status, time.Now().UTC().Format(time.RFC3339))
		sendEmail(cfg, emailTo, subject, body)
	}
}

func sendWebhook(url, runID, status, formName string) {
	payload := webhookPayload{
		RunID:    runID,
		Status:   status,
		FormName: formName,
		Time:     time.Now().UTC().Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		log.Printf("[notify] webhook to %s failed: %v", url, err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("[notify] webhook to %s returned HTTP %d", url, resp.StatusCode)
	}
}

// SendPasswordReset emails a password reset link to the given address.
func SendPasswordReset(to, resetURL string) {
	cfg := GetConfig()
	subject := "[Automation Hub] Password Reset Request"
	body := "You requested a password reset for your Automation Hub account.\r\n\r\n" +
		"Click the link below to set a new password. This link expires in 1 hour.\r\n\r\n" +
		resetURL + "\r\n\r\n" +
		"If you did not request this, you can safely ignore this email."
	sendEmail(cfg, to, subject, body)
}

// SendTest sends a test email using the provided config and returns any error.
func SendTest(cfg Config, to string) error {
	subject := "[Automation Hub] Test Email"
	body := "This is a test email from Automation Hub.\r\n\r\nIf you received this, your email configuration is working correctly."
	return sendEmailErr(cfg, to, subject, body)
}

func sendEmail(cfg Config, to, subject, body string) {
	if err := sendEmailErr(cfg, to, subject, body); err != nil {
		log.Printf("[notify] email to %s failed: %v", to, err)
	}
}

func sendEmailErr(cfg Config, to, subject, body string) error {
	switch cfg.Provider {
	case "mailgun":
		return sendMailgun(cfg, to, subject, body)
	case "smtp":
		return sendSMTP(cfg, to, subject, body)
	default:
		return fmt.Errorf("no email provider configured")
	}
}

func sendSMTP(cfg Config, to, subject, body string) error {
	if cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP host not configured")
	}
	port := cfg.SMTPPort
	if port == "" {
		port = "587"
	}
	from := cfg.SMTPFrom
	if from == "" {
		from = cfg.SMTPUsername
	}
	msg := "From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\n\r\n" + body
	addr := cfg.SMTPHost + ":" + port
	var a smtp.Auth
	if cfg.SMTPUsername != "" {
		a = smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	}
	toList := strings.Split(to, ",")
	for i := range toList {
		toList[i] = strings.TrimSpace(toList[i])
	}
	return smtp.SendMail(addr, a, from, toList, []byte(msg))
}

func sendMailgun(cfg Config, to, subject, body string) error {
	if cfg.MailgunAPIKey == "" || cfg.MailgunDomain == "" {
		return fmt.Errorf("Mailgun API key and domain are required")
	}
	from := cfg.MailgunFrom
	if from == "" {
		from = "Automation Hub <mailgun@" + cfg.MailgunDomain + ">"
	}
	region := cfg.MailgunRegion
	if region == "" {
		region = "us"
	}
	var apiBase string
	if region == "eu" {
		apiBase = "https://api.eu.mailgun.net"
	} else {
		apiBase = "https://api.mailgun.net"
	}
	endpoint := fmt.Sprintf("%s/v3/%s/messages", apiBase, cfg.MailgunDomain)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("from", from)
	_ = w.WriteField("to", to)
	_ = w.WriteField("subject", subject)
	_ = w.WriteField("text", body)
	w.Close()

	req, err := http.NewRequest(http.MethodPost, endpoint, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.SetBasicAuth("api", cfg.MailgunAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailgun returned HTTP %d: %s", resp.StatusCode, string(rb))
	}
	return nil
}
