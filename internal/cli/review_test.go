package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/nicovegasr/co-change/internal/index"
)

func TestSuggestKeepsStrongTrackedRelationships(t *testing.T) {
	related := []index.Relationship{
		{Path: "strong.go", Source: "a.go", Shared: 5, Ratio: 0.8},
		{Path: "deleted.go", Source: "a.go", Shared: 5, Ratio: 0.8},
		{Path: "weak-ratio.go", Source: "a.go", Shared: 9, Ratio: 0.2},
		{Path: "weak-support.go", Source: "a.go", Shared: 2, Ratio: 0.9},
		{Path: "second.go", Source: "b.go", Shared: 3, Ratio: 0.3},
		{Path: "third.go", Source: "b.go", Shared: 3, Ratio: 0.3},
	}
	tracked := map[string]bool{"strong.go": true, "weak-ratio.go": true, "weak-support.go": true, "second.go": true, "third.go": true}

	got := suggest(related, tracked, reviewFilter{MinSupport: 3, MinRatio: 0.3, Limit: 2})

	var paths []string
	for _, s := range got {
		paths = append(paths, s.Path)
	}
	if want := []string{"strong.go", "second.go"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths %v, want %v", paths, want)
	}
}

func TestReviewMarkdownListsSuggestionsWithEvidence(t *testing.T) {
	r := Review{MergeBase: "0123456789abcdef", Changed: []string{"a.go"}, Suggestions: []index.Relationship{{
		Path: "b.go", Source: "a.go", Shared: 4, Ratio: 0.5,
		Evidence: []index.Evidence{{OID: "abcdef0123456789", Subject: "feat: x."}, {OID: "fedcba9876543210"}},
	}}}

	md := reviewMarkdown(r)

	for _, part := range []string{reviewMarker, "| `b.go` | `a.go` | 50% (4 commits) | abcdef0, fedcba9 |", "0123456"} {
		if !strings.Contains(md, part) {
			t.Errorf("missing %q in:\n%s", part, md)
		}
	}
	if strings.Contains(md, emptyMarker) {
		t.Error("a review with suggestions is not empty")
	}
}

func TestReviewMarkdownWithoutSuggestionsIsMarkedEmpty(t *testing.T) {
	md := reviewMarkdown(Review{MergeBase: "0123456789abcdef", Changed: []string{"a.go"}})
	if !strings.Contains(md, reviewMarker) || !strings.Contains(md, emptyMarker) {
		t.Fatal(md)
	}
}
