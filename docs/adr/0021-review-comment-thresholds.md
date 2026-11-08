# ADR-0021 — Measuring the review comment thresholds

- **Status:** proposed (protocol fixed before any run; results pending)
- **Date:** 2026-11-08

## Context

`co-change review` (ADR-0019) shows at most 5 suggestions with at least 3
shared commits and a co-change ratio of at least 0.3. Those numbers were a
guess. They decide whether the comment is worth reading: too loose and it
cries wolf, too strict and it never speaks.

ADR-0009..0013 measured the *ranking* (precision@5, empty slots count as
misses). The comment is a different product question: of what it shows, how
much is right, and how often does it show anything.

## Decision

The protocol below is fixed before any result is seen.

- **Repositories and windows:** the six public repositories of ADR-0013 at
  the same HEADs (fzf, Astro, Flask, ripgrep, OkHttp, Django) with the same
  sources, `.cochangeignore` and `--case-max-files 50`. The personal blog is
  left out: its pinned HEAD no longer exists. Tuning windows are
  `--offset 400 --window 1600 --stride 3`; validation windows are
  `--window 400 --stride 3`.
- **Cases:** `evaluate` held-out commits, one source file each, predictor at
  the shipped defaults (support 1, ≤ 50 files, recent ranking, 90-day
  half-life, top 10). `evaluate --json` now also reports each co-change
  prediction's shared commits and ratio, so thresholds are applied afterwards
  without re-running anything.
- **Simulated comment:** for thresholds (S, R), the first 5 predictions with
  shared ≥ S and ratio ≥ R.
- **Metrics,** per repository:
  - *precision shown*: hits / suggestions shown, over all cases;
  - *speaks*: share of cases with at least one suggestion.
- **Grid:** S ∈ {1, 2, 3, 4, 5} × R ∈ {0, 0.1, 0.2, 0.3, 0.4, 0.5}.
- **Choice (tuning windows):** the highest mean precision shown over the six
  repositories among the configurations whose mean *speaks* is at least 25%;
  ties within half a point go to the one that speaks more.
- **Adoption (validation windows):** the chosen configuration replaces 3/0.3
  if its mean precision shown is not lower and it still speaks in at least
  25% of cases on average; otherwise 3/0.3 stays. Both are reported.

Reproduce with `scripts/review-thresholds.py WORKDIR`.

## Consequences

- One source file per case underestimates a real pull request, which has
  several changed files and keeps each suggestion's strongest relationship:
  real comments speak more often than measured here.
- Only the predictor's top 10 is filtered; a relationship ranked lower that
  would pass the thresholds is not seen. The comment shows 5, so the effect
  is small and only makes strict thresholds look quieter.
- The 25% floor is a product choice: a comment that never speaks is not a
  reviewer. It is fixed here so the choice cannot follow the data.
