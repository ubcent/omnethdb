package bolt

import (
	"errors"
	"testing"
	"time"

	"omnethdb/internal/memory"
)

func TestConcurrentUpdatesProduceOneWinnerAndOneConflict(t *testing.T) {
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
		t.Fatalf("initial Remember returned unexpected error: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	results := make(chan *memory.Memory, 2)

	runUpdate := func(content string) {
		<-start
		mem, err := store.Remember(memory.MemoryInput{
			SpaceID:    root.SpaceID,
			Content:    content,
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
			IfLatestID: &root.ID,
			Relations:  memory.MemoryRelations{Updates: []string{root.ID}},
		})
		errs <- err
		results <- mem
	}

	go runUpdate("deploy takes ~9 min")
	go runUpdate("deploy takes ~11 min")
	close(start)

	var conflictCount int
	var success *memory.Memory
	for i := 0; i < 2; i++ {
		err := <-errs
		mem := <-results
		switch {
		case err == nil:
			success = mem
		case errors.Is(err, memory.ErrConflict):
			conflictCount++
		default:
			t.Fatalf("expected success or ErrConflict, got err=%v mem=%#v", err, mem)
		}
	}

	if success == nil {
		t.Fatal("expected one successful contested update")
	}
	if conflictCount != 1 {
		t.Fatalf("expected exactly one conflict, got %d", conflictCount)
	}

	lineage, err := store.GetLineage(root.ID)
	if err != nil {
		t.Fatalf("GetLineage returned unexpected error: %v", err)
	}
	if len(lineage) != 2 {
		t.Fatalf("expected lineage length 2 after contention, got %d", len(lineage))
	}

	var latestCount int
	for _, mem := range lineage {
		if mem.IsLatest && !mem.IsForgotten {
			latestCount++
			if mem.ID != success.ID {
				t.Fatalf("expected contested winner %q to be latest, got %q", success.ID, mem.ID)
			}
		}
	}
	if latestCount != 1 {
		t.Fatalf("expected exactly one live latest after contention, got %d", latestCount)
	}
}

func TestRecallObservesCoherentSnapshotsDuringSequentialUpdates(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	current, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~5 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("initial Remember returned unexpected error: %v", err)
	}

	done := make(chan struct{})
	writerErr := make(chan error, 1)

	go func() {
		defer close(done)

		latest := current
		for i := 0; i < 12; i++ {
			next, err := store.Remember(memory.MemoryInput{
				SpaceID:    latest.SpaceID,
				Content:    latest.Content + " updated",
				Kind:       latest.Kind,
				Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
				Confidence: 1.0,
				IfLatestID: &latest.ID,
				Relations:  memory.MemoryRelations{Updates: []string{latest.ID}},
			})
			if err != nil {
				writerErr <- err
				return
			}
			latest = next
			time.Sleep(2 * time.Millisecond)
		}

		writerErr <- nil
	}()

	for {
		select {
		case <-done:
			if err := <-writerErr; err != nil {
				t.Fatalf("writer returned unexpected error: %v", err)
			}
			return
		default:
			scored, err := store.Recall(memory.RecallRequest{
				SpaceIDs: []string{"repo:company/app"},
				Query:    "deploy",
				TopK:     10,
			})
			if err != nil {
				t.Fatalf("Recall returned unexpected error during concurrent updates: %v", err)
			}
			if len(scored) != 1 {
				t.Fatalf("expected Recall to expose exactly one live memory snapshot, got %d", len(scored))
			}
			if !scored[0].IsLatest {
				t.Fatal("expected Recall snapshot to contain a latest memory only")
			}
			if scored[0].IsForgotten {
				t.Fatal("expected Recall snapshot to exclude forgotten memories")
			}
		}
	}
}
