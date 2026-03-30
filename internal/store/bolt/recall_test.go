package bolt

import (
	"testing"
	"time"

	. "omnethdb/internal/memory"
)

func TestRecallReturnsOnlyLiveCurrentKnowledge(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	staticLive, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "current static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("static Remember returned unexpected error: %v", err)
	}

	expiredAt := time.Now().UTC().Add(-time.Hour)
	if _, err := store.Remember(MemoryInput{
		SpaceID:     "repo:company/app",
		Content:     "expired episodic fact",
		Kind:        KindEpisodic,
		Actor:       Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence:  0.7,
		ForgetAfter: &expiredAt,
	}); err != nil {
		t.Fatalf("expired Remember returned unexpected error: %v", err)
	}

	root, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "superseded fact v1",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("root Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "superseded fact v2",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{root.ID}},
	}); err != nil {
		t.Fatalf("update Remember returned unexpected error: %v", err)
	}

	forgotten, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "forgotten fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("forgotten Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(forgotten.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	got, err := store.Recall(RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "current",
		TopK:     10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected exactly two live memories, got %d", len(got))
	}
	if got[0].ID != staticLive.ID && got[1].ID != staticLive.ID {
		t.Fatalf("expected current live fact to be present, got %#v", got)
	}
	for _, item := range got {
		if item.ID == root.ID {
			t.Fatal("expected superseded memory to be excluded from recall")
		}
		if item.ID == forgotten.ID {
			t.Fatal("expected forgotten memory to be excluded from recall")
		}
	}
}

func TestRecallFiltersByKindAndTopK(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	if _, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "static fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("static Remember returned unexpected error: %v", err)
	}
	episodic, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "episodic note",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("episodic Remember returned unexpected error: %v", err)
	}

	got, err := store.Recall(RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		TopK:     1,
		Kinds:    []MemoryKind{KindEpisodic},
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}

	if len(got) != 1 || got[0].ID != episodic.ID {
		t.Fatalf("expected only episodic result, got %#v", got)
	}
}

func TestRecallDoesNotTraverseRelations(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	base, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments schema",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("base Remember returned unexpected error: %v", err)
	}
	related, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "ledger entries are append-only",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Extends: []string{base.ID}},
	})
	if err != nil {
		t.Fatalf("related Remember returned unexpected error: %v", err)
	}

	got, err := store.Recall(RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "payments schema",
		TopK:     10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected both live memories in unscored recall candidate set, got %#v", got)
	}
	seenBase := false
	seenRelated := false
	for _, item := range got {
		if item.ID == base.ID {
			seenBase = true
		}
		if item.ID == related.ID {
			seenRelated = true
		}
	}
	if !seenBase || !seenRelated {
		t.Fatalf("expected recall filter stage to preserve both live memories regardless of relation links, got %#v", got)
	}
}

func TestRecallCanExcludeOrphanedDeriveds(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	source1, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "source one",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source1 Remember returned unexpected error: %v", err)
	}
	source2, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "source two",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("source2 Remember returned unexpected error: %v", err)
	}
	derived, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "derived synthesis",
		Kind:       KindDerived,
		Actor:      Actor{ID: "agent:analyst-1", Kind: ActorAgent},
		Confidence: 0.9,
		SourceIDs:  []string{source1.ID, source2.ID},
		Rationale:  "joint inference",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(source1.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	included, err := store.Recall(RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Kinds:    []MemoryKind{KindDerived},
		TopK:     10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	if len(included) != 1 || included[0].ID != derived.ID {
		t.Fatalf("expected orphaned derived to be included by default, got %#v", included)
	}

	excluded, err := store.Recall(RecallRequest{
		SpaceIDs:               []string{"repo:company/app"},
		Kinds:                  []MemoryKind{KindDerived},
		ExcludeOrphanedDerives: true,
		TopK:                   10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	if len(excluded) != 0 {
		t.Fatalf("expected orphaned derived to be excluded on request, got %#v", excluded)
	}
}
