// Package archmap builds the architecture map (ADR-0020): units of a
// repository, the imports between them and the co-changes no import explains.
package archmap

import (
	"cmp"
	"path"
	"slices"
	"strings"

	"github.com/nicovegasr/co-change/internal/index"
)

// Unit is a folder of the repository shown as one box.
type Unit struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Job    string `json:"job,omitempty"`
	Readme string `json:"readme,omitempty"`
	Files  int    `json:"files"`
	Rank   int    `json:"rank"`
}

// Import means some file of From imports some file of To.
type Import struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// CoChange means A and B changed together in Shared eligible commits. Ratio
// is Shared over the mean of both units' commits (code-maat's degree of
// coupling). Hidden when no import links them in either direction.
type CoChange struct {
	A        string           `json:"a"`
	B        string           `json:"b"`
	Shared   int              `json:"shared"`
	Ratio    float64          `json:"ratio"`
	Hidden   bool             `json:"hidden"`
	Evidence []index.Evidence `json:"evidence"`
}

// Map is everything the page draws.
type Map struct {
	Units     []Unit     `json:"units"`
	Imports   []Import   `json:"imports"`
	CoChanges []CoChange `json:"cochanges"`
}

// Thresholds keep weak co-changes off the map (ADR-0020).
type Thresholds struct {
	MinSupport int
	MinRatio   float64
}

// Build maps the repository's tracked files, read reads one of them, and sets
// are its eligible commits.
func Build(files []string, read func(string) []byte, sets []index.ChangeSet, t Thresholds) Map {
	units := discoverUnits(files, read)
	ids := make([]string, len(units))
	for i, u := range units {
		ids[i] = u.ID
	}
	owner := func(p string) string { return unitOf(ids, p) }
	for _, f := range files {
		if i := slices.Index(ids, owner(f)); i >= 0 && path.Base(f) != "README.md" {
			units[i].Files++
		}
	}
	imports := readImports(files, read, owner)
	rank(units, imports)
	return Map{Units: units, Imports: imports, CoChanges: coChanges(sets, owner, imports, t)}
}

// unitOf is the deepest unit containing p, or "" outside every unit.
func unitOf(ids []string, p string) string {
	best := ""
	for _, id := range ids {
		if (p == id || strings.HasPrefix(p, id+"/")) && len(id) > len(best) {
			best = id
		}
	}
	return best
}

var skippedDirs = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true, "target": true,
	"bin": true, "obj": true, "out": true, "coverage": true, "__pycache__": true,
}

// containers hold modules rather than being one.
var containers = map[string]bool{
	"src": true, "internal": true, "pkg": true, "lib": true, "app": true,
	"apps": true, "packages": true, "modules": true, "cmd": true,
}

func skipped(p string) bool {
	for segment := range strings.SplitSeq(path.Dir(p), "/") {
		if skippedDirs[segment] || strings.HasPrefix(segment, ".") && segment != "." {
			return true
		}
	}
	return false
}

// discoverUnits takes folders with a README.md. Code outside them is grouped
// by top-level folder, or by child folder inside containers like src/.
func discoverUnits(files []string, read func(string) []byte) []Unit {
	var units []Unit
	var ids []string
	for _, f := range files {
		if path.Base(f) == "README.md" && path.Dir(f) != "." && !skipped(f) {
			title, job := describe(string(read(f)))
			units = append(units, Unit{ID: path.Dir(f), Title: cmp.Or(title, path.Base(path.Dir(f))), Job: job, Readme: f})
			ids = append(ids, path.Dir(f))
		}
	}
	readmes := slices.Clone(ids)
	for _, f := range files {
		segments := strings.Split(f, "/")
		// Root files and loose files in a container are glue, not a module.
		if !isCode(f) || skipped(f) || len(segments) < 2 || containers[segments[0]] && len(segments) < 3 || unitOf(readmes, f) != "" {
			continue
		}
		id := segments[0]
		if containers[id] {
			id = path.Join(segments[0], segments[1])
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
			units = append(units, Unit{ID: id, Title: path.Base(id)})
		}
	}
	slices.SortFunc(units, func(a, b Unit) int { return strings.Compare(a.ID, b.ID) })
	return units
}

