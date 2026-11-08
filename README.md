# co-change

**English** · [Español](README.es.md)

![CI](https://github.com/nicovegasr/co-change/actions/workflows/ci.yml/badge.svg) · **[See this repository's architecture map →](https://nicovegasr.github.io/co-change/)**

A GitHub Action that tells you, on every pull request, **which files usually change with the ones you changed and were left untouched**, backed by the commits that prove it.

You changed the API but not its client. You touched the migration but not the model. Your repository's history already knows these files travel together; co-change reads it and leaves one comment on the PR:

> **co-change** · files that usually change with this PR but are untouched:
>
> | File | Usually changes with | Co-change | Evidence |
> |---|---|---|---|
> | `CHANGELOG.md` | `main.go` | 90% (9 commits) | 614c265, 61d6cfa, ef02702 |
> | `internal/audit/audit_test.go` | `internal/audit/audit.go` | 75% (3 commits) | 8b6387b, d85798c, d0cc3f0 |
> | `go.sum` | `go.mod` | 75% (3 commits) | ee98508, a046d1a, cdac97a |
>
> <sub>Historical co-change before 521568b, not a dependency analysis.</sub>

No AI guessing, no configuration, no code leaves the runner: it is plain git history, and every row links to commits you can open.

---

## Use it

```yaml
# .github/workflows/co-change.yml
name: co-change
on: pull_request

permissions:
  contents: read
  pull-requests: write

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0        # co-change needs the full history
      - uses: nicovegasr/co-change@main
```

| Input | Default | Meaning |
|---|---|---|
| `base` | the PR's base branch | Branch or commit the PR targets |
| `min-support` | `3` | Minimum commits both files changed in together |
| `min-ratio` | `0.3` | Minimum share of the changed file's commits that also touched the suggestion |
| `limit` | `5` | Maximum suggestions |
| `token` | `github.token` | Token allowed to comment |

The review is also written to the job summary. Pull requests from forks get a read-only token, so they only get the summary.

---

## How it works

1. Indexes the repository's history inside the runner (`.git/co-change/`).
2. Takes the files the PR changes since it branched off (the merge base).
3. For each one, finds the files that changed with it **in the history before the merge base**, so the PR's own commits never confirm its suggestions.
4. Keeps the untouched files that still exist and pass the noise filter, ranked with recent history weighing more (90-day half-life).
5. Creates one comment and updates it on every push. If nothing is found it posts nothing, or updates its earlier comment to say so.

It is **informational**: it never fails the check or blocks a merge.

### Cut the noise — `.cochangeignore`

Lockfiles, changelogs or generated code may co-change with everything. Add a `.cochangeignore` at the root with one glob or directory per line:

```gitignore
*.lock
package-lock.json
CHANGELOG.md
dist/
```

---

## Architecture map

A second, reusable workflow publishes a map of your repository to GitHub Pages on every push to `main`: folders as boxes, the imports between them, and **the folders that change together with no import between them** (hidden coupling: shared data, a contract, a copy-paste).

```yaml
# .github/workflows/architecture.yml
name: Architecture map
on:
  push:
    branches: [main]

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  map:
    uses: nicovegasr/co-change/.github/workflows/map.yml@main
```

Enable Pages once in *Settings → Pages → Source: GitHub Actions*. Then link the map from your README.

- **Boxes** are folders with a `README.md` (its title and first paragraph describe the box). Code outside them is grouped by top-level folder, or by child of `src/`, `internal/`, `pkg/`…, so it works without any README.
- **Imports** are read from TS/JS (with `tsconfig` aliases), Go, Java/Kotlin, C# and Python, with no compiler in CI. An import it cannot resolve is left out, never guessed.
- **Change together** uses [code-maat](https://github.com/adamtornhill/code-maat)'s defaults: at least 5 shared commits, 30% of the mean commits of both folders, commits of at most 30 files.
- Rows stack folders by import depth: what nothing imports on top, foundations at the bottom.

Locally: `co-change map` writes `co-change-map/index.html`. The step alone, to publish elsewhere: `uses: nicovegasr/co-change/map@main` with `upload: false`.

---

## Does it actually work?

`co-change evaluate` hides past commits and asks, using **only the history before** each one, which other files changed. It compares against two baselines: plain co-change counting with no filters (`unfiltered`) and suggesting the repository's most-touched files (`frequency`).

Precision of the top 5 suggestions on each repository's most recent commits (parameters were tuned on older commits):

| Repository | co-change | unfiltered | frequency |
|---|---:|---:|---:|
| fzf (Go) | **30.6%** | 23.9% | 18.0% |
| Django (Python) | **13.1%** | 11.5% | 2.3% |
| OkHttp (Kotlin/Java) | 20.0% | **21.9%** | 1.9% |
| Flask (Python) | 18.6% | **19.3%** | 2.8% |
| ripgrep (Rust) | 13.3% | 13.3% | 4.2% |
| Astro (TypeScript) | 7.9% | 7.9% | 1.7% |
| Personal blog (Astro) | 22.3% | **23.3%** | 11.6% |
| mean of 7 repositories | **18.0%** | 17.3% | 6.1% |

The honest reading: looking at **what changes with your file** hits about 3 times more often on average than looking at what changes most overall (from 1.7× on fzf to 10× on OkHttp). The filters add little over plain co-change on these small samples (24–52 cases per repository); on larger samples the advantage is about 2.5 points. These numbers measure ranking, not the PR comment: its thresholds (`min-support 3`, `min-ratio 0.3`) are a first choice to keep it quiet, not yet validated. Methodology in [docs/research/file-coupling-benchmarks.md](docs/research/file-coupling-benchmarks.md).

---

## Run it locally

Requires Go 1.26+ and git.

```bash
git clone https://github.com/nicovegasr/co-change
cd co-change
make install                                   # installs co-change into ~/go/bin
```

Then, inside any repository:

```bash
co-change review --base main                   # the PR comment for your branch
co-change review --base main --json            # the same, for other tools
co-change coupled internal/ui/model.go         # what changes with one file, with evidence
co-change coupled src/app.ts --before v1.0     # only history before a commit
co-change evaluate --prefix src/ --ext .ts     # measure it on your own repository
co-change map                                  # the architecture map, in co-change-map/
```

`co-change` with no arguments opens a terminal commit tool (Conventional Commits in one line, with related files while you mark them); see [docs/tui.md](docs/tui.md).

---

## Limits

- A file is a **path**: renames do not join their history.
- Only history reachable from `HEAD`; shallow clones are rejected.
- The Action builds co-change from source and indexes from scratch on every run (under a second on this repository; large repositories will take longer until caching lands).
- Historical co-change is not dependency nor a bug prediction: it is a hint.

---

## Documentation

| Document | Contents |
|---|---|
| [docs/adr/](docs/adr/) | Decisions, one per file (ADR-0019: the CI focus, ADR-0020: the map) |
| [docs/research/file-coupling-benchmarks.md](docs/research/file-coupling-benchmarks.md) | Benchmarks and evaluations on real repositories |
| [docs/architecture.md](docs/architecture.md) | Packages, flow and design rules |
| [docs/tui.md](docs/tui.md) | The commit TUI |
| [docs/ROADMAP.md](docs/ROADMAP.md) | What comes next |
| [docs/contributing.md](docs/contributing.md) | How to work on the repository |

---

## Development

```bash
make check   # format, vet, lint and tests (pre-commit)
make ci      # all of the above + ≥ 80% coverage + real git integration
make bench   # index microbenchmarks
```
