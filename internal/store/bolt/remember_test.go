package bolt

import (
	"encoding/json"
	"errors"
	"fmt"
	. "omnethdb/internal/memory"
	. "omnethdb/internal/policy"
	"path/filepath"
	"testing"

	bbolt "go.etcd.io/bbolt"
)

func TestRememberCreatesRootMemoryAsLatest(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	mem, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~5 min",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	if mem.Version != 1 {
		t.Fatalf("expected root version 1, got %d", mem.Version)
	}
	if !mem.IsLatest {
		t.Fatal("expected root memory to be latest")
	}
	if mem.ParentID != nil {
		t.Fatalf("expected root ParentID to be nil, got %v", *mem.ParentID)
	}
	if mem.RootID != nil {
		t.Fatalf("expected root RootID to be nil, got %v", *mem.RootID)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		latest := tx.Bucket(bucketLatest).Get([]byte(mem.ID))
		if string(latest) != mem.ID {
			t.Fatalf("expected latest bucket to point root to itself, got %q", string(latest))
		}
		ids := loadSpaceMemoryIDs(tx, mem.SpaceID)
		if len(ids) != 1 || ids[0] != mem.ID {
			t.Fatalf("unexpected live space ids: %#v", ids)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestRememberUpdateSwitchesLatestWithinLineage(t *testing.T) {
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
		Content:    "deploy takes ~12 min after auth middleware added",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations: MemoryRelations{
			Updates: []string{v1.ID},
		},
	})
	if err != nil {
		t.Fatalf("update Remember returned unexpected error: %v", err)
	}

	if v2.Version != 2 {
		t.Fatalf("expected updated version 2, got %d", v2.Version)
	}
	if v2.ParentID == nil || *v2.ParentID != v1.ID {
		t.Fatalf("expected ParentID %q, got %#v", v1.ID, v2.ParentID)
	}
	if v2.RootID == nil || *v2.RootID != v1.ID {
		t.Fatalf("expected RootID %q, got %#v", v1.ID, v2.RootID)
	}
	if !v2.IsLatest {
		t.Fatal("expected updated memory to be latest")
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedV1, err := loadMemory(tx, v1.ID)
		if err != nil {
			return err
		}
		if persistedV1.IsLatest {
			t.Fatal("expected previous version to be marked non-latest")
		}

		latest := tx.Bucket(bucketLatest).Get([]byte(v1.ID))
		if string(latest) != v2.ID {
			t.Fatalf("expected latest bucket to point to v2, got %q", string(latest))
		}

		ids := loadSpaceMemoryIDs(tx, v1.SpaceID)
		if len(ids) != 1 || ids[0] != v2.ID {
			t.Fatalf("unexpected live space ids after update: %#v", ids)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestRememberRejectsUpdateAgainstNonLatestVersion(t *testing.T) {
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

	if _, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~12 min",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	}); err != nil {
		t.Fatalf("first update returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "stale rewrite of v1",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	})
	if !errors.Is(err, ErrUpdateTargetNotLatest) {
		t.Fatalf("expected ErrUpdateTargetNotLatest, got %v", err)
	}
}

func TestRememberAcceptsUpdateWhenIfLatestIDMatchesCurrentLatest(t *testing.T) {
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
		Content:    "deploy takes ~9 min",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		IfLatestID: &v1.ID,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("update with matching IfLatestID returned unexpected error: %v", err)
	}
	if v2.ParentID == nil || *v2.ParentID != v1.ID {
		t.Fatalf("expected ParentID %q, got %#v", v1.ID, v2.ParentID)
	}
}

func TestRememberRejectsUpdateConflictWhenIfLatestIDIsStale(t *testing.T) {
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
		Content:    "deploy takes ~9 min",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("first update returned unexpected error: %v", err)
	}

	before := captureSpaceState(t, store, "repo:company/app")

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "stale client retry",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		IfLatestID: &v1.ID,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	after := captureSpaceState(t, store, "repo:company/app")
	if before != after {
		t.Fatalf("expected no partial state change on conflict, before=%v after=%v", before, after)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		latest := loadLatest(tx, v1.ID)
		if latest != v2.ID {
			t.Fatalf("expected latest to remain %q, got %q", v2.ID, latest)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestRememberRejectsMultipleUpdateTargets(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	_, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "invalid multi-update write",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations: MemoryRelations{
			Updates: []string{"a", "b"},
		},
	})
	if !errors.Is(err, ErrInvalidRelations) {
		t.Fatalf("expected ErrInvalidRelations, got %v", err)
	}
}

func TestRememberStoresValidExtendsRelationWithoutChangingTargetLiveness(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	base, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments schema uses ledger entries",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("base Remember returned unexpected error: %v", err)
	}

	ext, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "ledger entries are append-only",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Extends: []string{base.ID}},
	})
	if err != nil {
		t.Fatalf("extends Remember returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedBase, err := loadMemory(tx, base.ID)
		if err != nil {
			return err
		}
		if !persistedBase.IsLatest {
			t.Fatal("expected extends target to remain latest")
		}

		raw := tx.Bucket(bucketRelations).Get([]byte(ext.ID + ":extends"))
		if raw == nil {
			t.Fatal("expected extends relation to be stored")
		}

		var related []string
		if err := json.Unmarshal(raw, &related); err != nil {
			return err
		}
		if len(related) != 1 || related[0] != base.ID {
			t.Fatalf("unexpected stored extends relation: %#v", related)
		}

		ids := loadSpaceMemoryIDs(tx, base.SpaceID)
		if len(ids) != 2 {
			t.Fatalf("expected both memories to remain live, got %#v", ids)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}
}

func TestRememberRejectsExtendsAcrossSpaces(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)
	ensureRememberTestSpace(t, store, "repo:company/other")

	foreign, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/other",
		Content:    "orders schema is partitioned",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("foreign Remember returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments mentions orders context",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Extends: []string{foreign.ID}},
	})
	if !errors.Is(err, ErrExtendsAcrossSpaces) {
		t.Fatalf("expected ErrExtendsAcrossSpaces, got %v", err)
	}
}

