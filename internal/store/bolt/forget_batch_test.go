package bolt

import (
	"testing"

	"omnethdb/internal/memory"
)

func TestForgetBatchForgetsMultipleMemoriesAtomically(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	first, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "duplicate fact one",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	second, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "duplicate fact two",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	if err := store.ForgetBatch([]string{first.ID, second.ID}, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "duplicate cleanup"); err != nil {
		t.Fatalf("ForgetBatch returned unexpected error: %v", err)
	}

	listed, err := store.ListMemories(memory.ListMemoriesRequest{
		SpaceIDs: []string{"repo:company/app"},
		Kinds:    []memory.MemoryKind{memory.KindStatic},
	})
	if err != nil {
		t.Fatalf("ListMemories returned unexpected error: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected no live static memories after batch forget, got %#v", listed)
	}
}
