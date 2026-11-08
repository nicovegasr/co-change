package archmap

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// Each reader turns one file's imports into repository paths (files or
// folders). Unresolved imports are dropped, never guessed (ADR-0020).
type reader func(file, text string) []string

var languages = map[string]string{
	".ts": "js", ".tsx": "js", ".mts": "js", ".cts": "js", ".js": "js", ".jsx": "js", ".mjs": "js", ".cjs": "js",
	".astro": "js", ".vue": "js", ".svelte": "js",
	".go": "go", ".java": "jvm", ".kt": "jvm", ".kts": "jvm", ".cs": "cs", ".py": "py",
}

func isCode(p string) bool { return languages[path.Ext(p)] != "" }

var (
	jsImport     = regexp.MustCompile(`(?:import|export)\s+(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]|(?:import|require)\(\s*['"]([^'"]+)['"]\s*\)`)
	tsBaseURL    = regexp.MustCompile(`"baseUrl"\s*:\s*"([^"]*)"`)
	tsAlias      = regexp.MustCompile(`"([^"*]*)\*"\s*:\s*\[\s*"([^"*]*)\*"`)
	goModule     = regexp.MustCompile(`(?m)^module\s+(\S+)`)
	goImportLine = regexp.MustCompile(`(?m)^\s*import\s+(?:[\w.]+\s+)?"([^"]+)"`)
	goImportList = regexp.MustCompile(`(?s)\bimport\s*\((.*?)\)`)
	quoted       = regexp.MustCompile(`"([^"]+)"`)
	jvmPackage   = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)`)
	jvmImport    = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([\w.]+?)(?:\.\*)?\s*(?:;|\s+as\s+\w+)?\s*$`)
	csNamespace  = regexp.MustCompile(`(?m)^\s*namespace\s+([\w.]+)`)
	csUsing      = regexp.MustCompile(`(?m)^\s*(?:global\s+)?using\s+(?:static\s+)?(?:\w+\s*=\s*)?([\w.]+)\s*;`)
	pyFrom       = regexp.MustCompile(`(?m)^\s*from\s+(\.*)([\w.]*)\s+import\s+\(?([\w, ]+)`)
	pyImport     = regexp.MustCompile(`(?m)^\s*import\s+([\w.]+(?:\s*,\s*[\w.]+)*)`)
)

// readImports reads every code file and returns the unit-level imports,
// sorted and without self-imports.
func readImports(files []string, read func(string) []byte, owner func(string) string) []Import {
	readers := newReaders(files, read)
	seen := map[Import]bool{}
	for _, f := range files {
		from := owner(f)
		r := readers[languages[path.Ext(f)]]
		if from == "" || r == nil || skipped(f) {
			continue
		}
		for _, target := range r(f, string(read(f))) {
			if to := owner(target); to != "" && to != from {
				seen[Import{From: from, To: to}] = true
			}
		}
	}
	imports := make([]Import, 0, len(seen))
	for e := range seen {
		imports = append(imports, e)
	}
	slices.SortFunc(imports, func(a, b Import) int {
		if a.From != b.From {
			return strings.Compare(a.From, b.From)
		}
		return strings.Compare(a.To, b.To)
	})
	return imports
}

// newReaders prepares what the readers need from the whole repository: the
// tsconfig aliases, the Go module and the JVM/C# declarations.
func newReaders(files []string, read func(string) []byte) map[string]reader {
	exists := map[string]bool{}
	packages, namespaces := map[string][]string{}, map[string][]string{}
	for _, f := range files {
		for dir := f; dir != "."; dir = path.Dir(dir) {
			exists[dir] = true
		}
		switch languages[path.Ext(f)] {
		case "jvm":
			declare(packages, jvmPackage, f, read)
		case "cs":
			declare(namespaces, csNamespace, f, read)
		}
	}
	return map[string]reader{
		"js":  jsReader(files, read),
		"go":  goReader(files, read),
		"jvm": declaredReader(jvmImport, packages),
		"cs":  declaredReader(csUsing, namespaces),
		"py":  pyReader(exists),
	}
}

