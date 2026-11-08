#!/usr/bin/env python3
"""Review comment thresholds (ADR-0021): precision of what the comment shows and how often it speaks.

Usage: scripts/review-thresholds.py WORKDIR

WORKDIR holds a `co-change` binary and full clones named after the
repositories, checked out at the ADR-0013 HEADs. The script writes each
clone's .cochangeignore, indexes it, runs `evaluate --json` once per window
(cached as WORKDIR/<repo>-<split>.json) and applies every threshold pair
afterwards. Prints both windows and writes WORKDIR/review-thresholds.json.
"""
import json
import pathlib
import subprocess
import sys

SOURCES = {
    "fzf": ["--ext", ".go"],
    "astro": ["--prefix", "packages/astro/src/", "--ext", ".ts,.js,.astro"],
    "flask": ["--prefix", "src/", "--ext", ".py"],
    "ripgrep": ["--ext", ".rs"],
    "okhttp": ["--ext", ".kt,.java"],
    "django": ["--prefix", "django/", "--ext", ".py"],
}
WINDOWS = {
    "tune": ["--offset", "400", "--window", "1600", "--stride", "3"],
    "validate": ["--window", "400", "--stride", "3"],
}
IGNORE = "*.lock\npnpm-lock.yaml\npackage-lock.json\ngo.sum\n.changeset/\nCHANGELOG.md\nCHANGES.rst\nCHANGES.md\nCHANGELOG.rst\n"
SUPPORTS = [1, 2, 3, 4, 5]
RATIOS = [0.0, 0.1, 0.2, 0.3, 0.4, 0.5]
SHOWN = 5
SPEAK_FLOOR = 0.25
CURRENT = (3, 0.3)


def cases(workdir, repo, split):
    cache = workdir / f"{repo}-{split}.json"
    if not cache.exists():
        clone = workdir / repo
        (clone / ".cochangeignore").write_text(IGNORE)
        subprocess.run([str(workdir / "co-change"), "index"], cwd=clone, check=True, stdout=subprocess.DEVNULL)
        args = [str(workdir / "co-change"), "evaluate", "--json", "--case-max-files", "50", *SOURCES[repo], *WINDOWS[split]]
        cache.write_text(subprocess.run(args, cwd=clone, check=True, capture_output=True, text=True).stdout)
    return json.loads(cache.read_text())["cases"]


def measure(repo_cases, support, ratio):
    shown = hits = speaking = 0
    for case in repo_cases:
        comment = [r for r in case["cochange"] or [] if r["shared_commits"] >= support and r["cochange_ratio"] >= ratio][:SHOWN]
        shown += len(comment)
        hits += sum(r["path"] in case["targets"] for r in comment)
        speaking += bool(comment)
    return {"precision": hits / shown if shown else 0.0, "speaks": speaking / len(repo_cases), "shown": shown}


def table(workdir, split):
    data = {repo: cases(workdir, repo, split) for repo in SOURCES}
    rows = {}
    for s in SUPPORTS:
        for r in RATIOS:
            per_repo = {repo: measure(c, s, r) for repo, c in data.items()}
            rows[(s, r)] = {
                "repos": per_repo,
                "precision": sum(m["precision"] for m in per_repo.values()) / len(per_repo),
                "speaks": sum(m["speaks"] for m in per_repo.values()) / len(per_repo),
            }
    return rows, {repo: len(c) for repo, c in data.items()}


def choose(rows):
    eligible = [(k, v) for k, v in rows.items() if v["speaks"] >= SPEAK_FLOOR]
    best = max(v["precision"] for _, v in eligible)
    near = [(k, v) for k, v in eligible if v["precision"] >= best - 0.005]
    return max(near, key=lambda kv: kv[1]["speaks"])[0]


def show(title, rows, counts, marks):
    print(f"\n{title} · cases " + " ".join(f"{r} {n}" for r, n in counts.items()))
    print("config      precision  speaks  " + "  ".join(f"{r:>8}" for r in counts))
    for (s, r), v in sorted(rows.items(), key=lambda kv: -kv[1]["precision"]):
        mark = "  ".join(m for m, k in marks.items() if k == (s, r))
        cells = "  ".join(f"{v['repos'][repo]['precision'] * 100:5.1f}/{v['repos'][repo]['speaks'] * 100:3.0f}" for repo in counts)
        print(f"s{s}/r{r:.1f}   {v['precision'] * 100:8.1f}%  {v['speaks'] * 100:5.1f}%  {cells}  {mark}")


def main():
    workdir = pathlib.Path(sys.argv[1]).resolve()
    tune, tune_counts = table(workdir, "tune")
    chosen = choose(tune)
    validate, validate_counts = table(workdir, "validate")
    marks = {"<- chosen": chosen, "<- current": CURRENT}
    show("TUNE", tune, tune_counts, marks)
    show("VALIDATE", validate, validate_counts, marks)
    new, old = validate[chosen], validate[CURRENT]
    adopt = new["precision"] >= old["precision"] and new["speaks"] >= SPEAK_FLOOR
    print(f"\nchosen s{chosen[0]}/r{chosen[1]}: validation {new['precision'] * 100:.1f}% shown precision, speaks {new['speaks'] * 100:.1f}%")
    print(f"current s{CURRENT[0]}/r{CURRENT[1]}: validation {old['precision'] * 100:.1f}% shown precision, speaks {old['speaks'] * 100:.1f}%")
    print("ADOPT chosen" if adopt and chosen != CURRENT else "KEEP current")
    out = {split: {f"s{s}/r{r}": v for (s, r), v in rows.items()} for split, rows in (("tune", tune), ("validate", validate))}
    out["chosen"], out["adopt"] = f"s{chosen[0]}/r{chosen[1]}", adopt
    (workdir / "review-thresholds.json").write_text(json.dumps(out, indent=1))


if __name__ == "__main__":
    main()
