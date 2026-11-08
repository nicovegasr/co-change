package archmap

import (
	"reflect"
	"slices"
	"testing"

	"github.com/nicovegasr/co-change/internal/index"
)

// repo is an in-memory repository: path → content.
type repo map[string]string

func (r repo) build(sets ...index.ChangeSet) Map {
	files := make([]string, 0, len(r))
	for p := range r {
		files = append(files, p)
	}
	slices.Sort(files)
	return Build(files, func(p string) []byte { return []byte(r[p]) }, sets, Thresholds{MinSupport: 3, MinRatio: 0.3})
}

func unitIDs(m Map) []string {
	var ids []string
	for _, u := range m.Units {
		ids = append(ids, u.ID)
	}
	return ids
}

func importPairs(m Map) [][2]string {
	var pairs [][2]string
	for _, e := range m.Imports {
		pairs = append(pairs, [2]string{e.From, e.To})
	}
	return pairs
}

func TestUnitsAreFoldersWithReadme(t *testing.T) {
	m := repo{
		"README.md":                  "# root is never a unit",
		"src/api/README.md":          "# Public API\n\nServes the HTTP endpoints.\nOne per resource.\n\n## Inside\n- x\n",
		"src/api/server.ts":          "",
		"src/api/routes/README.md":   "# Routes",
		"src/api/routes/users.ts":    "",
		"node_modules/lib/README.md": "# skipped",
		".github/README.md":          "# hidden",
		"src/env.d.ts":               "",
		"src/pages/index.ts":         "",
	}.build()

	// src/pages has no README: the fallback groups it. A loose file in src/ is glue.
	if want := []string{"src/api", "src/api/routes", "src/pages"}; !reflect.DeepEqual(unitIDs(m), want) {
		t.Fatalf("units %v, want %v", unitIDs(m), want)
	}
	api := m.Units[0]
	if api.Title != "Public API" || api.Job != "Serves the HTTP endpoints. One per resource." || api.Files != 1 || api.Readme != "src/api/README.md" {
		t.Errorf("api unit %+v", api)
	}
}

func TestWithoutReadmesUnitsAreTopLevelCodeFolders(t *testing.T) {
	m := repo{
		"main.go":                 "",
		"internal/cli/cli.go":     "",
		"internal/index/store.go": "",
		"scripts/tune.py":         "",
		"docs/notes.md":           "",
		"vendor/x/x.go":           "",
	}.build()

	if want := []string{"internal/cli", "internal/index", "scripts"}; !reflect.DeepEqual(unitIDs(m), want) {
		t.Fatalf("units %v, want %v", unitIDs(m), want)
	}
	if m.Units[0].Title != "cli" {
		t.Errorf("fallback title %q", m.Units[0].Title)
	}
}

func TestImportReaders(t *testing.T) {
	for name, tc := range map[string]struct {
		files repo
		want  [][2]string
	}{
		"typescript relative, alias, dynamic and require": {repo{
			"tsconfig.json":   `{ "compilerOptions": { "baseUrl": ".", "paths": { "@/*": ["src/*"] } } }`,
			"src/a/README.md": "# A", "src/b/README.md": "# B", "src/c/README.md": "# C", "src/d/README.md": "# D",
			"src/a/index.ts": "import { b } from '../b/b';\nimport type { C } from \"@/c/c\";\nconst d = await import('../d/d.js');\nimport x from 'react';",
			"src/b/b.js":     "const c = require('../c');",
		}, [][2]string{{"src/a", "src/b"}, {"src/a", "src/c"}, {"src/a", "src/d"}, {"src/b", "src/c"}}},
		"go module imports": {repo{
			"go.mod":              "module github.com/x/app\n\ngo 1.26\n",
			"internal/cli/cli.go": "package cli\n\nimport (\n\t\"fmt\"\n\tidx \"github.com/x/app/internal/index\"\n)\n",
			"internal/index/s.go": "package index\n\nimport \"github.com/x/app/internal/store\"\n",
			"internal/store/s.go": "package store\n",
		}, [][2]string{{"internal/cli", "internal/index"}, {"internal/index", "internal/store"}}},
		"java and kotlin packages": {repo{
			"app/README.md": "# App", "domain/README.md": "# Domain",
			"app/src/Main.java":   "package com.acme.app;\n\nimport com.acme.domain.Order;\nimport static com.acme.domain.Rules.check;\nimport java.util.List;\n",
			"domain/src/Order.kt": "package com.acme.domain\n\nimport com.acme.app.*\n",
		}, [][2]string{{"app", "domain"}, {"domain", "app"}}},
		"csharp namespaces": {repo{
			"Api/README.md": "# Api", "Core/README.md": "# Core",
			"Api/Controller.cs":    "global using System;\nusing Acme.Core.Orders;\nusing Alias = Acme.Core.Orders.Order;\nnamespace Acme.Api;\n",
			"Core/Orders/Order.cs": "namespace Acme.Core.Orders\n{\n}\n",
		}, [][2]string{{"Api", "Core"}}},
		"python absolute and relative": {repo{
			"src/shop/README.md": "# Shop", "src/shop/cart/README.md": "# Cart", "src/billing/README.md": "# Billing",
			"src/shop/views.py":         "import os\nfrom billing.invoice import make\nfrom .cart import add\n",
			"src/shop/cart/__init__.py": "from ..views import index\n",
			"src/billing/invoice.py":    "",
		}, [][2]string{{"src/shop", "src/billing"}, {"src/shop", "src/shop/cart"}, {"src/shop/cart", "src/shop"}}},
	} {
		t.Run(name, func(t *testing.T) {
			if got := importPairs(tc.files.build()); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("imports %v, want %v", got, tc.want)
			}
		})
	}
}

