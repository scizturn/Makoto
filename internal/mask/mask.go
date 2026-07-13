// Package mask hides recipient addresses before they reach a log line, Discord, or
// anywhere else a human might read them. Every service in this repo masks; this is
// the one implementation of it.
package mask

import "strings"

// Email keeps the first and last character of the local part: fajri@gmail.com
// becomes f***i@gmail.com. Enough to recognise an address you already know, not
// enough to harvest one you do not.
func Email(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return email
	}
	if len(local) == 1 {
		return local + "***@" + domain
	}
	return local[:1] + "***" + local[len(local)-1:] + "@" + domain
}
