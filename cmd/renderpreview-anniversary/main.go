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
	jobPath := env("MAKOTO_PREVIEW_JOB_PATH", "templates/preview/anniversary-job.json")
	outputPath := env("MAKOTO_PREVIEW_HTML_PATH", "templates/preview/anniversary-preview.html")
	templateID := env("MAKOTO_PREVIEW_TEMPLATE_ID", "anniversary1.html")
	templateDir := env("MAKOTO_ANNIVERSARY_EMAIL_TEMPLATE_DIR", "templates/anniversary")
	subject := env("MAKOTO_ANNIVERSARY_EMAIL_SUBJECT", "Happy Anniversary, {{ .Name }}! 🎉")
	actionURL := env("MAKOTO_ACTION_URL", "https://kyou.id/user/my-voucher")

	payload, err := os.ReadFile(jobPath)
	if err != nil {
		log.Fatal("failed reading job file (make sure templates/preview/anniversary-job.json exists): ", err)
	}
	var job domain.AnniversaryJob
	if err := json.Unmarshal(payload, &job); err != nil {
		log.Fatal("failed parsing json: ", err)
	}

	anniversaryCampaign := campaign.AnniversaryCampaign{
		TemplateIDs: []string{templateID},
		Closing:     "Terima kasih sudah menjadi bagian dari Kyou! 🎉",
		ActionURL:   actionURL,
		RandomIntn:  func(n int) int { return 0 },
	}
	
	mergeData := anniversaryCampaign.BuildMergeData(job.User, "ANVPREVIEW2026", job.WishlistItems, job.Years, job.HistoricalItem)
	_, html, err := emailtemplate.FileRenderer{Dir: templateDir, Subject: subject}.Render(templateID, mergeData)
	if err != nil {
		log.Fatal("failed rendering html: ", err)
	}
	
	if err := os.WriteFile(outputPath, []byte(html), 0o600); err != nil {
		log.Fatal("failed writing html: ", err)
	}
	log.Printf("preview html written: path=%s user_id=%s years=%d", outputPath, job.UserID, job.Years)
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
