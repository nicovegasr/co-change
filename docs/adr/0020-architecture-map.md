# ADR-0020 — Architecture map: imports and co-change between modules

- **Status:** accepted
- **Date:** 2026-11-08

## Context

The author's blog generates an architecture map from its own code on every
push to `main` and publishes it to GitHub Pages: folders with a README are
units, real imports are edges, and edges against the layer rules are flagged.
It is useful but tied to that repository: Astro layers, TS/JS imports only,
hard-coded units.

co-change already knows something an import graph cannot show: which parts of
a repository change together. Crossing both answers a question no diagram
generator does:

| Import | Change together | Reading |
|---|---|---|
| yes | yes | live dependency |
| yes | no | stable dependency |
| no | yes | **hidden coupling**: invisible in the code, forgotten in changes |

## Decision

- **`co-change map [--out DIR]`** writes a self-contained `DIR/index.html`
  (default `co-change-map/`) with the data embedded as JSON. No JS library
  and no build step; only web fonts load from Google Fonts, with system
  fallbacks, so the page also works offline.
- **Units** are folders with a `README.md`, as in the blog: its `# title` and
  first paragraph describe the unit. Code outside them is grouped by its
  top-level folder, or its child folder inside the usual containers (`src`,
  `internal`, `pkg`, `lib`, `app`, `apps`, `packages`, `modules`, `cmd`);
  loose files at the root or in a container are glue and belong to no unit.
  So the map works with zero setup and READMEs only make it better. A file
  belongs to its deepest unit. Hidden, dependency and build folders are
  skipped.
- **Import edges** come from small text readers, one per language, without
  any toolchain in CI:
  - TS/JS (and `.astro`, `.vue`, `.svelte`): relative specifiers and
    `tsconfig.json` `paths` aliases.
  - Go: imports under the `module` path of the root `go.mod`.
  - Java/Kotlin: `import a.b.C` resolves to the folders whose files declare
    `package a.b` (dropping trailing segments until one matches).
  - C#: `using A.B` resolves to the folders whose files declare
    `namespace A.B` (same prefix rule).
  - Python: absolute modules as paths from the root or `src/`, and relative
    `from .` imports.

  Readers are best effort: an unresolved import is ignored, never guessed.
  Package and namespace languages may point to several folders; all count.
- **Co-change edges** come from the index: non-merge commits that touch two
  units. The defaults are code-maat's (Tornhill), not ours: commits of at most
  30 files, at least 5 shared commits, and a degree of coupling (shared over
  the mean commits of both units) of at least 30%. A first run with the
  review's thresholds (3 shared, 50 files, ratio over the smaller unit)
  flagged 19 hidden couplings in this repository, nearly all from two
  45-file rename commits. An edge cites its three latest commits. Without an
  import in either direction it is **hidden**.
- **Layout:** units are stacked in rows by import depth (what nothing imports
  on top, foundations at the bottom), like the blog's layers but derived, not
  configured. Imports are solid arrows, co-change dashed; both can be toggled.
  The hidden couplings are also listed in a table below the diagram.
- **Delivery:** a composite action (`map/action.yml`) builds co-change, writes
  the page and uploads it as a Pages artifact; a reusable workflow
  (`.github/workflows/map.yml`) adds the deployment, so a repository adopts it
  with one `uses:` line. This repository publishes its own map.

Out of scope for v1: layer rules and violations (the blog's `DOCUMENTED_EXCEPTIONS`
need configuration), more languages (Rust, PHP, Ruby…), maps per PR.

## Consequences

- Any repository gets a map with no configuration; READMEs make it better.
- Text readers miss what needs a compiler (Go build tags, C# namespaces
  declared far from their folders, dynamic imports): the map can lack edges,
  never invent them.
- Unit-level co-change is coarse: squash-merged pull requests and phase
  commits touch many folders at once, which is why the map's thresholds are
  stricter than the review's. On small repositories the map may show few or
  no co-change edges; that is the honest result, not a bug.
- Pages must be enabled with "GitHub Actions" as the source; the workflow
  cannot do it for the user.

## References

- H. Gall, K. Hajek, M. Jazayeri, *Detection of Logical Coupling Based on
  Product Release History*, ICSM 1998.
- A. Tornhill, *Your Code as a Crime Scene* (2015) and *Software Design
  X-Rays* (2018), Pragmatic Bookshelf.
- [code-maat](https://github.com/adamtornhill/code-maat): source of the
  co-change defaults (`--min-revs 5`, `--min-shared-revs 5`,
  `--min-coupling 30`, `--max-changeset-size 30`) and of the degree of
  coupling (shared revisions over the average revisions of both modules,
  `logical_coupling.clj`). Only the numbers and the formula are reused, no
  code.
