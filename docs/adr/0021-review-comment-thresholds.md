# ADR-0021 — Measuring the review comment thresholds

- **Status:** accepted (protocol fixed before the run; results below)
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

## Results (2026-11-08)

Clones at the ADR-0013 HEADs (same commit counts: fzf 3,764 · Flask 5,562 ·
ripgrep 2,287 · OkHttp 6,489 · Astro 15,322 · Django 35,036). Cases: tuning
933 (fzf 170, Astro 175, Flask 49, ripgrep 132, OkHttp 182, Django 225),
validation 235 (51, 48, 29, 24, 31, 52).

Tuning chose **5 shared commits, ratio 0.5**: 39.0% precision shown, speaks on
25.2% of cases (the guess, 3 / 0.3: 31.8%, 60.6%).

Validation, mean over the six repositories:

| Thresholds | Precision shown | Speaks | fzf | Astro | Flask | ripgrep | OkHttp | Django |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| **5 / 0.5 (chosen)** | **34.9%** | 28.2% | 67.3% | 25.0% | 29.6% | 12.5% | 25.0% | 50.0% |
| 3 / 0.3 (guess) | 32.0% | 62.7% | 41.6% | 16.9% | 28.4% | 18.4% | 25.0% | 61.9% |
| 1 / 0 (no filter) | 17.7% | 98.2% | 31.0% | 8.3% | 18.6% | 13.3% | 22.1% | 13.1% |

The rule adopts 5 / 0.5: more precise on validation and above the 25% floor.

Reading:

- Filtering is what makes the comment worth reading: without thresholds only
  1 in 6 shown suggestions is right; with them, about 1 in 3.
- The gain over the guess is small (+2.9 points) and costs half of the
  comments (62.7% → 28.2% of cases). That trade is what the protocol asked
  for: a reviewer that is right more often and silent otherwise.
- Repositories differ more than configurations: fzf reaches 67%, ripgrep 12.5%
  on 24 cases, and Django speaks on only 4% of its validation cases.
- On validation, 5 / 0.4 scored higher (37.2%, speaks 36.9%). It was not the
  tuning choice, so it is not adopted; a future protocol may test it.

## Consequences

- One source file per case underestimates a real pull request, which has
  several changed files and keeps each suggestion's strongest relationship:
  real comments speak more often than measured here.
- Only the predictor's top 10 is filtered; a relationship ranked lower that
  would pass the thresholds is not seen. The comment shows 5, so the effect
  is small and only makes strict thresholds look quieter.
- The 25% floor is a product choice: a comment that never speaks is not a
  reviewer. It is fixed here so the choice cannot follow the data.
