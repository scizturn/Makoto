package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/emailtemplate"
)

var templateIDs = []string{"wishlist_back_in1.html", "wishlist_back_in2.html", "wishlist_back_in3.html"}

func main() {
	jobPath := env("MAKOTO_PREVIEW_JOB_PATH", "templates/preview/wishlist-back-in-job.json")
	outputPath := env("MAKOTO_PREVIEW_HTML_PATH", "templates/preview/wishlist-back-in-preview.html")
	templateDir := env("MAKOTO_WISHLIST_BACK_IN_EMAIL_TEMPLATE_DIR", "templates/wishlist_back_in")
	payload, err := os.ReadFile(jobPath)
	if err != nil {
		log.Fatal(err)
	}
	var job domain.WishlistBackInJob
	if err := json.Unmarshal(payload, &job); err != nil {
		log.Fatal(err)
	}
	c := campaign.WishlistBackInCampaign{ActionURL: "https://kyou.id/user/my-voucher", Closing: "Mumpung sudah kembali, jangan sampai kelewatan lagi ya!"}
	greeting := c.RenderGreeting("Omatase, {{ .FirstName }}! Yang kamu tunggu akhirnya balik.", job.User)
	mergeData := c.BuildMergeData(job.User, job.VoucherCode, job.Items, job.CompanionItem, greeting)
	renderer := emailtemplate.FileRenderer{Dir: templateDir, Subject: "{{ .FirstName }}, wishlist kamu tersedia lagi!"}
	for index, templateID := range templateIDs {
		_, body, err := renderer.Render(templateID, mergeData)
		if err != nil {
			log.Fatal(err)
		}
		path := fmt.Sprintf("templates/preview/wishlist-back-in%d-preview.html", index+1)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			log.Fatal(err)
		}
	}
	if err := os.WriteFile(outputPath, []byte(indexHTML()), 0o600); err != nil {
		log.Fatal(err)
	}
	log.Printf("wishlist back in preview written: %s", outputPath)
}

func indexHTML() string {
	var sections strings.Builder
	for index, templateID := range templateIDs {
		fmt.Fprintf(&sections, `<section><p>Template %d &mdash; %s</p><iframe src="wishlist-back-in%d-preview.html" onload="this.style.height=this.contentDocument.body.scrollHeight+'px'"></iframe></section>`, index+1, templateID, index+1)
	}
	return `<!doctype html><html><head><meta charset="utf-8"><title>Wishlist Back In Preview</title><style>body{margin:0;padding:30px;background:#cbd5e1;font-family:Arial,sans-serif}section{margin:0 0 42px}p{text-align:center;color:#475569;font-size:12px;font-weight:700;text-transform:uppercase}iframe{display:block;width:720px;margin:auto;border:0;background:#fff}</style></head><body>` + sections.String() + `</body></html>`
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
