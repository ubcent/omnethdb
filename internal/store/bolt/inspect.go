package bolt

import (
	"encoding/json"
	"omnethdb/internal/memory"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) GetRelated(memoryID string, kind memory.RelationType, depth int) ([]memory.Memory, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateMemoryID(memoryID); err != nil {
		return nil, err
	}
	if depth <= 0 {
		return nil, nil
	}

	results := make([]memory.Memory, 0)
	seen := map[string]struct{}{memoryID: {}}
	current := []string{memoryID}

	err := s.db.View(func(tx *bbolt.Tx) error {
		for level := 0; level < depth && len(current) > 0; level++ {
			next := make([]string, 0)
			for _, fromID := range current {
				ids, err := loadRelatedIDs(tx, fromID, kind)
				if err != nil {
					return err
				}
				for _, toID := range ids {
					if _, ok := seen[toID]; ok {
						continue
					}
					seen[toID] = struct{}{}
					mem, err := loadMemoryByID(tx, toID)
					if err != nil {
						return err
					}
					results = append(results, *mem)
					next = append(next, toID)
				}
			}
			current = next
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (s *Store) GetAuditLog(spaceID string, since time.Time) ([]memory.AuditEntry, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(spaceID); err != nil {
		return nil, err
	}

	var entries []memory.AuditEntry
	err := s.db.View(func(tx *bbolt.Tx) error {
		var err error
		entries, err = loadAuditEntries(tx, spaceID, since)
		return err
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func loadRelatedIDs(tx *bbolt.Tx, fromID string, kind memory.RelationType) ([]string, error) {
	return loadIndexedRelatedIDs(tx, fromID, kind)
}

func loadMemoryByID(tx *bbolt.Tx, id string) (*memory.Memory, error) {
	raw := tx.Bucket(bucketMemories).Get([]byte(id))
	if raw == nil {
		return nil, memory.ErrMemoryNotFound
	}
	var mem memory.Memory
	if err := json.Unmarshal(raw, &mem); err != nil {
		return nil, err
	}
	return &mem, nil
}
