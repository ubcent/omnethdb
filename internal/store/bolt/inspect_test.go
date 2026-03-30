package bolt

import (
	"testing"
	"time"

	"omnethdb/internal/memory"
)

func TestGetRelatedTraversesExplicitRelationsIncludingHistoricalNodes(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	base, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments schema",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("base Remember returned unexpected error: %v", err)
	}
	ext1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "ledger entries",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
		Relations:  memory.MemoryRelations{Extends: []string{base.ID}},
	})
	if err != nil {
		t.Fatalf("ext1 Remember returned unexpected error: %v", err)
	}
	ext2, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "append-only ledger entries",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
		Relations:  memory.MemoryRelations{Extends: []string{ext1.ID}},
	})
	if err != nil {
		t.Fatalf("ext2 Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(ext1.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "historical cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	related, err := store.GetRelated(ext2.ID, memory.RelationExtends, 2)
	if err != nil {
		t.Fatalf("GetRelated returned unexpected error: %v", err)
	}
	if len(related) != 2 {
		t.Fatalf("expected two traversed related memories, got %#v", related)
	}
	if related[0].ID != ext1.ID || related[1].ID != base.ID {
		t.Fatalf("expected traversal order ext1 -> base, got %#v", related)
	}
	if !related[0].IsForgotten {
		t.Fatalf("expected forgotten node to remain traversable, got %#v", related[0])
	}
}

func TestGetAuditLogReturnsSpaceScopedChronologicalHistory(t *testing.T) {
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
	if _, err := store.Revive(root.ID, memory.ReviveInput{
		Content:    "deploy takes ~7 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("Revive returned unexpected error: %v", err)
	}

	audit, err := store.GetAuditLog("repo:company/app", time.Time{})
	if err != nil {
		t.Fatalf("GetAuditLog returned unexpected error: %v", err)
	}
	if len(audit) != 3 {
		t.Fatalf("expected three audit entries, got %#v", audit)
	}
	if audit[0].Operation != "remember" || audit[1].Operation != "forget" || audit[2].Operation != "revive" {
		t.Fatalf("expected chronological audit operations, got %#v", audit)
	}

	since := audit[1].Timestamp
	filtered, err := store.GetAuditLog("repo:company/app", since)
	if err != nil {
		t.Fatalf("GetAuditLog returned unexpected error: %v", err)
	}
	if len(filtered) != 2 || filtered[0].Operation != "forget" || filtered[1].Operation != "revive" {
		t.Fatalf("expected since-filtered audit history, got %#v", filtered)
	}
}
