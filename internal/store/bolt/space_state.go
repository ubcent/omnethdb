package bolt

import (
	"bytes"
	"encoding/json"
	"omnethdb/internal/memory"
	"slices"

	bbolt "go.etcd.io/bbolt"
)

type liveKindCounts struct {
	Episodic int `json:"episodic"`
	Static   int `json:"static"`
	Derived  int `json:"derived"`
}

func loadLiveKindCounts(tx *bbolt.Tx, spaceID string) (map[memory.MemoryKind]int, error) {
	b := tx.Bucket(bucketSpaceStats)
	raw := b.Get([]byte(spaceID))
	if raw != nil {
		var persisted liveKindCounts
		if err := json.Unmarshal(raw, &persisted); err != nil {
			return nil, err
		}
		return map[memory.MemoryKind]int{
			memory.KindEpisodic: persisted.Episodic,
			memory.KindStatic:   persisted.Static,
			memory.KindDerived:  persisted.Derived,
		}, nil
	}

	counts, err := countLiveKindsByScan(tx, spaceID)
	if err != nil {
		return nil, err
	}
	if err := saveLiveKindCounts(tx, spaceID, counts); err != nil {
		return nil, err
	}
	return counts, nil
}

func saveLiveKindCounts(tx *bbolt.Tx, spaceID string, counts map[memory.MemoryKind]int) error {
	encoded, err := json.Marshal(liveKindCounts{
		Episodic: counts[memory.KindEpisodic],
		Static:   counts[memory.KindStatic],
		Derived:  counts[memory.KindDerived],
	})
	if err != nil {
		return err
	}
	return tx.Bucket(bucketSpaceStats).Put([]byte(spaceID), encoded)
}

func incrementLiveKindCount(tx *bbolt.Tx, spaceID string, kind memory.MemoryKind, delta int) error {
	counts, err := loadLiveKindCounts(tx, spaceID)
	if err != nil {
		return err
	}
	counts[kind] += delta
	return saveLiveKindCounts(tx, spaceID, counts)
}

func countLiveKindsByScan(tx *bbolt.Tx, spaceID string) (map[memory.MemoryKind]int, error) {
	counts := map[memory.MemoryKind]int{
		memory.KindEpisodic: 0,
		memory.KindStatic:   0,
		memory.KindDerived:  0,
	}

	for _, id := range loadSpaceMemoryIDs(tx, spaceID) {
		mem, err := loadMemory(tx, id)
		if err != nil {
			return nil, err
		}
		counts[mem.Kind]++
	}

	return counts, nil
}

func loadSpaceMemoryIDs(tx *bbolt.Tx, spaceID string) []string {
	if migrated, err := loadIndexedSpaceMemoryIDs(tx, spaceID); err == nil && (len(migrated) > 0 || hasPersistedLiveKindCounts(tx, spaceID)) {
		return migrated
	}

	raw := tx.Bucket(bucketSpaces).Get([]byte(spaceID))
	if raw == nil {
		return nil
	}

	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil
	}
	return ids
}

func hasPersistedLiveKindCounts(tx *bbolt.Tx, spaceID string) bool {
	return tx.Bucket(bucketSpaceStats).Get([]byte(spaceID)) != nil
}

func addSpaceMemoryID(tx *bbolt.Tx, spaceID string, memoryID string) error {
	if err := migrateLegacySpaceMemoryIDs(tx, spaceID); err != nil {
		return err
	}
	return tx.Bucket(bucketSpaces).Put(spaceMembershipKey(spaceID, memoryID), []byte{1})
}

func removeSpaceMemoryID(tx *bbolt.Tx, spaceID string, memoryID string) error {
	if err := migrateLegacySpaceMemoryIDs(tx, spaceID); err != nil {
		return err
	}
	return tx.Bucket(bucketSpaces).Delete(spaceMembershipKey(spaceID, memoryID))
}

func loadIndexedSpaceMemoryIDs(tx *bbolt.Tx, spaceID string) ([]string, error) {
	b := tx.Bucket(bucketSpaces)
	c := b.Cursor()
	prefix := spaceMembershipPrefix(spaceID)
	ids := make([]string, 0)
	for key, _ := c.Seek(prefix); key != nil && bytes.HasPrefix(key, prefix); key, _ = c.Next() {
		ids = append(ids, string(key[len(prefix):]))
	}
	slices.Sort(ids)
	return ids, nil
}

func migrateLegacySpaceMemoryIDs(tx *bbolt.Tx, spaceID string) error {
	b := tx.Bucket(bucketSpaces)
	legacyKey := []byte(spaceID)
	raw := b.Get(legacyKey)
	if raw == nil {
		return nil
	}

	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return err
	}
	for _, id := range ids {
		if err := b.Put(spaceMembershipKey(spaceID, id), []byte{1}); err != nil {
			return err
		}
	}
	return b.Delete(legacyKey)
}

func spaceMembershipPrefix(spaceID string) []byte {
	return []byte(spaceID + "\x00")
}

func spaceMembershipKey(spaceID string, memoryID string) []byte {
	return []byte(spaceID + "\x00" + memoryID)
}
