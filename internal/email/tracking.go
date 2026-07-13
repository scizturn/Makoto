package email

import (
	"context"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/kyou-id/makoto/internal/domain"
)

// UTM carries the campaign attribution stamped onto every kyou.id link leaving
// this service. Mail clients strip or forge the Referer header (Gmail proxies it,
// most native clients send none at all), so the attribution has to travel inside
// the URL itself if kyou.id is to know a visit came from an email.
type UTM struct {
	Source   string // utm_source: the system that sent the mail, e.g. "makoto"
	Medium   string // utm_medium: the channel, e.g. "email"
	Campaign string // utm_campaign: the feature, e.g. "po_ready"
}

// TrackingSender rewrites every kyou.id href in the outgoing message so clicks
// are attributable, then hands the message to Next. Wrapping the Sender means the
// stamp lands on links from all three sources at once -- Yukari's SQL-built item
// URLs, the Go-rendered grids in internal/campaign, and the hardcoded links inside
// the HTML templates -- and any template added later is covered without edits.
type TrackingSender struct {
	Next Sender
	UTM  UTM
}

func WithUTM(next Sender, u UTM) Sender {
	if u.Source == "" || u.Medium == "" {
		return next
	}
	return TrackingSender{Next: next, UTM: u}
}

func (s TrackingSender) SendTemplate(ctx context.Context, msg domain.EmailMessage) (domain.SendResult, error) {
	// The template variant rides along as utm_content, so the deterministic
	// template pick is measurable: birthday1 vs birthday2 vs birthday3. File-backed
	// templates carry a .html suffix in their ID; it is noise in a report.
	content := strings.TrimSuffix(msg.TemplateID, ".html")

	msg.HTMLBody = stampHTML(msg.HTMLBody, s.UTM, content)

	// The kirim.email template path carries links inside substitution data instead
	// of an HTML body: pre-rendered grids ({{ .WishlistHTML }}) and bare URLs
	// (action_url). Stamp both shapes.
	if len(msg.SubstitutionData) > 0 {
		stamped := make(map[string]any, len(msg.SubstitutionData))
		for k, v := range msg.SubstitutionData {
			if str, ok := v.(string); ok {
				stamped[k] = stampValue(str, s.UTM, content)
				continue
			}
			stamped[k] = v
		}
		msg.SubstitutionData = stamped
	}

	return s.Next.SendTemplate(ctx, msg)
}

var hrefPattern = regexp.MustCompile(`(?i)href\s*=\s*"([^"]*)"`)

// StampHTML is what the preview renderers call so a preview shows the exact links
// a recipient would click, UTM and all, instead of the bare template links.
func StampHTML(body string, u UTM, templateID string) string {
	return stampHTML(body, u, strings.TrimSuffix(templateID, ".html"))
}

func stampHTML(body string, u UTM, content string) string {
	if body == "" {
		return body
	}
	return hrefPattern.ReplaceAllStringFunc(body, func(match string) string {
		raw := hrefPattern.FindStringSubmatch(match)[1]
		stamped := stampURL(html.UnescapeString(raw), u, content)
		if stamped == "" {
			return match
		}
		// Re-escape the separator so the attribute stays well-formed HTML.
		return `href="` + strings.ReplaceAll(stamped, "&", "&amp;") + `"`
	})
}

func stampValue(v string, u UTM, content string) string {
	if strings.Contains(strings.ToLower(v), "href=") {
		return stampHTML(v, u, content)
	}
	if stamped := stampURL(v, u, content); stamped != "" {
		return stamped
	}
	return v
}

// stampURL returns the URL with UTM params appended, or "" when the URL must be
// left alone: a non-kyou.id host (socials, kirim.email's unsubscribe), a mailto:,
// or a link that already carries a utm_source (so a re-render of the same job is
// idempotent and never stacks params).
func stampURL(raw string, u UTM, content string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(strings.ToLower(trimmed), "http") {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || !isKyouHost(parsed.Hostname()) {
		return ""
	}

	q := parsed.Query()
	if q.Get("utm_source") != "" {
		return ""
	}
	q.Set("utm_source", u.Source)
	q.Set("utm_medium", u.Medium)
	if u.Campaign != "" {
		q.Set("utm_campaign", u.Campaign)
	}
	if content != "" {
		q.Set("utm_content", content)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func isKyouHost(host string) bool {
	host = strings.ToLower(host)
	return host == "kyou.id" || strings.HasSuffix(host, ".kyou.id")
}
