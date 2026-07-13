package email

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/kyou-id/makoto/internal/domain"
)

type captureSender struct {
	msg domain.EmailMessage
}

func (c *captureSender) SendTemplate(_ context.Context, msg domain.EmailMessage) (domain.SendResult, error) {
	c.msg = msg
	return domain.SendResult{MessageID: "captured"}, nil
}

func trackingSender(next Sender) Sender {
	return WithUTM(next, UTM{Source: "makoto", Medium: "email", Campaign: "po_ready"})
}

func hrefQuery(t *testing.T, body, prefix string) url.Values {
	t.Helper()
	start := strings.Index(body, `href="`+prefix)
	if start < 0 {
		t.Fatalf("no href starting with %q in %q", prefix, body)
	}
	rest := body[start+len(`href="`):]
	raw := rest[:strings.Index(rest, `"`)]
	parsed, err := url.Parse(strings.ReplaceAll(raw, "&amp;", "&"))
	if err != nil {
		t.Fatalf("parse href %q: %v", raw, err)
	}
	return parsed.Query()
}

func TestTrackingSenderStampsKyouLinks(t *testing.T) {
	capture := &captureSender{}
	msg := domain.EmailMessage{
		TemplateID: "po_ready2",
		HTMLBody:   `<a href="https://kyou.id/items/195113/">Nendoroid</a>`,
	}

	if _, err := trackingSender(capture).SendTemplate(context.Background(), msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	q := hrefQuery(t, capture.msg.HTMLBody, "https://kyou.id/items/")
	if got := q.Get("utm_source"); got != "makoto" {
		t.Errorf("utm_source = %q, want makoto", got)
	}
	if got := q.Get("utm_medium"); got != "email" {
		t.Errorf("utm_medium = %q, want email", got)
	}
	if got := q.Get("utm_campaign"); got != "po_ready" {
		t.Errorf("utm_campaign = %q, want po_ready", got)
	}
	if got := q.Get("utm_content"); got != "po_ready2" {
		t.Errorf("utm_content = %q, want po_ready2 (the template variant)", got)
	}
}

func TestTrackingSenderKeepsExistingQueryParams(t *testing.T) {
	capture := &captureSender{}
	msg := domain.EmailMessage{
		TemplateID: "leftover_cart1",
		HTMLBody:   `<a href="https://kyou.id/search?page=1%2C40&amp;sort=kyou_search_score&amp;sold=false">Cari</a>`,
	}

	if _, err := trackingSender(capture).SendTemplate(context.Background(), msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	body := capture.msg.HTMLBody
	q := hrefQuery(t, body, "https://kyou.id/search")
	if got := q.Get("sort"); got != "kyou_search_score" {
		t.Errorf("existing sort param lost: %q", got)
	}
	if got := q.Get("page"); got != "1,40" {
		t.Errorf("existing page param lost: %q", got)
	}
	if q.Get("utm_source") == "" {
		t.Error("utm_source not added to a URL that already had a query")
	}
	if strings.Contains(body, "?sort=") || !strings.Contains(body, "&amp;") {
		t.Errorf("separators must stay HTML-escaped inside the attribute: %q", body)
	}
}

func TestTrackingSenderLeavesForeignAndNonHTTPLinksAlone(t *testing.T) {
	capture := &captureSender{}
	msg := domain.EmailMessage{
		TemplateID: "birthday1",
		HTMLBody: `<a href="https://www.tiktok.com/@kyou.id?lang=en">TikTok</a>` +
			`<a href="mailto:nandayo@kyou.id">Balas</a>` +
			`<a href="{{ unsubscribe_link }}">Berhenti</a>`,
	}

	if _, err := trackingSender(capture).SendTemplate(context.Background(), msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	if got := capture.msg.HTMLBody; got != msg.HTMLBody {
		t.Errorf("non-kyou.id links must be untouched:\n got %q\nwant %q", got, msg.HTMLBody)
	}
}

func TestTrackingSenderIsIdempotent(t *testing.T) {
	capture := &captureSender{}
	once := domain.EmailMessage{
		TemplateID: "po_ready1",
		HTMLBody:   `<a href="https://kyou.id/user/wishlist?utm_source=makoto&amp;utm_medium=email">Wishlist</a>`,
	}

	if _, err := trackingSender(capture).SendTemplate(context.Background(), once); err != nil {
		t.Fatalf("send: %v", err)
	}

	if got := capture.msg.HTMLBody; got != once.HTMLBody {
		t.Errorf("a link that already carries utm_source must not be stamped twice:\n got %q\nwant %q", got, once.HTMLBody)
	}
}

func TestTrackingSenderStampsSubstitutionData(t *testing.T) {
	capture := &captureSender{}
	msg := domain.EmailMessage{
		TemplateID: "wishlist_back_in3",
		SubstitutionData: map[string]any{
			"action_url":    "https://kyou.id/user/my-voucher",
			"wishlist_html": `<a href="https://kyou.id/items/228691/"><img src="https://kyoucdn.id/x.webp"></a>`,
			"voucher_code":  "KYOU-1234",
			"item_count":    3,
		},
	}

	if _, err := trackingSender(capture).SendTemplate(context.Background(), msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	data := capture.msg.SubstitutionData
	action, err := url.Parse(data["action_url"].(string))
	if err != nil {
		t.Fatalf("parse action_url: %v", err)
	}
	if action.Query().Get("utm_campaign") != "po_ready" {
		t.Errorf("bare URL in substitution data not stamped: %v", data["action_url"])
	}
	if q := hrefQuery(t, data["wishlist_html"].(string), "https://kyou.id/items/"); q.Get("utm_source") != "makoto" {
		t.Errorf("pre-rendered grid HTML not stamped: %v", data["wishlist_html"])
	}
	if got := data["voucher_code"]; got != "KYOU-1234" {
		t.Errorf("non-URL string mangled: %v", got)
	}
	if got := data["item_count"]; got != 3 {
		t.Errorf("non-string value mangled: %v", got)
	}
}

func TestWithUTMWithoutSourceReturnsSenderUnwrapped(t *testing.T) {
	capture := &captureSender{}
	if got := WithUTM(capture, UTM{}); got != Sender(capture) {
		t.Error("an unconfigured UTM must leave the sender unwrapped, not stamp empty params")
	}
}
