package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The skill reaches this repo from ai-plugins the way it reaches every other
// consumer, so the copy apm deployed here is the one to hold to go.mod.
const skillReferences = "../../.claude/skills/ade-rule-dsl/references"

// TestSkillCarriesWhatVendorWrites fails once go.mod moves ADE to a version
// the ade-rule-dsl skill's reference was not taken from.
func TestSkillCarriesWhatVendorWrites(t *testing.T) {
	dir := t.TempDir()
	if err := vendor(dir); err != nil {
		t.Fatal(err)
	}
	version, err := adeVersion()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{referenceFile, "LICENSE"} {
		want, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(skillReferences, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("the ade-rule-dsl skill's %s is not ADE %s's: vendor it into ai-plugins, then move the pin in apm.yml (README: 書く)", name, version)
		}
	}
}

func TestValidateNamesEachFileThatDoesNotParse(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "0001-good.rule")
	bad := filepath.Join(dir, "0002-bad.rule")
	writeFile(t, good, "adr \"0001\" \"Good\"\n\npath \"Readme\" = \"README.md\"\n\nfile \"readme\" {\n  Readme must exist\n}\n")
	// ADE requires every rule file to open with its adr declaration.
	writeFile(t, bad, "file \"readme\" {\n  path \"README.md\" must exist\n}\n")

	var stdout, stderr bytes.Buffer
	if err := validate([]string{dir}, &stdout, &stderr); err == nil {
		t.Fatal("validate passed a rule file with no adr declaration")
	}
	if !strings.Contains(stderr.String(), bad) || strings.Contains(stderr.String(), good) {
		t.Errorf("stderr = %q, want it to name %s alone", stderr.String(), bad)
	}
	if !strings.Contains(stdout.String(), good) {
		t.Errorf("stdout = %q, want it to name %s", stdout.String(), good)
	}
}

func TestValidateRejectsADirectoryWithNoRules(t *testing.T) {
	if err := validate([]string{t.TempDir()}, io.Discard, io.Discard); err == nil {
		t.Error("validate passed a directory holding no .rule files")
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
