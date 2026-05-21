package voucher

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

type KyouClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func (c KyouClient) IssueBirthdayVoucher(ctx context.Context, user domain.User, birthdayDate time.Time) (string, error) {
	expiry := time.Date(
		birthdayDate.Year(),
		birthdayDate.Month(),
		birthdayDate.Day(),
		0, 0, 0, 0,
		birthdayDate.Location(),
	).Add(14 * 24 * time.Hour)

	payload := map[string]string{
		"user_id":    user.ID,
		"email":      user.Email,
		"campaign":   "birthday-sales",
		"expires_at": expiry.Format(time.RFC3339),
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return "", err
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/api/internal/vouchers/birthday"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("kyou.id voucher issue failed: status=%d", resp.StatusCode)
	}

	var result struct {
		VoucherCode string `json:"voucher_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.VoucherCode == "" {
		return "", fmt.Errorf("kyou.id voucher issue response missing voucher_code")
	}
	return result.VoucherCode, nil
}