func TestRememberRejectsExtendsTargetThatIsNotLatest(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	v1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use ledger entries",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("initial Remember returned unexpected error: %v", err)
	}

	if _, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use ledger entries with snapshots",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	}); err != nil {
		t.Fatalf("update Remember returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "extra context for stale record",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Extends: []string{v1.ID}},
	})
	if !errors.Is(err, ErrExtendsTargetNotLatest) {
		t.Fatalf("expected ErrExtendsTargetNotLatest, got %v", err)
	}
}

func TestRememberRejectsExtendsTargetNotFound(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	_, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "invalid extends attempt",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Extends: []string{"missing-target"}},
	})
	if !errors.Is(err, ErrExtendsTargetNotFound) {
		t.Fatalf("expected ErrExtendsTargetNotFound, got %v", err)
	}
}

func TestRememberRejectsUnauthorizedStaticWrite(t *testing.T) {
	t.Parallel()

	store := newRememberTestStoreWithPolicy(t, "repo:company/app", SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters: WritersPolicy{
			AllowHuman:     true,
			AllowSystem:    true,
			AllowAllAgents: true,
		},
		StaticWriters: WritersPolicy{
			AllowHuman: true,
		},
		PromotePolicy: WritersPolicy{
			AllowHuman: true,
		},
	})

	_, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "agents should not write static facts directly here",
		Kind:       KindStatic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
	})
	if !errors.Is(err, ErrPolicyViolation) {
		t.Fatalf("expected ErrPolicyViolation, got %v", err)
	}
}