// describe reads a README's "# title" and the first paragraph after it.
func describe(readme string) (title, job string) {
	for block := range strings.SplitSeq(strings.ReplaceAll(readme, "\r\n", "\n"), "\n\n") {
		block = strings.TrimSpace(block)
		switch {
		case title == "" && strings.HasPrefix(block, "# "):
			first, rest, _ := strings.Cut(block, "\n")
			title = strings.TrimSpace(first[2:])
			if rest = strings.TrimSpace(rest); rest != "" && !strings.HasPrefix(rest, "#") {
				return title, strings.Join(strings.Fields(rest), " ")
			}
		case title != "" && block != "" && !strings.HasPrefix(block, "#"):
			return title, strings.Join(strings.Fields(block), " ")
		}
	}
	return title, ""
}

// rank stacks units by import depth: 0 imports no other unit, otherwise one
// more than the deepest unit it imports. Imports closing a cycle are ignored.
func rank(units []Unit, imports []Import) {
	targets := map[string][]string{}
	for _, e := range imports {
		targets[e.From] = append(targets[e.From], e.To)
	}
	ranks := map[string]int{}
	visiting := map[string]bool{}
	var depth func(string) int
	depth = func(id string) int {
		if r, ok := ranks[id]; ok {
			return r
		}
		visiting[id] = true
		r := 0
		for _, to := range targets[id] {
			if !visiting[to] {
				r = max(r, depth(to)+1)
			}
		}
		visiting[id] = false
		ranks[id] = r
		return r
	}
	for i := range units {
		units[i].Rank = depth(units[i].ID)
	}
}

const evidencePerEdge = 3

// coChanges counts the eligible commits each pair of units shared.
func coChanges(sets []index.ChangeSet, owner func(string) string, imports []Import, t Thresholds) []CoChange {
	commits := map[string]int{}
	pairs := map[[2]string]*CoChange{}
	for _, s := range sets {
		var touched []string
		for _, p := range s.Paths {
			if u := owner(p); u != "" && !slices.Contains(touched, u) {
				touched = append(touched, u)
			}
		}
		slices.Sort(touched)
		for i, a := range touched {
			commits[a]++
			for _, b := range touched[i+1:] {
				pair := pairs[[2]string{a, b}]
				if pair == nil {
					pair = &CoChange{A: a, B: b}
					pairs[[2]string{a, b}] = pair
				}
				pair.Shared++
				pair.Evidence = append(pair.Evidence, index.Evidence{OID: s.OID, Timestamp: s.Timestamp, Subject: s.Subject})
			}
		}
	}
	linked := map[[2]string]bool{}
	for _, e := range imports {
		linked[[2]string{e.From, e.To}], linked[[2]string{e.To, e.From}] = true, true
	}
	var kept []CoChange
	for key, pair := range pairs {
		pair.Ratio = float64(pair.Shared) / (float64(commits[pair.A]+commits[pair.B]) / 2)
		if pair.Shared < t.MinSupport || pair.Ratio < t.MinRatio {
			continue
		}
		pair.Hidden = !linked[key]
		slices.SortFunc(pair.Evidence, func(a, b index.Evidence) int {
			return cmp.Or(cmp.Compare(b.Timestamp, a.Timestamp), strings.Compare(b.OID, a.OID))
		})
		pair.Evidence = pair.Evidence[:min(len(pair.Evidence), evidencePerEdge)]
		kept = append(kept, *pair)
	}
	slices.SortFunc(kept, func(a, b CoChange) int { return cmp.Or(strings.Compare(a.A, b.A), strings.Compare(a.B, b.B)) })
	return kept
}
