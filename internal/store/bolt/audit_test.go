package bolt

import (
	"testing"
	"time"

	"omnethdb/internal/memory"

	bbolt "go.etcd.io/bbolt"
)

func TestLifecycleOperationsEmitAuditEntries(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	root, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~5 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(root.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "retired fact"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	revived, err := store.Revive(root.ID, memory.ReviveInput{
		Content:    "deploy takes ~7 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Revive returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		entries, err := loadAuditEntries(tx, "repo:company/app", zeroTime())
		if err != nil {
			return err
		}
		if len(entries) != 3 {
			t.Fatalf("expected 3 audit entries, got %#v", entries)
		}
		if entries[0].Operation != "remember" || len(entries[0].MemoryIDs) != 1 || entries[0].MemoryIDs[0] != root.ID {
			t.Fatalf("unexpected remember audit entry: %#v", entries[0])
		}
		if entries[1].Operation != "forget" || entries[1].Reason != "retired fact" || entries[1].MemoryIDs[0] != root.ID {
			t.Fatalf("unexpected forget audit entry: %#v", entries[1])
		}
		if entries[2].Operation != "revive" || len(entries[2].MemoryIDs) != 2 || entries[2].MemoryIDs[1] != revived.ID {
			t.Fatalf("unexpected revive audit entry: %#v", entries[2])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("View returned unexpected error: %v", err)
	}
}

func TestForgetStoresTargetedForgetRecord(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	mem, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "cleanup candidate",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(mem.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		record, err := loadForgetRecord(tx, mem.ID)
		if err != nil {
			return err
		}
		if record.MemoryID != mem.ID || record.SpaceID != mem.SpaceID {
			t.Fatalf("unexpected forget record identity: %#v", record)
		}
		if record.Actor.ID != "user:alice" || record.Reason != "cleanup" {
			t.Fatalf("unexpected forget record payload: %#v", record)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("View returned unexpected error: %v", err)
	}
}

func zeroTime() (t time.Time) { return }
