# ADR-0019 — Refocus on CI: co-change PR impact review

- **Status:** accepted
- **Date:** 2026-11-08

## Context

The commit TUI (Phases 0–6) works, but its author does not use it day to day.
The co-change analysis behind it (ADR-0008..0014) is the valuable part: it is
validated on seven repositories and already runs headless (`index`, `coupled`,
`evaluate`). A pull request is where "you changed X; Y usually changes with X
and is untouched" is most useful: reviewers see it before merging, and no one
has to keep a tool open.

The name `nib` says nothing about what the tool does, and neither does the
repository name `conventional-commit-tui` for a CI product.

## Decision

- **Product:** a pull-request impact reviewer, shipped as a GitHub Action. The
  TUI stays in the binary as a secondary feature and gets no new work.
- **Name:** binary, repository, Go module and Action are renamed to
  `co-change`; `.nibignore` becomes `.cochangeignore`. There are no external
  users, so there is no compatibility shim.
- **`co-change review --base <ref> [--json]`:**
  - PR files are `git diff --name-only <merge-base>...HEAD`.
  - Evidence only comes from history reachable from the merge base, so the
    PR's own commits never confirm its suggestions.
  - Suggestions are related paths (ADR-0014 aggregation, default ranking)
    that the PR does not touch, each with the PR file it relates to most
    strongly, its ratio, shared commits and up to three evidence commits.
  - Default output is the Markdown body of the PR comment; `--json` for
    other consumers.
  - It is informational: it exits 0 whatever it finds; only errors fail.
- **Noise filter:** a comment must not cry wolf. `review` shows at most 5
  suggestions with at least 3 shared commits and a co-change ratio of at least
  0.3. These thresholds are provisional product choices, not validated; they
  are flags, and measuring them with `evaluate` is follow-up work.
- **Action:** a composite `action.yml` at the repository root. It needs
  `fetch-depth: 0` and `pull-requests: write`, builds the binary with
  `setup-go`, runs `index` then `review`, and creates or updates one comment
  marked `<!-- co-change -->`. With no suggestions it updates an existing
  comment to say so and creates none.
- **Dogfooding:** this repository runs the Action on its own PRs.

Out of scope for v1: an LLM layer (v2, optional, the user's own API key, only
explaining the deterministic evidence), failing checks, platforms other than
GitHub, prebuilt binaries and index caching (after ingestion is measured,
roadmap milestone 2).

## Consequences

- The deterministic evidence is the product: every suggestion links to the
  commits that justify it, which separates it from generic AI reviewers.
- Shallow clones cannot work; the Action fails with a clear message instead.
- Indexing from scratch on every run costs time on large repositories until
  caching lands.
- The TUI, `/audit`, `/stats` and `/changes` keep working but are frozen; the
  roadmap and README lead with the CI product.
