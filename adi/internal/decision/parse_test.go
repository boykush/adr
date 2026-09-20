package decision

import "testing"

func TestNormalizeID(t *testing.T) {
	cases := map[string]string{
		"1":      "0001",
		"0001":   "0001",
		"AD0001": "0001",
		"ad1":    "0001",
		" 12 ":   "0012",
		"12345":  "12345",
		"0":      "0000",
		"":       "",
		// Not a number: left as written, so a malformed id reports as itself
		// rather than being silently rewritten into a plausible one.
		"draft": "DRAFT",
	}
	for in, want := range cases {
		if got := normalizeID(in); got != want {
			t.Errorf("normalizeID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitFrontmatter(t *testing.T) {
	front, body := splitFrontmatter("---\nadr_id: \"0001\"\n---\n\n# Title\n\nbody\n")
	if front != "adr_id: \"0001\"" {
		t.Errorf("front = %q", front)
	}
	if body != "# Title\n\nbody" {
		t.Errorf("body = %q", body)
	}

	front, body = splitFrontmatter("no fence here\n")
	if front != "" || body != "no fence here" {
		t.Errorf("unfenced: front = %q, body = %q", front, body)
	}

	front, body = splitFrontmatter("---\nadr_id: \"0001\"\n")
	if front != "adr_id: \"0001\"\n" || body != "" {
		t.Errorf("unterminated: front = %q, body = %q", front, body)
	}
}
