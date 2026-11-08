package index

import (
	"context"
	"strings"

	bolt "go.etcd.io/bbolt"
)

// ChangeSet is what one eligible commit changed, for analyses that group paths
// themselves (the architecture map, ADR-0020).
type ChangeSet struct {
	OID       string   `json:"oid"`
	Subject   string   `json:"subject"`
	Timestamp int64    `json:"timestamp"`
	Paths     []string `json:"paths"`
}

// ChangeSets lists the non-merge commits changing at most maxFiles paths, in
// index order, with the same eligibility rule as Coupled.
func (s *Store) ChangeSets(ctx context.Context, maxFiles int) ([]ChangeSet, error) {
	var sets []ChangeSet
	err := s.db.View(func(tx *bolt.Tx) error {
		paths := bucket(tx, "paths")
		return bucket(tx, "commits").ForEach(func(id, _ []byte) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			c, err := readCommit(tx, id)
			if err != nil || !eligibleObservation(c, Options{MaxFiles: maxFiles}) {
				return err
			}
			set := ChangeSet{OID: c.OID, Subject: strings.SplitN(c.Message, "\n", 2)[0], Timestamp: c.Timestamp}
			for _, f := range c.Files {
				set.Paths = append(set.Paths, string(paths.Get(key(f.FileID))))
			}
			sets = append(sets, set)
			return nil
		})
	})
	return sets, err
}
