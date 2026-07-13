package webhook

import "strings"

// Job IDs are built by Yukari as "<campaign>-<date>-user-<id>", e.g.
// "discounted-wishlist-2026-07-10-user-113164", and by forcejob as
// "force-<campaign>-<date>-<time>-user-<id>". They are the only thing a delivery
// webhook carries about the email, so everything a human wants to read — which
// campaign, which user — is parsed back out of them here.
//
// Longest slug first: "wishlist-back-in" must be tried before any shorter slug it
// could be confused with, and "po-ready" before "po".
var campaignTitles = []struct{ slug, title string }{
	{"discounted-wishlist", "Discounted Wishlist"},
	{"wishlist-back-in", "Wishlist Back In"},
	{"leftover-cart", "Leftover Cart"},
	{"anniversary", "Anniversary"},
	{"po-ready", "PO Ready"},
	{"birthday", "Birthday"},
	{"winback", "Winback"},
}

// JobFacts is what a job ID can tell us about the email that produced an event.
type JobFacts struct {
	Campaign string // human title, e.g. "PO Ready"; empty when the ID is unrecognised
	UserID   string
	Forced   bool // sent by cmd/forcejob rather than a cron run
}

func ParseJobID(jobID string) JobFacts {
	facts := JobFacts{}
	if jobID == "" {
		return facts
	}

	rest := jobID
	if trimmed, ok := strings.CutPrefix(rest, "force-"); ok {
		facts.Forced = true
		rest = trimmed
	}

	for _, campaign := range campaignTitles {
		if strings.HasPrefix(rest, campaign.slug) {
			facts.Campaign = campaign.title
			break
		}
	}

	if _, userID, ok := strings.Cut(jobID, "-user-"); ok {
		facts.UserID = userID
	}

	return facts
}
