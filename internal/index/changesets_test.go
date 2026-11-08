package index

import (
	"context"
	"reflect"
	"testing"
)

func TestChangeSetsListEligibleCommitsWithTheirPaths(t *testing.T) {
	s := linear(t, "AB", "ABCD", "C")

	got, err := s.ChangeSets(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}

	var paths [][]string
	for _, c := range got {
		paths = append(paths, c.Paths)
	}
	// The 4-path commit exceeds the limit of 3, like any mass change.
	if want := [][]string{{"A", "B"}, {"C"}}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths %v, want %v", paths, want)
	}
	if got[0].OID == "" {
		t.Error("a change set must cite its commit")
	}
}
