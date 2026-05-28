package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/emailtemplate"
)

func main() {
	jobPath := env("MAKOTO_PREVIEW_JOB_PATH", "/Users/sleepyreinze/Dev/birthday-147044-job.json")
	outputPath := env("MAKOTO_PREVIEW_HTML_PATH", "/Users/sleepyreinze/Dev/birthday-147044-preview.html")
	templateID := env("MAKOTO_PREVIEW_TEMPLATE_ID", "birthday1.html")
	templateDir := env("MAKOTO_EMAIL_TEMPLATE_DIR", "templates/birthday")
	subject := env("MAKOTO_EMAIL_SUBJECT", "Selamat ulang tahun, {{ .Name }}")
	actionURL := env("MAKOTO_ACTION_URL", "https://kyou.id/user/my-voucher")

	payload, err := os.ReadFile(jobPath)
	if err != nil {
		log.Fatal(err)
	}
	var job domain.BirthdayJob
	if err := json.Unmarshal(payload, &job); err != nil {
		log.Fatal(err)
	}

	birthdayCampaign := campaign.BirthdayCampaign{
		TemplateIDs: []string{templateID},
		Closing:     "Selamat merayakan hari spesialmu di Kyou!",
		ActionURL:   actionURL,
		RandomIntn:  func(n int) int { return 0 },
	}
	mergeData := birthdayCampaign.BuildMergeData(job.User, "LOCAL-PREVIEW", job.WishlistItems, job.FYPItems, job.PopularItems)
	_, html, err := emailtemplate.FileRenderer{Dir: templateDir, Subject: subject}.Render(templateID, mergeData)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte(html), 0o600); err != nil {
		log.Fatal(err)
	}
	log.Printf("preview html written: path=%s user_id=%s wishlist=%d fyp=%d popular=%d", outputPath, job.UserID, len(job.WishlistItems), len(job.FYPItems), len(job.PopularItems))
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
