package bolt

import (
	"encoding/json"
	"fmt"
	"omnethdb/internal/memory"
	"sort"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func writeAuditEntry(tx *bbolt.Tx, entry memory.AuditEntry) error {
	entry.Timestamp = entry.Timestamp.UTC()
	key := fmt.Sprintf("%020d:%s", entry.Timestamp.UnixNano(), newMemoryID())
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return tx.Bucket(bucketAuditLog).Put([]byte(key), encoded)
}

func writeForgetRecord(tx *bbolt.Tx, record memory.ForgetRecord) error {
	record.Timestamp = record.Timestamp.UTC()
	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return tx.Bucket(bucketForgetLog).Put([]byte(record.MemoryID), encoded)
}

func loadAuditEntries(tx *bbolt.Tx, spaceID string, since time.Time) ([]memory.AuditEntry, error) {
	entries := make([]memory.AuditEntry, 0)
	b := tx.Bucket(bucketAuditLog)
	c := b.Cursor()
	for _, raw := c.First(); raw != nil; _, raw = c.Next() {
		var entry memory.AuditEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, err
		}
		if spaceID != "" && entry.SpaceID != spaceID {
			continue
		}
		if !since.IsZero() && entry.Timestamp.Before(since) {
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	return entries, nil
}

func loadForgetRecord(tx *bbolt.Tx, memoryID string) (*memory.ForgetRecord, error) {
	raw := tx.Bucket(bucketForgetLog).Get([]byte(memoryID))
	if raw == nil {
		return nil, memory.ErrMemoryNotFound
	}
	var record memory.ForgetRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return &record, nil
}
