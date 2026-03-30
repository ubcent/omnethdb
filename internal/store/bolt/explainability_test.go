package bolt

import (
	"testing"
	"time"

	"omnethdb/internal/memory"

	bbolt "go.etcd.io/bbolt"
)

func TestInspectionAndAuditCanExplainCurrentCorpusState(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	v1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~5 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("v1 Remember returned unexpected error: %v", err)
	}
	v2, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~12 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
		Relations:  memory.MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("v2 Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(v2.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "retired fact"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}
	v3, err := store.Revive(v1.ID, memory.ReviveInput{
		Content:    "deploy takes ~7 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Revive returned unexpected error: %v", err)
	}

	lineage, err := store.GetLineage(v1.ID)
	if err != nil {
		t.Fatalf("GetLineage returned unexpected error: %v", err)
	}
	if len(lineage) != 3 {
		t.Fatalf("expected full lineage history, got %#v", lineage)
	}
	if lineage[0].ID != v1.ID || lineage[1].ID != v2.ID || lineage[2].ID != v3.ID {
		t.Fatalf("expected ordered lineage v1 -> v2 -> v3, got %#v", lineage)
	}
	if lineage[1].IsLatest || !lineage[1].IsForgotten {
		t.Fatalf("expected middle version to explain retirement, got %#v", lineage[1])
	}
	if !lineage[2].IsLatest || lineage[2].IsForgotten {
		t.Fatalf("expected revived version to explain current live state, got %#v", lineage[2])
	}

	audit, err := store.GetAuditLog("repo:company/app", time.Time{})
	if err != nil {
		t.Fatalf("GetAuditLog returned unexpected error: %v", err)
	}
	if len(audit) != 4 {
		t.Fatalf("expected write/update/forget/revive audit trail, got %#v", audit)
	}
	if audit[0].Operation != "remember" || audit[1].Operation != "remember" || audit[2].Operation != "forget" || audit[3].Operation != "revive" {
		t.Fatalf("unexpected audit operation flow: %#v", audit)
	}
	if len(audit[1].MemoryIDs) != 2 || audit[1].MemoryIDs[0] != v1.ID || audit[1].MemoryIDs[1] != v2.ID {
		t.Fatalf("expected update remember audit entry to link v1 -> v2, got %#v", audit[1])
	}
	if len(audit[3].MemoryIDs) != 2 || audit[3].MemoryIDs[0] != v2.ID || audit[3].MemoryIDs[1] != v3.ID {
		t.Fatalf("expected revive audit entry to link v2 -> v3, got %#v", audit[3])
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		record, err := loadForgetRecord(tx, v2.ID)
		if err != nil {
			return err
		}
		if record.MemoryID != v2.ID || record.Reason != "retired fact" {
			t.Fatalf("unexpected forget record: %#v", record)
		}
		if audit[2].Operation != "forget" || audit[2].MemoryIDs[0] != record.MemoryID || audit[2].Reason != record.Reason {
			t.Fatalf("expected forget log and audit log to agree, audit=%#v record=%#v", audit[2], record)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("View returned unexpected error: %v", err)
	}
}
