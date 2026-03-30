package bolt

import (
	"encoding/json"
	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) Revive(rootID string, input memory.ReviveInput) (*memory.Memory, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateMemoryID(rootID); err != nil {
		return nil, err
	}
	if err := memory.ValidateContent(input.Content); err != nil {
		return nil, err
	}
	if err := memory.ValidateMemoryKind(input.Kind); err != nil {
		return nil, err
	}
	if err := memory.ValidateActor(input.Actor); err != nil {
		return nil, err
	}
	if err := memory.ValidateConfidence(input.Confidence); err != nil {
		return nil, err
	}

	var revived *memory.Memory
	err := s.db.Update(func(tx *bbolt.Tx) error {
		if tx.Bucket(bucketLatest).Get([]byte(rootID)) != nil {
			return memory.ErrLineageActive
		}

		root, err := loadMemoryForForget(tx, rootID)
		if err != nil {
			return err
		}
		if root.Kind == memory.KindDerived {
			return memory.ErrReviveDerivedUnsupported
		}
		if input.Kind != root.Kind {
			return memory.ErrUpdateAcrossKinds
		}

		cfg, err := loadSpaceConfig(tx, root.SpaceID)
		if err != nil {
			return err
		}
		if err := ensureSpaceWritable(cfg); err != nil {
			return err
		}
		if !policy.CanWriteKind(cfg.WritePolicy, input.Actor, input.Kind) {
			return memory.ErrPolicyViolation
		}

		predecessor, err := loadForgottenLatestForRoot(tx, rootID)
		if err != nil {
			return err
		}

		mem := &memory.Memory{
			ID:         newMemoryID(),
			SpaceID:    root.SpaceID,
			Content:    input.Content,
			Kind:       input.Kind,
			Actor:      input.Actor,
			Confidence: input.Confidence,
			Metadata:   cloneMetadata(input.Metadata),
			CreatedAt:  time.Now().UTC(),
			Version:    predecessor.Version + 1,
			IsLatest:   true,
			ParentID:   &predecessor.ID,
			RootID:     &rootID,
		}

		if err := saveMemory(tx, mem); err != nil {
			return err
		}
		if err := putLatest(tx, rootID, mem.ID); err != nil {
			return err
		}
		if err := putSpaceMemoryIDs(tx, root.SpaceID, appendSpaceMemoryID(loadSpaceMemoryIDs(tx, root.SpaceID), mem.ID)); err != nil {
			return err
		}
		if err := writeAuditEntry(tx, memory.AuditEntry{
			Timestamp: mem.CreatedAt,
			SpaceID:   mem.SpaceID,
			Operation: "revive",
			Actor:     mem.Actor,
			MemoryIDs: []string{predecessor.ID, mem.ID},
		}); err != nil {
			return err
		}

		revived = mem
		return nil
	})
	if err != nil {
		return nil, err
	}

	return revived, nil
}

func loadForgottenLatestForRoot(tx *bbolt.Tx, rootID string) (*memory.Memory, error) {
	b := tx.Bucket(bucketMemories)
	c := b.Cursor()
	for key, raw := c.First(); key != nil; key, raw = c.Next() {
		var mem memory.Memory
		if err := json.Unmarshal(raw, &mem); err != nil {
			return nil, err
		}

		memRootID := mem.ID
		if mem.RootID != nil {
			memRootID = *mem.RootID
		}
		if memRootID != rootID {
			continue
		}
		if !mem.IsForgotten {
			continue
		}
		if mem.IsLatest {
			continue
		}

		isSuccessor := false
		for childKey, childRaw := c.First(); childKey != nil; childKey, childRaw = c.Next() {
			var child memory.Memory
			if err := json.Unmarshal(childRaw, &child); err != nil {
				return nil, err
			}
			if child.ParentID != nil && *child.ParentID == mem.ID {
				isSuccessor = true
				break
			}
		}
		if !isSuccessor {
			memCopy := mem
			return &memCopy, nil
		}
	}

	return nil, memory.ErrMemoryNotFound
}
