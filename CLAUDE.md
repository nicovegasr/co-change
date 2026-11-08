# co-change — Claude Code project guide

`co-change` is a **CI pull-request impact review**: a GitHub Action
(`action.yml`) that runs `co-change review` and comments on the PR with the
untouched files that historically change with the ones it changes, backed by
evidence commits (ADR-0019). Deterministic first; an LLM layer is v2.

The original commit TUI (`co-change` with no arguments) still ships but is
**frozen**: no new features. Its design docs are `docs/tui.md`,
`docs/ux-mockups.html` and the "Original commit-TUI roadmap" in
`docs/ROADMAP.md`.

## Build & run

Go is the runtime. If `go` is missing: `brew install go`.

```sh
go mod tidy            # populate go.sum + transitive deps (needs network)
make tools             # install golangci-lint + lefthook, wire git hooks (once)
make build             # build ./co-change
make check             # fmt + vet + lint + test  (the pre-commit gate)
make cover             # per-package coverage gate ≥80% domain  (pre-push & CI)
```

The Makefile is the single source of truth: lefthook and CI call the same
targets (see `docs/contributing.md`, ADR-0003). Do not invent ad-hoc commands.

## Architecture

```
action.yml            composite GitHub Action: build, review, one sticky PR comment
main.go               entrypoint: CLI subcommands, or the TUI with no arguments
internal/cli          index / coupled / evaluate / review (review.go: ADR-0019)
internal/history      git history reader (merge base, changed and tracked files)
internal/index        bbolt index, co-change ranking, held-out evaluation
internal/pathignore   .cochangeignore / .auditignore patterns
internal/commit, git, audit, stats, settings, editor, ui   the frozen TUI
```

Domain packages split **pure logic** (tested) from a **thin shell-out seam**
(`var run = func(...)`, stubbed in tests). The TUI talks to them through
narrow injected interfaces (ADR-0005). See `docs/architecture.md`.

## Quality rules (harness)

- **TDD, pragmatic** (ADR-0002): logic written test-first (parsing, ranking,
  `Format`, model `Update→state`); raw render + git shell-out are not.
- **Coverage ≥80% per domain package** (`commit, audit, stats, git, settings, index, pathignore, editor`);
  `internal/ui` requires 0%. Gate: `make cover`.
- **Branch + PR**, never commit to `main`; CI (`make ci`) is the gate (ADR-0004).
- **Non-obvious design decision → write an ADR first** (`docs/adr/`, copy
  `0000-template.md`).

## Non-negotiable design rules

- **The PR review is informational**: it never fails a check or blocks a
  merge; only errors fail.
- **No suggestion without evidence**: every row cites commits, and evidence
  only comes from history before the merge base (the PR never confirms itself).
- **Do not cry wolf**: one comment per PR, updated in place; silent when there
  is nothing. Thresholds are flags; change defaults only with `evaluate` data.
- **Deterministic first.** An LLM may explain the evidence later (v2, opt-in,
  the user's own key), never invent suggestions.
- **Commit types are duplicated** from conventional-stats on purpose (TUI). Do
  not build a sync mechanism.
- **SemVer**, currently `0.1.0` (see `VERSION`).

## Conventions

- Commits: Conventional Commits in English (`feat:`, `fix:`, `chore:` …),
  trailing period in the subject (matches the tool's own `Format`).
- Prose to the user in Spanish; code/identifiers/commits in English. (This is
  why `misspell` is off in `.golangci.yml` — it only understands English.)
- Keep packages small; shell out to `git` rather than pulling a git library.
- **Good naming over comments.** Names carry intent.
- **Rule of three:** do not abstract until ~3 real repetitions justify it. No
  speculative interfaces or layers.
- **CQS:** a function either changes state (command) or returns data (query),
  not both.
- **Less code is better.** Prefer the smallest clear solution; cut boilerplate.
