package bolt

import (
	"errors"
	. "omnethdb/internal/memory"
	"testing"

	bbolt "go.etcd.io/bbolt"
)

func TestReviveCreatesNewLatestForInactiveLineage(t *testing.T) {
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

	revived, err := store.Revive(v1.ID, ReviveInput{
		Content:    "deploy takes ~7 min after rollback",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 0.95,
		Metadata:   map[string]any{"source": "incident-review"},
	})
	if err != nil {
		t.Fatalf("Revive returned unexpected error: %v", err)
	}

	if revived.ParentID == nil || *revived.ParentID != v2.ID {
		t.Fatalf("expected revived ParentID %q, got %#v", v2.ID, revived.ParentID)
	}
	if revived.RootID == nil || *revived.RootID != v1.ID {
		t.Fatalf("expected revived RootID %q, got %#v", v1.ID, revived.RootID)
	}
	if revived.Version != 3 {
		t.Fatalf("expected revived version 3, got %d", revived.Version)
	}
	if !revived.IsLatest {
		t.Fatal("expected revived memory to be latest")
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedForgotten, err := loadMemoryForForget(tx, v2.ID)
		if err != nil {
			return err
		}
		if !persistedForgotten.IsForgotten {
			t.Fatal("expected forgotten predecessor to remain forgotten")
		}

		latest := tx.Bucket(bucketLatest).Get([]byte(v1.ID))
		if string(latest) != revived.ID {
			t.Fatalf("expected latest entry to point to revived memory, got %q", string(latest))
		}

		ids := loadSpaceMemoryIDs(tx, v1.SpaceID)
		if len(ids) != 1 || ids[0] != revived.ID {
			t.Fatalf("expected revived memory to be the only live ID, got %#v", ids)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestReviveRejectsActiveLineage(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	root, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "schema uses ledger entries",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	_, err = store.Revive(root.ID, ReviveInput{
		Content:    "reactivation attempt",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if !errors.Is(err, ErrLineageActive) {
		t.Fatalf("expected ErrLineageActive, got %v", err)
	}
}

func TestReviveRejectsKindChangeAgainstRootLineage(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	root, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "observed timeout during deploy",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.7,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(root.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "retired observation"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	_, err = store.Revive(root.ID, ReviveInput{
		Content:    "attempt to bypass promotion",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if !errors.Is(err, ErrUpdateAcrossKinds) {
		t.Fatalf("expected ErrUpdateAcrossKinds, got %v", err)
	}
}

func TestReviveRejectsDerivedLineage(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	s1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payment retries are capped at 3",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	s2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "circuit breaker opens after consecutive failures",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	derived, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payment reliability depends on retries and breaker thresholds",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:analyst-1", Kind: ActorAgent},
		Confidence: 0.9,
		SourceIDs:  []string{s1.ID, s2.ID},
		Rationale:  "joint operational inference",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(derived.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "retired synthesis"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	_, err = store.Revive(derived.ID, ReviveInput{
		Content:    "reconstructed synthesis",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:analyst-1", Kind: ActorAgent},
		Confidence: 0.9,
	})
	if !errors.Is(err, ErrReviveDerivedUnsupported) {
		t.Fatalf("expected ErrReviveDerivedUnsupported, got %v", err)
	}
}
