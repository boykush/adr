package decision

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Load reads every decision in dir, ordered by id. A duplicate id fails the
// whole load rather than resolving to whichever file was read first: two
// decisions answering to one address is a broken model, not a lookup to guess.
// So does a rule with no decision beside it, which a rename leaves behind and
// which every session would otherwise stop receiving without a word.
func Load(dir string) ([]Decision, error) {
	names, rules, err := modelFiles(dir)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if record := strings.TrimSuffix(rule, ruleExt) + ".md"; !slices.Contains(names, record) {
			return nil, fmt.Errorf("rule %s has no decision %s beside it", rule, record)
		}
	}
	decisions := make([]Decision, 0, len(names))
	seen := make(map[string]string, len(names))
	for _, name := range names {
		d, err := parseFile(filepath.Join(dir, name), name)
		if err != nil {
			return nil, err
		}
		if first, dup := seen[d.ID]; dup {
			return nil, fmt.Errorf("adr_id %s is claimed by both %s and %s", d.ID, first, name)
		}
		seen[d.ID] = name
		decisions = append(decisions, d)
	}
	sort.Slice(decisions, func(i, j int) bool { return decisions[i].ID < decisions[j].ID })
	return decisions, nil
}

// Find returns the decision addressed by id, written any of "1", "0001" or
// "AD0001".
func Find(dir, id string) (Decision, error) {
	decisions, err := Load(dir)
	if err != nil {
		return Decision{}, err
	}
	want := normalizeID(id)
	for _, d := range decisions {
		if d.ID == want {
			return d, nil
		}
	}
	return Decision{}, fmt.Errorf("no decision ADR-%s in %s", want, dir)
}
