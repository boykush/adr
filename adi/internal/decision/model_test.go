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

func decisionFile(id, status, title string) string {
	return "---\nadr_id: \"" + id + "\"\nstatus: " + status + "\ntitle: " + title + "\n---\n\n## Decision\n\nbody of " + id + "\n"
}

func TestLoadOrdersByIDAndReadsOnlyMarkdown(t *testing.T) {
	dir := writeModel(t, map[string]string{
		"AD0002-b.md":   decisionFile("0002", "open", "Second"),
		"AD0001-a.md":   decisionFile("0001", "decided", "First"),
		"AD0001-a.rule": "adr \"0001\" \"First\"\n",
		"index.yaml":    "decisions: {}\n",
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
	if decisions[0].Title != "First" || decisions[0].Status != "decided" {
		t.Errorf("first = %+v", decisions[0])
	}
	if decisions[0].Path != "AD0001-a.md" {
		t.Errorf("path = %q, want it relative to the model dir", decisions[0].Path)
	}
	if !strings.HasPrefix(decisions[0].Body, "## Decision") {
		t.Errorf("body = %q, want the frontmatter stripped", decisions[0].Body)
	}
}

func TestLoadRejectsDuplicateID(t *testing.T) {
	dir := writeModel(t, map[string]string{
		"AD0001-a.md": decisionFile("0001", "decided", "First"),
		"AD0001-b.md": decisionFile("AD1", "open", "Also first"),
	})

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "0001") {
		t.Fatalf("err = %v, want it to name the duplicated id", err)
	}
}

func TestLoadRejectsMissingID(t *testing.T) {
	dir := writeModel(t, map[string]string{"stray.md": "---\ntitle: No id\n---\n\nbody\n"})

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "adr_id") {
		t.Fatalf("err = %v, want it to name the missing field", err)
	}
}

func TestFindAcceptsEverySpelling(t *testing.T) {
	dir := writeModel(t, map[string]string{"AD0001-a.md": decisionFile("0001", "decided", "First")})

	for _, id := range []string{"1", "0001", "AD0001", "ad0001"} {
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
