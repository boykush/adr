package decision

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// header is the subset of the YAML frontmatter the surfaces read. Keys it does
// not name are ignored rather than rejected, so a file may carry more than this
// -- ADG writes a comments list, and nothing here needs to know.
type header struct {
	ADRID  string `yaml:"adr_id"`
	Title  string `yaml:"title"`
	Status string `yaml:"status"`
}

// parseFile reads one decision file. name is its path relative to the model
// directory, which is what Decision.Path reports.
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
	if h.ADRID == "" {
		return Decision{}, fmt.Errorf("%s: frontmatter has no adr_id", name)
	}
	return Decision{
		ID:     normalizeID(h.ADRID),
		Title:  h.Title,
		Status: h.Status,
		Body:   body,
		Path:   name,
	}, nil
}

// splitFrontmatter separates a leading YAML block fenced by --- from the body.
// A file that opens with no fence is all body, which keeps a decision written
// by hand readable before it grows a header.
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

// normalizeID renders an id the one way every surface prints it: four digits,
// no prefix. Callers may address a decision as "1", "0001" or "AD0001", and a
// model file may quote its adr_id or not, so both ends pass through here.
func normalizeID(raw string) string {
	id := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(raw)), "AD")
	if id == "" || strings.TrimLeft(id, "0123456789") != "" {
		return id
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

// modelFiles lists the Markdown files in dir. ADG keeps an index.yaml and ADE a
// .rule beside them; both are derived from the decisions, so only .md is read.
func modelFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}
