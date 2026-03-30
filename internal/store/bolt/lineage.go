package bolt

import (
	"encoding/json"
	"omnethdb/internal/memory"
	"sort"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) GetLineage(rootID string) ([]memory.Memory, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateMemoryID(rootID); err != nil {
		return nil, err
	}

	lineage := make([]memory.Memory, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMemories)
		c := b.Cursor()
		for _, raw := c.First(); raw != nil; _, raw = c.Next() {
			var mem memory.Memory
			if err := json.Unmarshal(raw, &mem); err != nil {
				return err
			}

			memRootID := mem.ID
			if mem.RootID != nil {
				memRootID = *mem.RootID
			}
			if memRootID != rootID {
				continue
			}

			lineage = append(lineage, mem)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(lineage) == 0 {
		return nil, memory.ErrMemoryNotFound
	}

	sort.Slice(lineage, func(i, j int) bool {
		if lineage[i].Version != lineage[j].Version {
			return lineage[i].Version < lineage[j].Version
		}
		if !lineage[i].CreatedAt.Equal(lineage[j].CreatedAt) {
			return lineage[i].CreatedAt.Before(lineage[j].CreatedAt)
		}
		return lineage[i].ID < lineage[j].ID
	})

	return lineage, nil
}
