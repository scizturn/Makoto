package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/emailtemplate"
)

var templateIDs = []string{"po_ready1.html", "po_ready2.html", "po_ready3.html"}

func main() {
	jobPath := env("MAKOTO_PREVIEW_JOB_PATH", "templates/preview/po-ready-job.json")
	outputPath := env("MAKOTO_PREVIEW_HTML_PATH", "templates/preview/po-ready-preview.html")
	templateDir := env("MAKOTO_PO_READY_EMAIL_TEMPLATE_DIR", "templates/po_ready")
	subject := env("MAKOTO_PO_READY_EMAIL_SUBJECT", "{{ .FirstName }}, PO kamu udah sampai — yuk lunasin!")
	historyURL := env("MAKOTO_PO_READY_URL", "https://kyou.id/user/history")

	job := sampleJob()
	if payload, err := os.ReadFile(jobPath); err == nil {
		if err := json.Unmarshal(payload, &job); err != nil {
			log.Fatal("failed parsing json: ", err)
		}
	} else {
		log.Printf("job file %s not found; using built-in sample job", jobPath)
	}

	pr := campaign.PoReadyCampaign{
		HistoryURL: historyURL,
		Closing:    "Barang udah di tangan Kyou — tinggal selesaikan pembayaran biar bisa segera dikirim!",
		RandomIntn: func(n int) int { return 0 },
	}

	greeting := pr.RenderGreeting("Omatase, {{ .FirstName }}! Barang PO kamu akhirnya sampai di Kyou.", job.User)
	mergeData := pr.BuildMergeData(job.User, job.OrderID, job.Items, job.Remaining, job.DownPayment, job.ETA, greeting)

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
		individualPath := fmt.Sprintf("templates/preview/po-ready%d-preview.html", idx)
		if writeErr := os.WriteFile(individualPath, []byte(html), 0o600); writeErr != nil {
			log.Printf("failed writing %s: %v", individualPath, writeErr)
		}
		rendered = append(rendered, individualPath)
		log.Printf("rendered: %s → %s", tmplID, individualPath)
	}

	index := buildIndexHTML(rendered, time.Now().Unix())
	if err := os.WriteFile(outputPath, []byte(index), 0o600); err != nil {
		log.Fatal("failed writing index: ", err)
	}
	log.Printf("preview index written: %s (order_id=%s items=%d remaining=%d)", outputPath, job.OrderID, len(job.Items), job.Remaining)
}

func sampleJob() domain.PoReadyJob {
	return domain.PoReadyJob{
		ID:      "preview-po-ready-sample",
		OrderID: "123456",
		UserID:  "1",
		User:    domain.User{ID: "1", Name: "Reinze Tanaka", Email: "reinze@example.test", IsActive: true},
		Items: []domain.PoReadyItem{
			{ID: "1", Name: "Hatsune Miku 1/7 Scale Figure", URL: "https://kyou.id/items/1/", ImageURL: "https://kyoucdn.id/static/assets/brand_logo.png", Price: 2350000, Quantity: 1},
			{ID: "2", Name: "Nendoroid Frieren", URL: "https://kyou.id/items/2/", ImageURL: "https://kyoucdn.id/static/assets/brand_logo.png", Price: 850000, Quantity: 2},
		},
		Remaining:   2100000,
		DownPayment: 1100000,
		ETA:         "Juli 2026",
		Attempt:     1,
	}
}

func buildIndexHTML(rendered []string, version int64) string {
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
		src := fmt.Sprintf("po-ready%d-preview.html?v=%d", i+1, version)
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
  <title>PO Ready Templates Preview</title>
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
