package notify

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

type webhookPayload struct {
	RunID    string `json:"run_id"`
	Status   string `json:"status"`
	FormName string `json:"form_name"`
	Time     string `json:"time"`
}

// Send fires a webhook POST and/or SMTP email for a completed run.
// Safe to call concurrently; call in a goroutine for non-blocking behaviour.
// Empty webhookURL or emailTo are silently skipped.
func Send(webhookURL, emailTo, runID, status, formName string) {
	if webhookURL != "" {
		sendWebhook(webhookURL, runID, status, formName)
	}
	if emailTo != "" {
		sendEmail(emailTo, runID, status, formName)
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

func sendEmail(to, runID, status, formName string) {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	if host == "" {
		log.Printf("[notify] SMTP_HOST not configured, skipping email to %s", to)
		return
	}
	if port == "" {
		port = "587"
	}
	if from == "" {
		from = username
	}

	subject := fmt.Sprintf("[Ansible UI] %s: %s", formName, status)
	msgBody := fmt.Sprintf("Run ID: %s\nForm: %s\nStatus: %s\nTime: %s",
		runID, formName, status, time.Now().UTC().Format(time.RFC3339))
	msg := "From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\n\r\n" + msgBody

	addr := host + ":" + port
	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	toList := strings.Split(to, ",")
	for i := range toList {
		toList[i] = strings.TrimSpace(toList[i])
	}

	if err := smtp.SendMail(addr, auth, from, toList, []byte(msg)); err != nil {
		log.Printf("[notify] email to %s failed: %v", to, err)
	}
}
