package decision

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const ruleExt = ".rule"

// header is the subset of MADR's frontmatter the surfaces read. Keys it does
// not name are ignored rather than rejected, so a decision may carry the rest
// of MADR's optional metadata -- date, decision-makers, consulted, informed.
type header struct {
	Status string `yaml:"status"`
}

// parseFile reads one decision file. name is its path relative to the model
// directory, which is what Decision.Path reports and where the id comes from.
func parseFile(path, name string) (Decision, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Decision{}, err
	}
	front, body := splitFrontmatter(string(data))

	var h header
	if err := yaml.Unmarshal([]byte(front), &h); err != nil {
		return Decision{}, fmt.Errorf("%s: %w", name, err)
	}
	id := idFromName(name)
	if id == "" {
		return Decision{}, fmt.Errorf("%s: filename does not start with a MADR number (NNNN-title.md)", name)
	}
	title := titleFrom(body)
	if title == "" {
		return Decision{}, fmt.Errorf("%s: no title; MADR puts it in the document's first heading", name)
	}
	d := Decision{ID: id, Title: title, Status: h.Status, Body: body, Path: name}
	if err := d.readRule(path, name); err != nil {
		return Decision{}, err
	}
	return d, nil
}

// readRule picks up the ADE rule file named after the record. ADE and ADG both
// keep it beside the decision it enforces, so the pairing is the filename and
// needs nothing in the frontmatter to declare it.
func (d *Decision) readRule(path, name string) error {
	rulePath := strings.TrimSuffix(path, ".md") + ruleExt
	data, err := os.ReadFile(rulePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	d.Rule = string(data)
	d.RulePath = strings.TrimSuffix(name, ".md") + ruleExt
	return nil
}

// splitFrontmatter separates a leading YAML block fenced by --- from the body.
// MADR's frontmatter is optional, so a file that opens with no fence is all
// body rather than an error.
func splitFrontmatter(text string) (front, body string) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return "", strings.TrimSpace(text)
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			return strings.Join(lines[1:i], "\n"), strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
		}
	}
	// Unterminated fence: the file is a header and nothing else.
	return strings.Join(lines[1:], "\n"), ""
}

// titleFrom returns the text of the document's first level-one heading, which
// is where MADR keeps the title. The body keeps the heading: it is part of the
// decision as written, and a reader of the whole file expects it there.
func titleFrom(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "# "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// idFromName takes the digits MADR puts at the front of a filename.
func idFromName(name string) string {
	digits := name[:len(name)-len(strings.TrimLeft(name, "0123456789"))]
	return normalizeID(digits)
}

// normalizeID renders an id the one way every surface prints it: four digits,
// no prefix. A caller may address a decision as "1", "0001" or "ADR-0001", so
// both ends pass through here.
func normalizeID(raw string) string {
	id := strings.ToUpper(strings.TrimSpace(raw))
	for _, prefix := range []string{"ADR-", "ADR", "AD"} {
		if trimmed, ok := strings.CutPrefix(id, prefix); ok {
			id = trimmed
			break
		}
	}
	if id == "" || strings.TrimLeft(id, "0123456789") != "" {
		return ""
	}
	id = strings.TrimLeft(id, "0")
	if id == "" {
		id = "0"
	}
	if len(id) < 4 {
		id = strings.Repeat("0", 4-len(id)) + id
	}
	return id
}

// modelFiles lists the numbered files in dir: the decisions, and apart from
// them the rule files. MADR numbers every decision at the front of its name, so
// a file that does not start with a digit is something else kept alongside --
// a template, a README -- and is skipped.
func modelFiles(dir string) (decisions, rules []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.ContainsAny(name[:1], "0123456789") {
			continue
		}
		switch filepath.Ext(name) {
		case ".md":
			decisions = append(decisions, name)
		case ruleExt:
			rules = append(rules, name)
		}
	}
	return decisions, rules, nil
}
