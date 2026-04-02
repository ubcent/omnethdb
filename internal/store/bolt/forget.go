package bolt

import (
	"encoding/json"
	"omnethdb/internal/memory"
	"strings"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) Forget(id string, actor memory.Actor, reason string) error {
	if s == nil || s.db == nil {
		return memory.ErrStoreClosed
	}
	if err := memory.ValidateMemoryID(id); err != nil {
		return err
	}
	if err := memory.ValidateActor(actor); err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return memory.ErrInvalidContent
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		return forgetInTx(tx, id, actor, reason)
	})
}

func forgetInTx(tx *bbolt.Tx, id string, actor memory.Actor, reason string) error {
	mem, err := loadMemoryForForget(tx, id)
	if err != nil {
		return err
	}
	cfg, err := loadSpaceConfig(tx, mem.SpaceID)
	if err != nil {
		return err
	}
	if err := ensureSpaceWritable(cfg); err != nil {
		return err
	}
	if mem.IsForgotten {
		return nil
	}

	wasLatest := mem.IsLatest
	mem.IsForgotten = true
	mem.IsLatest = false

	if err := saveMemory(tx, mem); err != nil {
		return err
	}
	if err := markOrphanedDeriveds(tx, mem.SpaceID, mem.ID); err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := writeAuditEntry(tx, memory.AuditEntry{
		Timestamp: now,
		SpaceID:   mem.SpaceID,
		Operation: "forget",
		Actor:     actor,
		MemoryIDs: []string{mem.ID},
		Reason:    reason,
	}); err != nil {
		return err
	}
	if err := writeForgetRecord(tx, memory.ForgetRecord{
		MemoryID:  mem.ID,
		SpaceID:   mem.SpaceID,
		Actor:     actor,
		Reason:    reason,
		Timestamp: now,
	}); err != nil {
		return err
	}
	if !wasLatest {
		return nil
	}

	rootID := mem.ID
	if mem.RootID != nil {
		rootID = *mem.RootID
	}
	if err := deleteLatest(tx, rootID); err != nil {
		return err
	}
	if err := removeSpaceMemoryID(tx, mem.SpaceID, mem.ID); err != nil {
		return err
	}
	return incrementLiveKindCount(tx, mem.SpaceID, mem.Kind, -1)
}

func markOrphanedDeriveds(tx *bbolt.Tx, spaceID string, sourceID string) error {
	derivedIDs, err := loadInboundRelationIDs(tx, sourceID, memory.RelationDerives)
	if err != nil {
		return err
	}
	for _, id := range derivedIDs {
		mem, err := loadMemory(tx, id)
		if err != nil {
			return err
		}
		if mem.SpaceID != spaceID {
			continue
		}
		if mem.Kind != memory.KindDerived {
			continue
		}
		if !mem.IsLatest || mem.IsForgotten {
			continue
		}
		if !containsSourceID(mem.SourceIDs, sourceID) {
			continue
		}
		if mem.HasOrphanedSources {
			continue
		}
		mem.HasOrphanedSources = true
		if err := saveMemory(tx, mem); err != nil {
			return err
		}
	}
	return nil
}

func loadMemoryForForget(tx *bbolt.Tx, id string) (*memory.Memory, error) {
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

func deleteLatest(tx *bbolt.Tx, rootID string) error {
	return tx.Bucket(bucketLatest).Delete([]byte(rootID))
}

func containsSourceID(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
