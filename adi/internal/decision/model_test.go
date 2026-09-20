package decision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeModel(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func decisionFile(status, title string) string {
	return "---\nstatus: " + status + "\ndate: 2026-09-20\n---\n\n# " + title + "\n\n## Context and Problem Statement\n\nwhy\n"
}

func TestLoadOrdersByIDAndSkipsWhatIsNotADecision(t *testing.T) {
	dir := writeModel(t, map[string]string{
		"0002-second.md": decisionFile("proposed", "Second"),
		"0001-first.md":  decisionFile("accepted", "First"),
		// MADR numbers every decision, so an unnumbered file kept alongside is
		// something else and must not fail the load.
		"adr-template.md": "# Template\n",
		"README.md":       "# How to write these\n",
	})

	decisions, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 2 {
		t.Fatalf("got %d decisions, want 2", len(decisions))
	}
	if decisions[0].ID != "0001" || decisions[1].ID != "0002" {
		t.Errorf("ids = %q, %q", decisions[0].ID, decisions[1].ID)
	}
	if decisions[0].Title != "First" || decisions[0].Status != "accepted" {
		t.Errorf("first = %+v", decisions[0])
	}
	if decisions[0].Path != "0001-first.md" {
		t.Errorf("path = %q, want it relative to the model dir", decisions[0].Path)
	}
	if !strings.HasPrefix(decisions[0].Body, "# First") {
		t.Errorf("body = %q, want the frontmatter stripped and the heading kept", decisions[0].Body)
	}
}

func TestLoadRejectsDuplicateID(t *testing.T) {
	dir := writeModel(t, map[string]string{
		"0001-first.md":      decisionFile("accepted", "First"),
		"0001-also-first.md": decisionFile("proposed", "Also first"),
	})

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "0001") {
		t.Fatalf("err = %v, want it to name the duplicated id", err)
	}
}

func TestLoadRejectsMissingTitle(t *testing.T) {
	dir := writeModel(t, map[string]string{
		"0001-first.md": "---\nstatus: accepted\n---\n\n## Context and Problem Statement\n\nwhy\n",
	})

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("err = %v, want it to name the missing title", err)
	}
}

func TestFindAcceptsEverySpelling(t *testing.T) {
	dir := writeModel(t, map[string]string{"0001-first.md": decisionFile("accepted", "First")})

	for _, id := range []string{"1", "0001", "ADR-0001", "adr-0001"} {
		d, err := Find(dir, id)
		if err != nil {
			t.Fatalf("Find(%q): %v", id, err)
		}
		if d.ID != "0001" {
			t.Errorf("Find(%q).ID = %q", id, d.ID)
		}
	}

	if _, err := Find(dir, "99"); err == nil {
		t.Error("Find of an absent id returned no error")
	}
}

func TestLoadPicksUpTheRuleBesideTheRecord(t *testing.T) {
	dir := writeModel(t, map[string]string{
		"0001-first.md":   decisionFile("accepted", "First"),
		"0001-first.rule": "adr \"0001\" \"First\"\n",
		"0002-second.md":  decisionFile("accepted", "Second"),
	})

	decisions, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if decisions[0].RulePath != "0001-first.rule" || !strings.Contains(decisions[0].Rule, "adr \"0001\"") {
		t.Errorf("first rule = %q at %q", decisions[0].Rule, decisions[0].RulePath)
	}
	// A rule is optional, and its absence is the record that the decision has
	// no machine-readable form -- not a missing file.
	if decisions[1].Rule != "" || decisions[1].RulePath != "" {
		t.Errorf("second carries a rule it should not: %+v", decisions[1])
	}
	// The rule is not itself a decision.
	if len(decisions) != 2 {
		t.Errorf("got %d decisions, want 2", len(decisions))
	}
}
