package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/nicovegasr/co-change/internal/history"
	"github.com/nicovegasr/co-change/internal/index"
)

// The PR comment carries reviewMarker so the Action updates it in place, and
// emptyMarker when there is nothing to suggest (ADR-0019).
const (
	reviewMarker = "<!-- co-change -->"
	emptyMarker  = "<!-- co-change:empty -->"
)

// Review is what a pull request changed and the untouched paths that usually
// change with it, according to history before the merge base.
type Review struct {
	MergeBase   string               `json:"merge_base"`
	Changed     []string             `json:"changed_files"`
	Suggestions []index.Relationship `json:"suggestions"`
	Repository  string               `json:"-"` // links commits in the comment
}

// reviewFilter keeps a PR comment from crying wolf. Its defaults were chosen
// on held-out commits of six repositories (ADR-0021).
type reviewFilter struct {
	MinSupport int
	MinRatio   float64
	Limit      int
}

func runReview(ctx context.Context, dir string, args []string, out io.Writer) error {
	filter := reviewFilter{MinSupport: 5, MinRatio: 0.5, Limit: 5}
	fs := flags("review", out)
	base := fs.String("base", "", "Branch or commit the pull request targets")
	asJSON := fs.Bool("json", false, "Emit JSON")
	fs.IntVar(&filter.MinSupport, "min-support", filter.MinSupport, "Minimum shared commits")
	fs.Float64Var(&filter.MinRatio, "min-ratio", filter.MinRatio, "Minimum co-change ratio, 0..1")
	fs.IntVar(&filter.Limit, "limit", filter.Limit, "Maximum suggestions")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return helpError(err)
	}
	if len(positional) != 0 || *base == "" {
		return errors.New("usage: co-change review --base <ref> [options]")
	}
	if filter.MinSupport < 1 || filter.Limit < 1 || filter.MinRatio < 0 || filter.MinRatio > 1 {
		return errors.New("min-support and limit must be positive; min-ratio must be 0..1")
	}
	if err = Refresh(ctx, dir); err != nil {
		return err
	}
	review, err := reviewPullRequest(ctx, dir, *base, filter)
	if err != nil {
		return err
	}
	review.Repository = githubRepositoryURL()
	if *asJSON {
		return json.NewEncoder(out).Encode(review)
	}
	_, err = io.WriteString(out, reviewMarkdown(review))
	return err
}

func reviewPullRequest(ctx context.Context, dir, base string, filter reviewFilter) (Review, error) {
	repo, err := history.Open(ctx, dir)
	if err != nil {
		return Review{}, err
	}
	var r Review
	if r.MergeBase, err = repo.MergeBase(ctx, base); err != nil {
		return Review{}, err
	}
	if r.Changed, err = repo.ChangedSince(ctx, r.MergeBase); err != nil {
		return Review{}, err
	}
	trackedPaths, err := repo.Tracked(ctx)
	if err != nil {
		return Review{}, err
	}
	tracked := make(map[string]bool, len(trackedPaths))
	for _, p := range trackedPaths {
		tracked[p] = true
	}
	opts := index.DefaultOptions()
	opts.Limit, opts.Evidence, opts.Before = 100, 3, r.MergeBase
	err = withFreshIndex(repo, func(store *index.Store, ignore []string) error {
		opts.Ignore = ignore
		related, err := store.CoupledWithAny(ctx, r.Changed, opts)
		r.Suggestions = suggest(related, tracked, filter)
		return err
	})
	return r, err
}

// suggest keeps related paths that still exist and pass the filter, in rank order.
func suggest(related []index.Relationship, tracked map[string]bool, f reviewFilter) []index.Relationship {
	kept := []index.Relationship{}
	for _, r := range related {
		if len(kept) < f.Limit && tracked[r.Path] && r.Shared >= f.MinSupport && r.Ratio >= f.MinRatio {
			kept = append(kept, r)
		}
	}
	return kept
}

// strongRatio marks the suggestions that almost always travel together.
const strongRatio = 0.75

func reviewMarkdown(r Review) string {
	var b strings.Builder
	b.WriteString(reviewMarker + "\n")
	if len(r.Suggestions) == 0 {
		fmt.Fprintf(&b, "%s\n### ✨ co-change · all good!\n\nNothing that usually travels with these %s is missing.\n", emptyMarker, plural(len(r.Changed), "file"))
		return b.String()
	}
	b.WriteString("### 🧳 co-change · files that usually travel together\n\n")
	fmt.Fprintf(&b, "This PR changes **%s**. In this repo's history, these usually change with them, and they're missing here:\n\n", plural(len(r.Changed), "file"))
	b.WriteString("| | File | Usually changes with | How often | Proof |\n|---|---|---|---|---|\n")
	for _, s := range r.Suggestions {
		mood := "👀"
		if s.Ratio >= strongRatio {
			mood = "🔥"
		}
		proof := make([]string, len(s.Evidence))
		for i, e := range s.Evidence {
			proof[i] = r.commit(e.OID)
		}
		fmt.Fprintf(&b, "| %s | `%s` | `%s` | **%d of %d** times | %s |\n", mood, s.Path, s.Source, s.Shared, int(math.Round(float64(s.Shared)/s.Ratio)), strings.Join(proof, " · "))
	}
	b.WriteString("\n💡 Just history, not a rule: if they don't need changes, ignore me. Too noisy? Add paths to `.cochangeignore`.\n\n")
	fmt.Fprintf(&b, "<sub>🕰️ History before `%s` · [co-change](https://github.com/nicovegasr/co-change)</sub>\n", short(r.MergeBase))
	return b.String()
}

// commit links an OID to the repository when it is known (in GitHub Actions).
func (r Review) commit(oid string) string {
	if r.Repository == "" {
		return "`" + short(oid) + "`"
	}
	return fmt.Sprintf("[%s](%s/commit/%s)", short(oid), r.Repository, oid)
}

func short(oid string) string { return oid[:min(len(oid), 7)] }

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
