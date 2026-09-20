package decision

import (
	"fmt"
	"path/filepath"
	"sort"
)

// Load reads every decision in dir, ordered by id. A duplicate id fails the
// whole load rather than resolving to whichever file was read first: two
// decisions answering to one address is a broken model, not a lookup to guess.
func Load(dir string) ([]Decision, error) {
	names, err := modelFiles(dir)
	if err != nil {
		return nil, err
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
	return Decision{}, fmt.Errorf("no decision AD%s in %s", want, dir)
}
