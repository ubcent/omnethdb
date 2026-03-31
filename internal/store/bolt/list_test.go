package bolt

import (
	"testing"
	"time"

	"omnethdb/internal/memory"
)

func TestListMemoriesReturnsDeterministicLiveStaticMemories(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	staticOne, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "adr one",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("staticOne Remember returned unexpected error: %v", err)
	}
	time.Sleep(time.Millisecond)
	staticTwo, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "adr two",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("staticTwo Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "incident note",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("episodic Remember returned unexpected error: %v", err)
	}

	listed, err := store.ListMemories(memory.ListMemoriesRequest{
		SpaceIDs: []string{"repo:company/app"},
		Kinds:    []memory.MemoryKind{memory.KindStatic},
	})
	if err != nil {
		t.Fatalf("ListMemories returned unexpected error: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected exactly two static memories, got %#v", listed)
	}
	if listed[0].ID != staticOne.ID || listed[1].ID != staticTwo.ID {
		t.Fatalf("expected deterministic created-at ordering, got %#v", listed)
	}
}
