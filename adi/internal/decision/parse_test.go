package decision

import "testing"

func TestNormalizeID(t *testing.T) {
	cases := map[string]string{
		"1":        "0001",
		"0001":     "0001",
		"ADR-0001": "0001",
		"adr-1":    "0001",
		"AD0001":   "0001",
		" 12 ":     "0012",
		"12345":    "12345",
		"0":        "0000",
		// Not a number: no id at all, so a malformed reference fails to resolve
		// rather than being rewritten into a plausible one.
		"":      "",
		"draft": "",
	}
	for in, want := range cases {
		if got := normalizeID(in); got != want {
			t.Errorf("normalizeID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIDFromName(t *testing.T) {
	cases := map[string]string{
		"0001-choose-the-instruction-file.md": "0001",
		"12-short.md":                         "0012",
		"template.md":                         "",
	}
	for in, want := range cases {
		if got := idFromName(in); got != want {
			t.Errorf("idFromName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTitleFrom(t *testing.T) {
	if got := titleFrom("# Choose the instruction file\n\n## Context\n"); got != "Choose the instruction file" {
		t.Errorf("titleFrom = %q", got)
	}
	// A level-two heading is a section, not the title.
	if got := titleFrom("## Context and Problem Statement\n"); got != "" {
		t.Errorf("titleFrom of a body with no h1 = %q, want empty", got)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	front, body := splitFrontmatter("---\nstatus: accepted\n---\n\n# Title\n\nbody\n")
	if front != "status: accepted" {
		t.Errorf("front = %q", front)
	}
	// The heading stays in the body: it is part of the decision as written.
	if body != "# Title\n\nbody" {
		t.Errorf("body = %q", body)
	}

	// MADR's frontmatter is optional.
	front, body = splitFrontmatter("# Title\n")
	if front != "" || body != "# Title" {
		t.Errorf("unfenced: front = %q, body = %q", front, body)
	}

	front, body = splitFrontmatter("---\nstatus: accepted\n")
	if front != "status: accepted\n" || body != "" {
		t.Errorf("unterminated: front = %q, body = %q", front, body)
	}
}