func TestRememberRejectsUpdateAcrossSpacesWithoutPartialState(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)
	ensureRememberTestSpace(t, store, "repo:company/other")

	foreign, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/other",
		Content:    "foreign memory",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("foreign Remember returned unexpected error: %v", err)
	}

	beforeAppState := captureSpaceState(t, store, "repo:company/app")
	beforeOtherState := captureSpaceState(t, store, "repo:company/other")

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "illegal cross-space update",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{foreign.ID}},
	})
	if !errors.Is(err, ErrUpdateAcrossSpaces) {
		t.Fatalf("expected ErrUpdateAcrossSpaces, got %v", err)
	}

	afterAppState := captureSpaceState(t, store, "repo:company/app")
	afterOtherState := captureSpaceState(t, store, "repo:company/other")
	if beforeAppState != afterAppState || beforeOtherState != afterOtherState {
		t.Fatalf("expected no partial state change, before=(%v,%v) after=(%v,%v)", beforeAppState, beforeOtherState, afterAppState, afterOtherState)
	}
}

func TestRememberRejectsUpdateAcrossKindsWithoutPartialState(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	staticMem, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("static Remember returned unexpected error: %v", err)
	}

	before := captureSpaceState(t, store, "repo:company/app")

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "illegal episodic replacement",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{staticMem.ID}},
	})
	if !errors.Is(err, ErrUpdateAcrossKinds) {
		t.Fatalf("expected ErrUpdateAcrossKinds, got %v", err)
	}

	after := captureSpaceState(t, store, "repo:company/app")
	if before != after {
		t.Fatalf("expected no partial state change, before=%v after=%v", before, after)
	}
}

func TestRememberAllowsPromotionOnlyWithPromotePolicy(t *testing.T) {
	t.Parallel()

	restricted := newRememberTestStoreWithPolicy(t, "repo:company/app", SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters: WritersPolicy{
			AllowHuman:     true,
			AllowSystem:    true,
			AllowAllAgents: true,
		},
		StaticWriters: WritersPolicy{
			AllowHuman:     true,
			AllowAllAgents: true,
		},
		PromotePolicy: WritersPolicy{
			AllowHuman: true,
		},
	})

	episodic, err := restricted.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "pagination looked cursor-based in code review",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("episodic Remember returned unexpected error: %v", err)
	}

	_, err = restricted.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "pagination is cursor-based",
		Kind:       KindStatic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
		Relations:  MemoryRelations{Updates: []string{episodic.ID}},
	})
	if !errors.Is(err, ErrPolicyViolation) {
		t.Fatalf("expected ErrPolicyViolation for unauthorized promotion, got %v", err)
	}

	allowed := newRememberTestStoreWithPolicy(t, "repo:company/app", SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters: WritersPolicy{
			AllowHuman:     true,
			AllowSystem:    true,
			AllowAllAgents: true,
		},
		StaticWriters: WritersPolicy{
			AllowHuman:     true,
			AllowAllAgents: true,
		},
		PromotePolicy: WritersPolicy{
			AllowAllAgents: true,
		},
	})

	episodic, err = allowed.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "pagination looked cursor-based in code review",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("episodic Remember returned unexpected error: %v", err)
	}

	promoted, err := allowed.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "pagination is cursor-based",
		Kind:       KindStatic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
		Relations:  MemoryRelations{Updates: []string{episodic.ID}},
	})
	if err != nil {
		t.Fatalf("authorized promotion returned unexpected error: %v", err)
	}
	if promoted.Kind != KindStatic {
		t.Fatalf("expected promoted memory to be static, got %v", promoted.Kind)
	}
}

func TestRememberRejectsRootWriteWhenCorpusLimitReached(t *testing.T) {
	t.Parallel()

	store := newRememberTestStoreWithPolicy(t, "repo:company/app", SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters: WritersPolicy{
			AllowHuman:     true,
			AllowSystem:    true,
			AllowAllAgents: true,
		},
		StaticWriters: WritersPolicy{
			AllowHuman: true,
		},
		PromotePolicy: WritersPolicy{
			AllowHuman: true,
		},
		MaxStaticMemories:   1,
		MaxEpisodicMemories: 100,
	})

	if _, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "first static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("first root write returned unexpected error: %v", err)
	}

	_, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "second static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if !errors.Is(err, ErrCorpusLimit) {
		t.Fatalf("expected ErrCorpusLimit, got %v", err)
	}
}

