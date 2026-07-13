package mask

import "testing"

func TestEmailKeepsOnlyTheEdgesOfTheLocalPart(t *testing.T) {
	cases := map[string]string{
		"fajri@gmail.com":  "f***i@gmail.com",
		"a@kyou.id":        "a***@kyou.id",
		"bimo@kyou.id":     "b***o@kyou.id",
		"fajri.@gmail.com": "f***.@gmail.com",
	}
	for address, want := range cases {
		if got := Email(address); got != want {
			t.Fatalf("Email(%q) = %q, want %q", address, got, want)
		}
	}
}
