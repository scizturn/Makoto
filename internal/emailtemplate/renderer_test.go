package emailtemplate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRendererRendersBirthdayHTMLTemplate(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "birthday1.html")
	err := os.WriteFile(templatePath, []byte(`<!doctype html>
<html>
<body>
<h1>Happy birthday {{ .Name }}</h1>
<p>Voucher: {{ .VoucherCode }}</p>
<section>{{ .WishlistHTML }}</section>
<section>{{ .FYPHTML }}</section>
<a href="{{ .ActionURL }}">Use voucher</a>
<p>{{ .Closing }}</p>
</body>
</html>`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	renderer := FileRenderer{Dir: dir, Subject: "Selamat ulang tahun, {{ .Name }}"}
	subject, html, err := renderer.Render("birthday1.html", map[string]any{
		"name":          "Ruby <script>",
		"voucher_code":  "HBD-RUBY",
		"wishlist_html": `<ul><li>Figure Ruby</li></ul>`,
		"fyp_html":      `<ul><li>Popular Series</li></ul>`,
		"action_url":    "https://kyou.id/account/vouchers?x=<bad>",
		"closing":       "Selamat merayakan hari spesialmu di Kyou!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if subject != "Selamat ulang tahun, Ruby <script>" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if strings.Contains(html, "Ruby <script>") {
		t.Fatalf("expected name to be escaped in html, got %q", html)
	}
	for _, want := range []string{
		"Ruby &lt;script&gt;",
		"HBD-RUBY",
		"<ul><li>Figure Ruby</li></ul>",
		"<ul><li>Popular Series</li></ul>",
		"https://kyou.id/account/vouchers?x=%3cbad%3e",
		"Selamat merayakan hari spesialmu di Kyou!",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected rendered html to contain %q, got %q", want, html)
		}
	}
}