func TestRememberAllowsSameKindUpdateAtCorpusLimit(t *testing.T) {
	t.Parallel()

	store := newRememberTestStoreWithPolicy(t, "repo:company/app", SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		StaticWriters: WritersPolicy{
			AllowHuman: true,
		},
		PromotePolicy: WritersPolicy{
			AllowHuman: true,
		},
		MaxStaticMemories:   1,
		MaxEpisodicMemories: 100,
	})

	v1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "first static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("initial write returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "updated static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("same-kind update at corpus limit returned unexpected error: %v", err)
	}
}

func TestRememberRejectsPromotionWhenStaticCorpusLimitReached(t *testing.T) {
	t.Parallel()

	store := newRememberTestStoreWithPolicy(t, "repo:company/app", SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters: WritersPolicy{
			AllowHuman: true,
		},
		StaticWriters: WritersPolicy{
			AllowHuman: true,
		},
		PromotePolicy: WritersPolicy{
			AllowHuman: true,
		},
		MaxStaticMemories:   1,
		MaxEpisodicMemories: 100,
	})

	if _, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "existing static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("static seed write returned unexpected error: %v", err)
	}

	episodic, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "episodic observation",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("episodic seed write returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "promoted static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{episodic.ID}},
	})
	if !errors.Is(err, ErrCorpusLimit) {
		t.Fatalf("expected ErrCorpusLimit for promotion, got %v", err)
	}
}

func TestRememberCreatesDerivedMemoryWithProvenance(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	src1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments migration failed in prod",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("source 1 Remember returned unexpected error: %v", err)
	}
	src2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "orders migration failed in prod",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("source 2 Remember returned unexpected error: %v", err)
	}

	derived, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "all DB migrations require a smoke test before production deploy",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.85,
		SourceIDs:  []string{src1.ID, src2.ID},
		Rationale:  "two independent migration failures indicate a recurring deployment risk",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}

	if len(derived.SourceIDs) != 2 {
		t.Fatalf("expected two distinct source ids, got %#v", derived.SourceIDs)
	}
	if derived.Rationale == "" {
		t.Fatal("expected derived rationale to be persisted")
	}
	if len(derived.Relations.Derives) != 2 {
		t.Fatalf("expected derives relations to mirror source ids, got %#v", derived.Relations.Derives)
	}
}

func TestRememberRejectsDerivedWithoutEnoughDistinctSources(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	src, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "single source event",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("source Remember returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "invalid derived memory",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.8,
		SourceIDs:  []string{src.ID, src.ID},
		Rationale:  "not enough distinct evidence",
	})
	if !errors.Is(err, ErrDerivedSourceCount) {
		t.Fatalf("expected ErrDerivedSourceCount, got %v", err)
	}
}

func TestRememberRejectsDerivedWithoutRationale(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	src1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "source one",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("source 1 Remember returned unexpected error: %v", err)
	}
	src2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "source two",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source 2 Remember returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "invalid derived memory",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.8,
		SourceIDs:  []string{src1.ID, src2.ID},
	})
	if !errors.Is(err, ErrDerivedRationale) {
		t.Fatalf("expected ErrDerivedRationale, got %v", err)
	}
}

func TestRememberRejectsDerivedFromDerivedSource(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	src1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "source one",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("source 1 Remember returned unexpected error: %v", err)
	}
	src2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "source two",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source 2 Remember returned unexpected error: %v", err)
	}
	derived, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "first derived memory",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.85,
		SourceIDs:  []string{src1.ID, src2.ID},
		Rationale:  "valid synthesis",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "invalid higher-order synthesis",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.8,
		SourceIDs:  []string{derived.ID, src1.ID},
		Rationale:  "should be rejected",
	})
	if !errors.Is(err, ErrDerivedSourceKind) {
		t.Fatalf("expected ErrDerivedSourceKind, got %v", err)
	}
}