func set(oid string, ts int64, paths ...string) index.ChangeSet {
	return index.ChangeSet{OID: oid, Timestamp: ts, Paths: paths}
}

func TestCoChangeBetweenUnitsFlagsHiddenCoupling(t *testing.T) {
	files := repo{
		"go.mod":    "module m\n",
		"api/a.go":  "package api\n\nimport \"m/core\"\n",
		"core/c.go": "package core\n",
		"docs/d.go": "package docs\n",
		"root.go":   "package main\n",
	}
	m := files.build(
		set("1", 1, "api/a.go", "core/c.go"),
		set("2", 2, "api/a.go", "core/c.go", "docs/d.go"),
		set("3", 3, "api/a.go", "core/c.go", "docs/d.go"),
		set("4", 4, "api/a.go", "docs/d.go", "root.go"),
		set("5", 5, "api/a.go"),
		set("6", 6, "core/c.go", "docs/d.go"),
	)

	var got []CoChange
	for _, c := range m.CoChanges {
		got = append(got, CoChange{A: c.A, B: c.B, Shared: c.Shared, Hidden: c.Hidden})
	}
	// api (5 commits) ↔ core (4): 3 shared, import → not hidden.
	// api ↔ docs (4) and core ↔ docs: 3 shared, no import → hidden.
	want := []CoChange{{A: "api", B: "core", Shared: 3}, {A: "api", B: "docs", Shared: 3, Hidden: true}, {A: "core", B: "docs", Shared: 3, Hidden: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("co-change %+v, want %+v", got, want)
	}
	if e := m.CoChanges[0].Evidence; len(e) != 3 || e[0].OID != "3" {
		t.Errorf("evidence must be the latest three commits first: %+v", e)
	}
	if want := 3 / 4.5; m.CoChanges[0].Ratio != want {
		t.Errorf("ratio over the mean of both units' commits: %v, want %v", m.CoChanges[0].Ratio, want)
	}
}

func TestCoChangeBelowThresholdsIsDropped(t *testing.T) {
	m := repo{"a/x.go": "", "b/y.go": "", "c/z.go": ""}.build(
		set("1", 1, "a/x.go", "b/y.go"), set("2", 2, "a/x.go", "b/y.go"), // only 2 shared
		set("3", 3, "a/x.go", "c/z.go"), set("4", 4, "a/x.go", "c/z.go"), set("5", 5, "a/x.go", "c/z.go"),
		set("6", 6, "c/z.go"), set("7", 7, "c/z.go"), set("8", 8, "c/z.go"), set("9", 9, "c/z.go"), set("10", 10, "c/z.go"),
		set("11", 11, "c/z.go"), set("12", 12, "c/z.go"), set("13", 13, "c/z.go"), set("14", 14, "c/z.go"), set("15", 15, "c/z.go"), set("16", 16, "c/z.go"),
	)
	// a↔c shares 3 over a mean of (5 + 14) / 2 commits: 0.32 passes.
	if len(m.CoChanges) != 1 || m.CoChanges[0].B != "c" {
		t.Fatalf("%+v", m.CoChanges)
	}
}

func TestRanksStackUnitsByImportDepth(t *testing.T) {
	m := repo{
		"go.mod":    "module m\n",
		"web/m.go":  "package main\n\nimport (\n\t\"m/api\"\n\t\"m/core\"\n)\n",
		"api/a.go":  "package api\n\nimport \"m/core\"\n",
		"core/c.go": "package core\n",
		"lone/l.go": "package lone\n",
		"ping/p.go": "package ping\n\nimport \"m/pong\"\n", // a cycle must not loop
		"pong/p.go": "package pong\n\nimport \"m/ping\"\n",
	}.build()
	ranks := map[string]int{}
	for _, u := range m.Units {
		ranks[u.ID] = u.Rank
	}
	if ranks["lone"] != 0 || ranks["core"] != 0 || ranks["api"] != 1 || ranks["web"] != 2 {
		t.Fatalf("ranks %v", ranks)
	}
}
