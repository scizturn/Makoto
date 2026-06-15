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
	Greeting string
	CartHTML template.HTML
	RecoHTML template.HTML
	CartURL  string
}

func viewDataFromMergeData(data map[string]any) viewData {
	return viewData{
		Name:               stringValue(data["name"]),
		FirstName:          stringValue(data["first_name"]),
		VoucherCode:        stringValue(data["voucher_code"]),
		WishlistHTML:       template.HTML(stringValue(data["wishlist_html"])),
		FYPHTML:            template.HTML(stringValue(data["fyp_html"])),
		HistoricalHTML:     template.HTML(stringValue(data["historical_html"])),
		Years:              stringValue(data["years"]),
		JoinDate:           stringValue(data["join_date"]),
		AnniversaryEdition: stringValue(data["anniversary_edition"]),
		ActionURL:          stringValue(data["action_url"]),
		Closing:            stringValue(data["closing"]),
		Greeting:           stringValue(data["greeting"]),
		CartHTML:           template.HTML(stringValue(data["cart_html"])),
		RecoHTML:           template.HTML(stringValue(data["reco_html"])),
		CartURL:            stringValue(data["cart_url"]),
	}
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
