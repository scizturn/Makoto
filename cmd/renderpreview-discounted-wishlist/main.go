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

var templateIDs = []string{"discounted_wishlist1.html", "discounted_wishlist2.html", "discounted_wishlist3.html"}

func main() {
	jobPath := env("MAKOTO_PREVIEW_JOB_PATH", "templates/preview/discounted-wishlist-job.json")
	outputPath := env("MAKOTO_PREVIEW_HTML_PATH", "templates/preview/discounted-wishlist-preview.html")
	templateDir := env("MAKOTO_DISCOUNTED_WISHLIST_EMAIL_TEMPLATE_DIR", "templates/discounted_wishlist")
	subject := env("MAKOTO_DISCOUNTED_WISHLIST_EMAIL_SUBJECT", "{{ .FirstName }}, wishlist kamu lagi diskon nih!")
	wishlistURL := env("MAKOTO_DISCOUNTED_WISHLIST_URL", "https://kyou.id/user/wishlist")

	payload, err := os.ReadFile(jobPath)
	if err != nil {
		log.Fatal("failed reading job file: ", err)
	}
	var job domain.DiscountedWishlistJob
	if err := json.Unmarshal(payload, &job); err != nil {
		log.Fatal("failed parsing json: ", err)
	}

	dw := campaign.DiscountedWishlistCampaign{
		WishlistURL:    wishlistURL,
		Closing:        "Jangan sampai kehabisan — yuk cek wishlistmu sekarang di Kyou!",
		RandomIntn:     func(n int) int { return 0 },
	}

	greetingTpl := "Psst {{ .FirstName }}, wishlist incaran kamu lagi diskon nih!"
	greeting := dw.RenderGreeting(greetingTpl, job.User)
	mergeData := dw.BuildMergeData(job.User, job.Items, greeting)

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
		individualPath := fmt.Sprintf("templates/preview/discounted-wishlist%d-preview.html", idx)
		if writeErr := os.WriteFile(individualPath, []byte(html), 0o600); writeErr != nil {
			log.Printf("failed writing %s: %v", individualPath, writeErr)
		}
		rendered = append(rendered, individualPath)
		log.Printf("rendered: %s → %s", tmplID, individualPath)
	}

	wishlisted, fill := 0, 0
	for _, item := range job.Items {
		if item.IsWishlisted {
			wishlisted++
		} else {
			fill++
		}
	}

	index := buildIndexHTML(rendered, time.Now().Unix())
	if err := os.WriteFile(outputPath, []byte(index), 0o600); err != nil {
		log.Fatal("failed writing index: ", err)
	}
	log.Printf("preview index written: %s (user_id=%s wishlisted=%d fill=%d)",
		outputPath, job.UserID, wishlisted, fill)
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
		src := fmt.Sprintf("discounted-wishlist%d-preview.html?v=%d", i+1, version)
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
  <title>Discounted Wishlist Templates Preview</title>
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
