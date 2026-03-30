package bolt

import (
	"errors"
	. "omnethdb/internal/memory"
	"testing"

	bbolt "go.etcd.io/bbolt"
)

func TestForgetLatestDeactivatesLineageWithoutRestoringPreviousVersion(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	v1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~5 min",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("initial Remember returned unexpected error: %v", err)
	}

	v2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~12 min after middleware",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("update Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(v2.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "no longer true"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedV1, err := loadMemoryForForget(tx, v1.ID)
		if err != nil {
			return err
		}
		persistedV2, err := loadMemoryForForget(tx, v2.ID)
		if err != nil {
			return err
		}

		if persistedV1.IsLatest {
			t.Fatal("expected previous version to remain non-latest after forgetting latest")
		}
		if !persistedV2.IsForgotten {
			t.Fatal("expected forgotten latest to be marked forgotten")
		}
		if persistedV2.IsLatest {
			t.Fatal("expected forgotten latest to be removed from latest state")
		}

		latest := tx.Bucket(bucketLatest).Get([]byte(v1.ID))
		if latest != nil {
			t.Fatalf("expected latest entry to be cleared, got %q", string(latest))
		}

		ids := loadSpaceMemoryIDs(tx, v1.SpaceID)
		if len(ids) != 0 {
			t.Fatalf("expected inactive lineage to disappear from live corpus, got %#v", ids)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestForgetNonLatestKeepsCurrentLatestLive(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	v1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "schema uses ledger entries",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("initial Remember returned unexpected error: %v", err)
	}

	v2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "schema uses append-only ledger entries",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("update Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(v1.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "superseded historical version"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedV1, err := loadMemoryForForget(tx, v1.ID)
		if err != nil {
			return err
		}
		persistedV2, err := loadMemoryForForget(tx, v2.ID)
		if err != nil {
			return err
		}

		if !persistedV1.IsForgotten {
			t.Fatal("expected non-latest memory to be marked forgotten")
		}
		if !persistedV2.IsLatest {
			t.Fatal("expected current latest to remain latest")
		}

		latest := tx.Bucket(bucketLatest).Get([]byte(v1.ID))
		if string(latest) != v2.ID {
			t.Fatalf("expected latest pointer to remain on current latest, got %q", string(latest))
		}

		ids := loadSpaceMemoryIDs(tx, v1.SpaceID)
		if len(ids) != 1 || ids[0] != v2.ID {
			t.Fatalf("expected current latest to remain live, got %#v", ids)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestForgetRejectsMissingMemory(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	err := store.Forget("missing-memory", Actor{ID: "user:alice", Kind: ActorHuman}, "cleanup")
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("expected ErrMemoryNotFound, got %v", err)
	}
}

func TestForgetMarksAffectedLiveDerivedMemoryAsOrphanedAndKeepsItLive(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	source1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "checkout retries are capped at 3",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source1 Remember returned unexpected error: %v", err)
	}
	source2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "breaker opens after repeated payment failures",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source2 Remember returned unexpected error: %v", err)
	}
	derived, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payment resilience depends on retries and breaker thresholds",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:analyst-1", Kind: ActorAgent},
		Confidence: 0.9,
		SourceIDs:  []string{source1.ID, source2.ID},
		Rationale:  "joint operational inference",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(source1.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "retired source evidence"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedDerived, err := loadMemory(tx, derived.ID)
		if err != nil {
			return err
		}
		if !persistedDerived.HasOrphanedSources {
			t.Fatal("expected affected derived memory to be marked orphaned")
		}
		if !persistedDerived.IsLatest {
			t.Fatal("expected affected derived memory to remain live")
		}

		ids := loadSpaceMemoryIDs(tx, derived.SpaceID)
		if !containsSourceID(ids, derived.ID) {
			t.Fatalf("expected derived memory to remain in live corpus, got %#v", ids)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestForgetMarksOnlyLiveDerivedVersionAsOrphaned(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	source1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "checkout retries are capped at 3",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source1 Remember returned unexpected error: %v", err)
	}
	source2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "breaker opens after repeated payment failures",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source2 Remember returned unexpected error: %v", err)
	}
	derivedV1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "first synthesis",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:analyst-1", Kind: ActorAgent},
		Confidence: 0.8,
		SourceIDs:  []string{source1.ID, source2.ID},
		Rationale:  "initial inference",
	})
	if err != nil {
		t.Fatalf("derivedV1 Remember returned unexpected error: %v", err)
	}
	derivedV2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "refined synthesis",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:analyst-1", Kind: ActorAgent},
		Confidence: 0.9,
		SourceIDs:  []string{source1.ID, source2.ID},
		Rationale:  "refined inference",
		Relations:  MemoryRelations{Updates: []string{derivedV1.ID}},
	})
	if err != nil {
		t.Fatalf("derivedV2 Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(source1.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "retired source evidence"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedV1, err := loadMemory(tx, derivedV1.ID)
		if err != nil {
			return err
		}
		persistedV2, err := loadMemory(tx, derivedV2.ID)
		if err != nil {
			return err
		}
		if persistedV1.HasOrphanedSources {
			t.Fatal("expected superseded derived version to remain unmarked")
		}
		if !persistedV2.HasOrphanedSources {
			t.Fatal("expected latest derived version to be marked orphaned")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestOrphanedSourceFlagIsNotClearedBySourceRevive(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	source1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "checkout retries are capped at 3",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source1 Remember returned unexpected error: %v", err)
	}
	source2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "breaker opens after repeated payment failures",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source2 Remember returned unexpected error: %v", err)
	}
	derived, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payment resilience depends on retries and breaker thresholds",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:analyst-1", Kind: ActorAgent},
		Confidence: 0.9,
		SourceIDs:  []string{source1.ID, source2.ID},
		Rationale:  "joint operational inference",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(source1.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "retired source evidence"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}
	if _, err := store.Revive(source1.ID, ReviveInput{
		Content:    "checkout retries still capped at 3",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("Revive returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedDerived, err := loadMemory(tx, derived.ID)
		if err != nil {
			return err
		}
		if !persistedDerived.HasOrphanedSources {
			t.Fatal("expected orphaned flag to remain set after source revive")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}
