# Contributing

## Setup (once)

```sh
go mod tidy     # populate go.sum
make tools      # install golangci-lint + lefthook, wire git hooks
```

## The loop

```sh
make check            # fmt + vet + lint + test — the pre-commit gate
make cover            # per-package coverage gate (≥80% domain) — pre-push & CI
make test-integration # real git, happy paths (build tag: integration)
make build            # build ./co-change
make run              # build and run
make help             # list all targets
```

`lefthook` runs `make check` on commit and `make cover` on push. CI runs
`make ci` (both). All three call the same Makefile — one source of truth
(ADR-0003).

## Rules

- **TDD, pragmatic.** Logic (parsing, ranking, `Format`, model transitions) is
  written test-first. Raw rendering and the git shell-out are not. Domain
  packages stay ≥80% covered; `internal/ui` requires 0% (ADR-0002).
- **Three test tiers** (ADR-0006): unit (stubbed, counts to coverage),
  integration (real git, `integration` tag), UI e2e (`teatest`, later).
- **Branch + PR**, never commit to `main` (ADR-0004).
- **Conventional Commits in English**, trailing period in the subject
  (`feat: add type dropdown.`).

## Code style

- **Good naming** over comments. Names carry intent.
- **Rule of three:** do not abstract until ~3 real repetitions justify it.
  No speculative interfaces or layers.
- **CQS:** a function either changes state (command) or returns data (query),
  not both.
- **Less code is better.** Prefer the smallest clear solution; cut boilerplate.
- **Bilingual on purpose:** identifiers/commits/comments in English;
  user-facing strings in Spanish (that is why `misspell` is off).

## Adding an architectural decision

Non-obvious design choices get an ADR *before* implementing:
copy `docs/adr/0000-template.md` to the next number and fill it in.
