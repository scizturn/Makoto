package email

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The templates are the one place a link can be added without touching Go code, so
// this walks the real HTML on disk and asserts every kyou.id href survives a trip
// through TrackingSender with a UTM stamp on it. A new template that hardcodes a
// kyou.id link is covered automatically; one that defeats the stamp (a link built
// from string concatenation, say) fails here instead of silently going untracked.
func TestEveryKyouLinkInTheRealTemplatesGetsStamped(t *testing.T) {
	root := filepath.Join("..", "..", "templates")
	hrefPattern := regexp.MustCompile(`(?i)href\s*=\s*"([^"]*)"`)

	var checked int
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// preview/ holds generated output, not source templates.
		if d.IsDir() && d.Name() == "preview" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".html") {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		campaignName := filepath.Base(filepath.Dir(path))
		stamped := stampHTML(string(raw), UTM{Source: "makoto", Medium: "email", Campaign: campaignName}, "v1")

		for _, m := range hrefPattern.FindAllStringSubmatch(stamped, -1) {
			link := strings.ReplaceAll(m[1], "&amp;", "&")
			parsed, err := url.Parse(link)
			if err != nil || !isKyouHost(parsed.Hostname()) {
				continue // socials, maps, whatsapp, template placeholders
			}
			checked++
			if parsed.Query().Get("utm_source") != "makoto" {
				t.Errorf("%s: kyou.id link left untracked: %s", path, link)
			}
			if got := parsed.Query().Get("utm_campaign"); got != campaignName {
				t.Errorf("%s: utm_campaign = %q, want %q", path, got, campaignName)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk templates: %v", err)
	}
	if checked == 0 {
		t.Fatal("no kyou.id links found in templates/ — the walk is not seeing the real templates")
	}
	t.Logf("%d kyou.id links across the real templates, all stamped", checked)
}
