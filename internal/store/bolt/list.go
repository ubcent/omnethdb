package bolt

import (
	"cmp"
	"omnethdb/internal/memory"
	"slices"
	"sort"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) ListMemories(req memory.ListMemoriesRequest) ([]memory.Memory, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if len(req.SpaceIDs) == 0 {
		return nil, nil
	}
	for _, spaceID := range req.SpaceIDs {
		if err := memory.ValidateSpaceID(spaceID); err != nil {
			return nil, err
		}
	}
	for _, kind := range req.Kinds {
		if err := memory.ValidateMemoryKind(kind); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	results := make([]memory.Memory, 0)
	kindFilter := kindSet(req.Kinds)

	err := s.db.View(func(tx *bbolt.Tx) error {
		for _, spaceID := range req.SpaceIDs {
			for _, id := range loadSpaceMemoryIDs(tx, spaceID) {
				mem, err := loadMemory(tx, id)
				if err != nil {
					return err
				}
				if !isRecallEligible(mem, now, kindFilter, req.ExcludeOrphanedDerives) {
					continue
				}
				results = append(results, *mem)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(results, func(i, j int) bool {
		if !results[i].CreatedAt.Equal(results[j].CreatedAt) {
			return results[i].CreatedAt.Before(results[j].CreatedAt)
		}
		return cmp.Less(results[i].ID, results[j].ID)
	})

	return slices.Clone(results), nil
}
