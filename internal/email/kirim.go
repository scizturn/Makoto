package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	if msg.HTMLBody != "" {
		return c.sendTransactionalV4(ctx, msg)
	}

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
	responseBody := readResponseBody(resp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.SendResult{StatusCode: resp.StatusCode, Response: responseBody}, fmt.Errorf("kirim.email template send failed: status=%d body=%s", resp.StatusCode, responseBody)
	}

	var result struct {
		ID        string `json:"id"`
		MessageID string `json:"message_id"`
	}
	if responseBody != "" {
		if err := json.NewDecoder(strings.NewReader(responseBody)).Decode(&result); err != nil {
			return domain.SendResult{StatusCode: resp.StatusCode, Response: responseBody}, nil
		}
	}
	if result.MessageID == "" {
		result.MessageID = result.ID
	}
	return domain.SendResult{MessageID: result.MessageID, StatusCode: resp.StatusCode, Response: responseBody}, nil
}

func (c KirimClient) sendTransactionalV4(ctx context.Context, msg domain.EmailMessage) (domain.SendResult, error) {
	form := url.Values{}
	form.Set("from", msg.FromEmail)
	if msg.FromName != "" {
		form.Set("from_name", msg.FromName)
	}
	form.Set("to", msg.ToEmail)
	form.Set("subject", msg.Subject)
	form.Set("html", msg.HTMLBody)
	if msg.TextBody != "" {
		form.Set("text", msg.TextBody)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/api/v4/transactional/message"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(form.Encode()))
	if err != nil {
		return domain.SendResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("domain", msg.Domain)
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
	responseBody := readResponseBody(resp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.SendResult{StatusCode: resp.StatusCode, Response: responseBody}, fmt.Errorf("kirim.email transactional send failed: status=%d body=%s", resp.StatusCode, responseBody)
	}

	var result struct {
		ID        string `json:"id"`
		MessageID string `json:"message_id"`
	}
	if responseBody != "" {
		if err := json.NewDecoder(strings.NewReader(responseBody)).Decode(&result); err != nil {
			return domain.SendResult{StatusCode: resp.StatusCode, Response: responseBody}, nil
		}
	}
	if result.MessageID == "" {
		result.MessageID = result.ID
	}
	return domain.SendResult{MessageID: result.MessageID, StatusCode: resp.StatusCode, Response: responseBody}, nil
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
		return false, fmt.Errorf("kirim.email validation failed: status=%d body=%s", resp.StatusCode, readResponseBody(resp))
	}

	var result struct {
		Valid  *bool  `json:"valid"`
		Status string `json:"status"`
		Data   struct {
			IsValid *bool `json:"is_valid"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if result.Valid != nil {
		return *result.Valid, nil
	}
	if result.Data.IsValid != nil {
		return *result.Data.IsValid, nil
	}
	return strings.EqualFold(result.Status, "valid"), nil
}

func readResponseBody(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return err.Error()
	}
	return strings.TrimSpace(string(body))
}