func declare(declared map[string][]string, pattern *regexp.Regexp, file string, read func(string) []byte) {
	for _, m := range pattern.FindAllStringSubmatch(string(read(file)), -1) {
		if dir := path.Dir(file); !slices.Contains(declared[m[1]], dir) {
			declared[m[1]] = append(declared[m[1]], dir)
		}
	}
}

func jsReader(files []string, read func(string) []byte) reader {
	aliases := map[string]string{}
	for _, config := range []string{"tsconfig.json", "jsconfig.json"} {
		if !slices.Contains(files, config) {
			continue
		}
		text := string(read(config))
		base := "."
		if m := tsBaseURL.FindStringSubmatch(text); m != nil {
			base = m[1]
		}
		for _, m := range tsAlias.FindAllStringSubmatch(text, -1) {
			aliases[m[1]] = path.Join(base, m[2])
		}
	}
	return func(file, text string) []string {
		var targets []string
		for _, m := range jsImport.FindAllStringSubmatch(text, -1) {
			specifier, _, _ := strings.Cut(m[1]+m[2], "?")
			if strings.HasPrefix(specifier, ".") {
				targets = append(targets, path.Join(path.Dir(file), specifier))
				continue
			}
			for prefix, target := range aliases {
				if prefix != "" && strings.HasPrefix(specifier, prefix) {
					targets = append(targets, path.Join(target, strings.TrimPrefix(specifier, prefix)))
				}
			}
		}
		return targets
	}
}

func goReader(files []string, read func(string) []byte) reader {
	module := ""
	if slices.Contains(files, "go.mod") {
		if m := goModule.FindStringSubmatch(string(read("go.mod"))); m != nil {
			module = m[1]
		}
	}
	return func(_, text string) []string {
		var imports []string
		for _, m := range goImportLine.FindAllStringSubmatch(text, -1) {
			imports = append(imports, m[1])
		}
		for _, block := range goImportList.FindAllStringSubmatch(text, -1) {
			for _, m := range quoted.FindAllStringSubmatch(block[1], -1) {
				imports = append(imports, m[1])
			}
		}
		var targets []string
		for _, imp := range imports {
			if module != "" && strings.HasPrefix(imp, module+"/") {
				targets = append(targets, strings.TrimPrefix(imp, module+"/"))
			}
		}
		return targets
	}
}

// declaredReader resolves dotted names (Java packages, C# namespaces) to the
// folders declaring them, dropping trailing segments (a class, a member)
// until a declared name matches.
func declaredReader(pattern *regexp.Regexp, declared map[string][]string) reader {
	return func(_, text string) []string {
		var targets []string
		for _, m := range pattern.FindAllStringSubmatch(text, -1) {
			for name := m[1]; name != ""; name = trimLastSegment(name) {
				if dirs, ok := declared[name]; ok {
					targets = append(targets, dirs...)
					break
				}
			}
		}
		return targets
	}
}

func trimLastSegment(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[:i]
	}
	return ""
}

// pyReader resolves absolute modules from the root or src/, and relative ones
// from the importing file's package.
func pyReader(exists map[string]bool) reader {
	found := func(candidate string) bool { return exists[candidate] || exists[candidate+".py"] }
	return func(file, text string) []string {
		var targets []string
		resolve := func(dots, module string) {
			name := strings.ReplaceAll(module, ".", "/")
			if dots != "" {
				base := path.Dir(file)
				for range len(dots) - 1 {
					base = path.Dir(base)
				}
				targets = append(targets, path.Join(base, name))
				return
			}
			for _, candidate := range []string{name, path.Join("src", name)} {
				if found(candidate) {
					targets = append(targets, candidate)
					return
				}
			}
		}
		for _, m := range pyFrom.FindAllStringSubmatch(text, -1) {
			if m[2] != "" {
				resolve(m[1], m[2])
				continue
			}
			for name := range strings.SplitSeq(m[3], ",") {
				resolve(m[1], strings.TrimSpace(name))
			}
		}
		for _, m := range pyImport.FindAllStringSubmatch(text, -1) {
			for module := range strings.SplitSeq(m[1], ",") {
				resolve("", strings.TrimSpace(module))
			}
		}
		return targets
	}
}
