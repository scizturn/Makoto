package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type KirimClient struct {
	BaseURL    string
	Username   string
	APIToken   string
	HTTPClient *http.Client
}

func (c KirimClient) SendTemplate(ctx context.Context, msg domain.EmailMessage) (domain.SendResult, error) {
	payload := map[string]any{
		"from_email":        msg.FromEmail,
		"from_name":         msg.FromName,
		"to":                msg.ToEmail,
		"template_id":       msg.TemplateID,
		"substitution_data": msg.SubstitutionData,
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return domain.SendResult{}, err
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/api/domains/" + msg.Domain + "/message/template"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return domain.SendResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Username, c.APIToken)

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return domain.SendResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.SendResult{}, fmt.Errorf("kirim.email template send failed: status=%d", resp.StatusCode)
	}

	var result struct {
		ID        string `json:"id"`
		MessageID string `json:"message_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.SendResult{}, nil
	}
	if result.MessageID == "" {
		result.MessageID = result.ID
	}
	return domain.SendResult{MessageID: result.MessageID}, nil
}

func (c KirimClient) Validate(ctx context.Context, email string) (bool, error) {
	payload := map[string]string{"email": email}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return false, err
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/api/email/validate/strict"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Username, c.APIToken)

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("kirim.email validation failed: status=%d", resp.StatusCode)
	}

	var result struct {
		Valid  *bool  `json:"valid"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if result.Valid != nil {
		return *result.Valid, nil
	}
	return strings.EqualFold(result.Status, "valid"), nil
}
