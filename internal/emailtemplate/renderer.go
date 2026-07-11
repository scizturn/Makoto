package emailtemplate

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	texttemplate "text/template"
)

type FileRenderer struct {
	Dir     string
	Subject string
}

func (r FileRenderer) Render(templateID string, mergeData map[string]any) (string, string, error) {
	if strings.TrimSpace(r.Dir) == "" {
		return "", "", fmt.Errorf("email template dir is empty")
	}
	if strings.TrimSpace(templateID) == "" {
		return "", "", fmt.Errorf("email template id is empty")
	}
	if !filepath.IsLocal(templateID) {
		return "", "", fmt.Errorf("email template id must be a local relative path")
	}

	data := viewDataFromMergeData(mergeData)
	path := filepath.Join(r.Dir, filepath.Clean(templateID))
	htmlTemplate, err := template.ParseFiles(path)
	if err != nil {
		return "", "", err
	}

	var htmlBody bytes.Buffer
	if err := htmlTemplate.Execute(&htmlBody, data); err != nil {
		return "", "", err
	}

	subject := r.Subject
	if strings.TrimSpace(subject) == "" {
		subject = "Selamat ulang tahun, {{ .Name }}"
	}
	subjectTemplate, err := texttemplate.New("subject").Parse(subject)
	if err != nil {
		return "", "", err
	}
	var subjectBody bytes.Buffer
	if err := subjectTemplate.Execute(&subjectBody, data); err != nil {
		return "", "", err
	}

	return subjectBody.String(), htmlBody.String(), nil
}

type viewData struct {
	Name               string
	FirstName          string
	VoucherCode        string
	WishlistHTML       template.HTML
	FYPHTML            template.HTML
	HistoricalHTML     template.HTML
	Years              string
	JoinDate           string
	AnniversaryEdition string
	ActionURL          string
	Closing            string
	// Leftover Cart fields
	Greeting   string
	CartHTML   template.HTML
	RecoHTML   template.HTML
	FooterHTML template.HTML
	CartURL    string
	// Discounted Wishlist fields
	FillHTML             template.HTML
	FeaturedHTML         template.HTML
	PromoHTML            template.HTML
	WishlistURL          string
	WishlistCount        string
	FillCount            string
	PromoCount           string
	DisplayWishlistCount string
	DisplayFillCount     string
	DiscountName         string
	BackInItemHTML       template.HTML
	CompanionHTML        template.HTML
	CompanionName        string
	RecoSeries           string
	HasCompanion         bool
	HasVoucher           bool
	// VoucherDiscount is the tier percent as text ("8"/"6"). Templates print it
	// next to a literal '%'. Never hardcode the number in the HTML.
	VoucherDiscount string
	// PO Ready fields
	OrderID         string
	ItemsHTML       template.HTML
	ItemCount       string
	BlastDate       string
	RemainingText   string
	DownPaymentText string
	ETA             string
}

func viewDataFromMergeData(data map[string]any) viewData {
	return viewData{
		Name:                 stringValue(data["name"]),
		FirstName:            stringValue(data["first_name"]),
		VoucherCode:          stringValue(data["voucher_code"]),
		WishlistHTML:         template.HTML(stringValue(data["wishlist_html"])),
		FYPHTML:              template.HTML(stringValue(data["fyp_html"])),
		HistoricalHTML:       template.HTML(stringValue(data["historical_html"])),
		Years:                stringValue(data["years"]),
		JoinDate:             stringValue(data["join_date"]),
		AnniversaryEdition:   stringValue(data["anniversary_edition"]),
		ActionURL:            stringValue(data["action_url"]),
		Closing:              stringValue(data["closing"]),
		Greeting:             stringValue(data["greeting"]),
		CartHTML:             template.HTML(stringValue(data["cart_html"])),
		RecoHTML:             template.HTML(stringValue(data["reco_html"])),
		FooterHTML:           template.HTML(stringValue(data["footer_html"])),
		CartURL:              stringValue(data["cart_url"]),
		FillHTML:             template.HTML(stringValue(data["fill_html"])),
		FeaturedHTML:         template.HTML(stringValue(data["featured_html"])),
		PromoHTML:            template.HTML(stringValue(data["promo_html"])),
		WishlistURL:          stringValue(data["wishlist_url"]),
		WishlistCount:        stringValue(data["wishlist_count"]),
		FillCount:            stringValue(data["fill_count"]),
		PromoCount:           stringValue(data["promo_count"]),
		DisplayWishlistCount: stringValue(data["display_wishlist_count"]),
		DisplayFillCount:     stringValue(data["display_fill_count"]),
		DiscountName:         stringValue(data["discount_name"]),
		BackInItemHTML:       template.HTML(stringValue(data["back_in_item_html"])),
		CompanionHTML:        template.HTML(stringValue(data["companion_html"])),
		CompanionName:        stringValue(data["companion_name"]),
		RecoSeries:           stringValue(data["reco_series"]),
		HasCompanion:         boolValue(data["has_companion"]),
		HasVoucher:           boolValue(data["has_voucher"]),
		VoucherDiscount:      stringValue(data["voucher_discount"]),
		OrderID:              stringValue(data["order_id"]),
		ItemsHTML:            template.HTML(stringValue(data["items_html"])),
		ItemCount:            stringValue(data["item_count"]),
		BlastDate:            stringValue(data["blast_date"]),
		RemainingText:        stringValue(data["remaining_text"]),
		DownPaymentText:      stringValue(data["down_payment_text"]),
		ETA:                  stringValue(data["eta"]),
	}
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}
