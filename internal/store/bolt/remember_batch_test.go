package bolt

import (
	"testing"

	"omnethdb/internal/memory"
)

func TestRememberBatchWritesAllMemoriesInOneTransaction(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	written, err := store.RememberBatch([]memory.MemoryInput{
		{
			SpaceID:    "repo:company/app",
			Content:    "adr one",
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "adr two",
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
		},
	})
	if err != nil {
		t.Fatalf("RememberBatch returned unexpected error: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("expected two written memories, got %#v", written)
	}

	listed, err := store.ListMemories(memory.ListMemoriesRequest{
		SpaceIDs: []string{"repo:company/app"},
		Kinds:    []memory.MemoryKind{memory.KindStatic},
	})
	if err != nil {
		t.Fatalf("ListMemories returned unexpected error: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected two committed static memories, got %#v", listed)
	}
}

func TestRememberBatchIsAtomicOnFailure(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	_, err := store.RememberBatch([]memory.MemoryInput{
		{
			SpaceID:    "repo:company/app",
			Content:    "adr one",
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "",
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
		},
	})
	if err == nil {
		t.Fatal("expected RememberBatch to fail on invalid second input")
	}

	listed, listErr := store.ListMemories(memory.ListMemoriesRequest{
		SpaceIDs: []string{"repo:company/app"},
	})
	if listErr != nil {
		t.Fatalf("ListMemories returned unexpected error: %v", listErr)
	}
	if len(listed) != 0 {
		t.Fatalf("expected batch failure to leave no committed memories, got %#v", listed)
	}
}
