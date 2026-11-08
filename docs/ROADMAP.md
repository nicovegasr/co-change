# co-change — roadmap & design decisions

## Current direction — CI pull-request impact review (ADR-0019)

`co-change` is a GitHub Action: on every pull request it comments with the
untouched files that historically change with the ones the PR changes, each
backed by evidence commits. Deterministic first, informational only.

- [x] **v0.1 — review + Action.** `co-change review --base <ref>`, composite
  `action.yml`, sticky PR comment, dogfooded on this repository.
- [x] **v0.2 — architecture map** (ADR-0020). `co-change map`, `map/action.yml`
  and the reusable Pages workflow; imports for TS/JS, Go, Java/Kotlin, C#,
  Python.
- [ ] **Map: more languages and layer rules.** Rust, PHP, Ruby readers;
  optional configured layers that flag imports against the direction (the
  blog's violations).
- [ ] **Validate the comment thresholds.** Measure `min-support 3` /
  `min-ratio 0.3` with `evaluate` (precision of what the comment would show,
  and how often it stays silent) on the benchmark repositories.
- [ ] **Speed on large repositories.** Cache the index between runs
  (`actions/cache` keyed by base) and ship prebuilt binaries instead of
  building from source; needs ingestion measured first (milestone 2 below).
- [ ] **Publish.** Tag `v1`, list the Action on the GitHub Marketplace.
- [ ] **v2 — agent layer (opt-in).** An LLM, with the user's own API key,
  explains the evidence in the PR's context; it never invents suggestions.
- [ ] **Beyond files.** Ownership ("who usually reviews these files") and
  symbol-level relationships, once their semantics are decided.

## Analysis groundwork

The analysis the review is built on (ADR-0008..0014):

0. ~~Coupling in the commit TUI~~ — related-files panel over the git log,
   background index refresh and 2 s live reload (ADR-0014).
1. ~~Reproducible held-out evaluation~~ — `nib evaluate` (ADR-0009). First
   preregistered run on three repositories: source conditioning beats global
   frequency, but Nib's default filters lost to unfiltered co-change. Tuning on
   older windows and validating on newer ones (ADR-0010) replaced the defaults
   with support 1 and recency ranking (90-day half-life). ADR-0011 replicated
   that choice on four new repositories (Flask, ripgrep, OkHttp, Django).
   ADR-0012 fixed the case population: `--case-max-files` is separate from the
   predictor's `--max-files`, and non-merge commits are sampled from the whole
   DAG (Flask: 67 cases instead of 2). Earlier numbers are not comparable.
   ADR-0013 re-ran the replication under it: the defaults hold (better than
   the previous ones on six of seven repositories), though their margin over
   unfiltered co-change on validation is under a point.
2. Measure real repositories at increasing sizes and profile ingestion/query costs.
   Initial Astro smoke completed; repeat under controlled conditions and include
   another repository size before making capacity claims.
3. Add file history/churn/ownership queries, then decide symbol identity and
   extraction semantics before implementing symbol relationships.
4. Expose validated evidence to agents and eventually PR workflows. Risk scores
   require an operational definition and historical validation first.

## Original commit-TUI roadmap (frozen)

Decisions locked in a design session (2026-06-25). This file is the source of
truth so a fresh Claude session (or you, in a month) doesn't re-debate them.

## Why nib exists

A commit flow you keep open in the VSCode terminal and use 50× a day, faster
than the Source Control panel or lazygit/gitui (both tried, neither fit). It is
**not** a full git client — it's a fast committer + a few inspection commands.

## Decisions (locked)

| Topic | Decision |
|---|---|
| Repo | Separate from `conventional-stats`. No submodule. Independent. |
| Language | Go + Bubble Tea (TUI). |
| Versioning | SemVer, starts `0.1.0`. |
| Minimal flow | One line: type dropdown + message, over the file list. |
| Interaction | Mouse-first; keyboard always available. |
| Mouse caveat | Mouse capture hijacks terminal selection/scroll → toggle in `/settings`. |
| Staging | `git add` takes an explicit file list, never `git add .`. |
| Heavy features | Slash-commands: `/audit`, `/stats`, `/settings`. Not panels in the hot path. |
| Philosophy | Rich by default; everything heavy can be switched off. |
| `/audit` metric | Sort by **commit count** per file (Tornhill hotspots). Lines = context only. |
| Commit types | Duplicated from conventional-stats. No sync. `tests` → `test:`. |

## Phases

- [x] **Phase 0 — scaffold.** Structure, commit/git/audit/stats/settings
  packages, skeleton UI, mockups, CLAUDE.md. (current)
- [x] **Phase 1 — minimal commit flow (the core).** `internal/ui` wired to the
  file list + inline commit line through an injected `Repo` port (ADR-0005).
  Keyboard path: `j/k` move, `space` stage, type shortcuts, `tab` to message,
  `enter` commit, result reported. Transitions covered by tests. *MVP.*
- [x] **Phase 2 — type dropdown + mouse.** Three focus zones (Files → Type →
  Message, `tab`/`shift+tab` cycle); hybrid type dropdown (letter instant-select
  only in the Type zone, `↓`/`enter`/`space`/click opens the cheat-sheet,
  picking auto-advances to Message); `tea.MouseMsg` clicks via a pure
  `hitTest`, wheel scrolls the open dropdown. `layout()` is the single source of
  the row geometry shared by `View()` and `hitTest()`. `▸` marks the active zone
  + contextual help line. See the closed grill decisions in this file's history
  and `internal/ui/model_test.go`.
- [x] **Phase 3 — git-log panel.** Optional right-side log, `LogScope`
  current/all. Off-by-toggle. Panel is **read-only** y se auto-oculta bajo 80
  columnas; la app es ahora una **sesión persistente** (un commit refresca el
  estado en vez de cerrar — ADR-0007). El scope `custom` queda diferido a Fase 4.
- [x] **Phase 4 — `/settings`.** `/` command palette + persisted TOML
  (`~/.config/nib/settings.toml`), all toggles live: log panel, related panel,
  mouse capture, log scope, auto-reload interval. No custom log command;
  `/audit` rows wait for Phase 5 (ADR-0015).
- [x] **Phase 5 — `/audit`.** Hotspot screen from the `/` palette: commits,
  ±lines as context, `refactor?`/`vigilar` against the `/settings` threshold,
  window cycled in-screen (7 · 30 · 90 · 365). No flags; `.auditignore`
  replaces `--exclude` (ADR-0016).
- [x] **Phase 5b — `/changes`.** Diff review screen with search, scrollbar and
  `n`/`p`; editing delegated to `$EDITOR` (ADR-0017).
- [x] **Phase 6 — `/stats`.** Commits by type plus an activity row, per day up
  to 30 days and per week beyond; window cycled in-screen. Authors and scopes
  later (ADR-0018).
- [ ] **Phase 7 — release.** Superseded by the CI focus (ADR-0019).
  `gh repo create nib-git`, push, tag `v0.1.0`,
  build/install instructions, maybe `brew`/`go install`.

## Working style

Phase by phase, a commit at the end of each phase (visible progress in git
history). Conventional Commits in English, trailing period in the subject.

## Open questions

- Distribution: `go install`, Homebrew tap, or prebuilt binaries?
- Does `/audit` v0.2 (commits × file size) need file LOC, or git blame lines?