func TestRememberRejectsDerivedBySystemActor(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	src1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "source one",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("source 1 Remember returned unexpected error: %v", err)
	}
	src2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "source two",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source 2 Remember returned unexpected error: %v", err)
	}

	_, err = store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "system-derived memory",
		Kind:       KindDerived,
		Actor:      Actor{ID: "system", Kind: ActorSystem},
		Confidence: 1.0,
		SourceIDs:  []string{src1.ID, src2.ID},
		Rationale:  "system should not be allowed here",
	})
	if !errors.Is(err, ErrDerivedActorKind) && !errors.Is(err, ErrPolicyViolation) {
		t.Fatalf("expected derived actor rejection, got %v", err)
	}
}

func newRememberTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ensureRememberTestSpaceWithEmbedderAndPolicy(t, store, "repo:company/app", DefaultSpaceWritePolicy(), testEmbedder{
		modelID:    "openai/text-embedding-3-small",
		dimensions: 1536,
	}, 1.0, 30)

	return store
}

func ensureRememberTestSpace(t *testing.T, store *Store, spaceID string) {
	t.Helper()
	ensureRememberTestSpaceWithEmbedderAndPolicy(t, store, spaceID, DefaultSpaceWritePolicy(), testEmbedder{
		modelID:    "openai/text-embedding-3-small",
		dimensions: 1536,
	}, 1.0, 30)
}

func newRememberTestStoreWithPolicy(t *testing.T, spaceID string, policy SpaceWritePolicy) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ensureRememberTestSpaceWithEmbedderAndPolicy(t, store, spaceID, policy, testEmbedder{
		modelID:    "openai/text-embedding-3-small",
		dimensions: 1536,
	}, 1.0, 30)

	return store
}

func ensureRememberTestSpaceWithPolicy(t *testing.T, store *Store, spaceID string, policy SpaceWritePolicy) {
	t.Helper()
	ensureRememberTestSpaceWithEmbedderAndPolicy(t, store, spaceID, policy, testEmbedder{
		modelID:    "openai/text-embedding-3-small",
		dimensions: 1536,
	}, 1.0, 30)
}

func newRememberTestStoreWithEmbedderAndPolicy(t *testing.T, spaceID string, policy SpaceWritePolicy, embedder Embedder) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ensureRememberTestSpaceWithEmbedderAndPolicy(t, store, spaceID, policy, embedder, 1.0, 30)

	return store
}

func ensureRememberTestSpaceWithEmbedderAndPolicy(t *testing.T, store *Store, spaceID string, policy SpaceWritePolicy, embedder Embedder, defaultWeight float32, halfLifeDays float32) {
	t.Helper()

	_, err := store.EnsureSpace(spaceID, testEmbedder{
		modelID:    embedder.ModelID(),
		dimensions: embedder.Dimensions(),
	}, SpaceInit{
		DefaultWeight: defaultWeight,
		HalfLifeDays:  halfLifeDays,
		WritePolicy:   policy,
	})
	if err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}
	store.RegisterEmbedder(embedder)
}

func captureSpaceState(t *testing.T, store *Store, spaceID string) string {
	t.Helper()

	var state string
	err := store.db.View(func(tx *bbolt.Tx) error {
		ids := loadSpaceMemoryIDs(tx, spaceID)
		latestBucket := tx.Bucket(bucketLatest)
		memoriesBucket := tx.Bucket(bucketMemories)
		state = "ids="
		rawIDs, err := json.Marshal(ids)
		if err != nil {
			return err
		}
		state += string(rawIDs)
		state += fmt.Sprintf("|latestCount=%d|memoryCount=%d", latestBucket.Stats().KeyN, memoriesBucket.Stats().KeyN)
		return nil
	})
	if err != nil {
		t.Fatalf("captureSpaceState returned unexpected error: %v", err)
	}
	return state
}
