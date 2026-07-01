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

var templateIDs = []string{"winback1.html", "winback2.html", "winback3.html"}

func main() {
	jobPath := env("MAKOTO_PREVIEW_JOB_PATH", "templates/preview/winback-job.json")
	outputPath := env("MAKOTO_PREVIEW_HTML_PATH", "templates/preview/winback-preview.html")
	templateDir := env("MAKOTO_WINBACK_EMAIL_TEMPLATE_DIR", "templates/winback")
	subject := env("MAKOTO_WINBACK_EMAIL_SUBJECT", "{{ .FirstName }}, kita kangen kamu nih!")
	actionURL := env("MAKOTO_WINBACK_ACTION_URL", "https://kyou.id/user/my-voucher")

	payload, err := os.ReadFile(jobPath)
	if err != nil {
		log.Fatal("failed reading job file: ", err)
	}
	var job domain.WinbackJob
	if err := json.Unmarshal(payload, &job); err != nil {
		log.Fatal("failed parsing json: ", err)
	}

	wb := campaign.WinbackCampaign{
		ActionURL:  actionURL,
		Closing:    "Yuk balik lagi, masih banyak koleksi keren yang nunggu kamu di Kyou!",
		RandomIntn: func(n int) int { return 0 },
	}

	greetingTpl := "Hei {{ .FirstName }}, udah lama banget nih nggak ketemu!"
	greeting := wb.RenderGreeting(greetingTpl, job.User)
	mergeData := wb.BuildMergeData(job.User, job.VoucherCode, job.WishlistItems, job.HistoricalItem, greeting)

	renderer := emailtemplate.FileRenderer{Dir: templateDir, Subject: subject}

	var rendered []string
	for _, tmplID := range templateIDs {
		tmplPath := templateDir + "/" + tmplID
		if _, statErr := os.Stat(tmplPath); os.IsNotExist(statErr) {
			log.Printf("skipping %s (not found)", tmplID)
			rendered = append(rendered, "")
			continue
		}
		_, html, renderErr := renderer.Render(tmplID, mergeData)
		if renderErr != nil {
			log.Printf("failed rendering %s: %v", tmplID, renderErr)
			rendered = append(rendered, "")
			continue
		}
		idx := len(rendered) + 1
		individualPath := fmt.Sprintf("templates/preview/winback%d-preview.html", idx)
		if writeErr := os.WriteFile(individualPath, []byte(html), 0o600); writeErr != nil {
			log.Printf("failed writing %s: %v", individualPath, writeErr)
		}
		rendered = append(rendered, individualPath)
		log.Printf("rendered: %s → %s", tmplID, individualPath)
	}

	index := buildIndexHTML(rendered)
	if err := os.WriteFile(outputPath, []byte(index), 0o600); err != nil {
		log.Fatal("failed writing index: ", err)
	}
	log.Printf("preview index written: %s (user_id=%s voucher=%s wishlist=%d historical=%s)",
		outputPath, job.UserID, job.VoucherCode, len(job.WishlistItems), job.HistoricalItem.Name)
}

func buildIndexHTML(rendered []string) string {
	var sections strings.Builder
	for i, path := range rendered {
		tmplID := templateIDs[i]
		if path == "" {
			sections.WriteString(fmt.Sprintf(`
  <div class="section">
    <div class="label">Template %d — %s</div>
    <div class="placeholder">Template file tidak ditemukan</div>
  </div>`, i+1, tmplID))
			continue
		}
		src := fmt.Sprintf("winback%d-preview.html", i+1)
		sections.WriteString(fmt.Sprintf(`
  <div class="section">
    <div class="label">Template %d &mdash; %s</div>
    <iframe src="%s" onload="resizeIframe(this)" scrolling="no"></iframe>
  </div>`, i+1, tmplID, src))
	}

	return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Winback Templates Preview</title>
  <style>
    * { box-sizing: border-box; }
    body { margin: 0; padding: 0; background: #cbd5e1; font-family: Arial, sans-serif; }
    .wrapper { padding: 32px 20px; }
    .section { margin-bottom: 48px; }
    .label { text-align: center; padding: 0 0 10px; font-size: 11px; font-weight: 700; color: #64748b; letter-spacing: 2px; text-transform: uppercase; }
    iframe { display: block; margin: 0 auto; border: none; width: 720px; background: #fff; }
    .placeholder { width: 720px; margin: 0 auto; padding: 40px; background: #f1f5f9; border: 2px dashed #cbd5e1; text-align: center; color: #94a3b8; font-size: 14px; border-radius: 8px; }
  </style>
  <script>
    function resizeIframe(el) {
      try { el.style.height = el.contentDocument.body.scrollHeight + 'px'; } catch(e) {}
    }
  </script>
</head>
<body>
<div class="wrapper">` + sections.String() + `
</div>
</body>
</html>`
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
