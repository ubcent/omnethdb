package bolt

import (
	"testing"

	"omnethdb/internal/memory"
)

func TestGetLineageReturnsFullHistoryIncludingForgottenAndRevived(t *testing.T) {
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
	if err := store.Forget(v2.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "retired"); err != nil {
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
		t.Fatalf("expected 3 lineage entries, got %#v", lineage)
	}
	if lineage[0].ID != v1.ID || lineage[1].ID != v2.ID || lineage[2].ID != v3.ID {
		t.Fatalf("expected lineage ordered by version, got %#v", lineage)
	}
	if !lineage[1].IsForgotten {
		t.Fatalf("expected forgotten middle version to remain visible, got %#v", lineage[1])
	}
}

func TestGetLineageRejectsMissingRoot(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	if _, err := store.GetLineage("missing-root"); err != memory.ErrMemoryNotFound {
		t.Fatalf("expected ErrMemoryNotFound, got %v", err)
	}
}
