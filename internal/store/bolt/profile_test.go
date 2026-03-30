package bolt

import (
	"fmt"
	"testing"

	"omnethdb/internal/memory"
)

func TestGetProfileReturnsSeparateStaticAndEpisodicLayers(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	staticMem, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "static fact",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("static Remember returned unexpected error: %v", err)
	}
	episodicMem, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "episodic note",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("episodic Remember returned unexpected error: %v", err)
	}

	profile, err := store.GetProfile(memory.ProfileRequest{
		SpaceIDs:     []string{"repo:company/app"},
		Query:        "query",
		StaticTopK:   10,
		EpisodicTopK: 10,
	})
	if err != nil {
		t.Fatalf("GetProfile returned unexpected error: %v", err)
	}
	if len(profile.Static) != 1 || profile.Static[0].ID != staticMem.ID {
		t.Fatalf("expected static layer to contain static memory, got %#v", profile.Static)
	}
	if len(profile.Episodic) != 1 || profile.Episodic[0].ID != episodicMem.ID {
		t.Fatalf("expected episodic layer to contain episodic memory, got %#v", profile.Episodic)
	}
}

func TestGetProfileIncludesDerivedInStaticLayer(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	source1, _ := store.Remember(memory.MemoryInput{
		SpaceID: "repo:company/app", Content: "source one", Kind: memory.KindStatic,
		Actor: memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, Confidence: 1.0,
	})
	source2, _ := store.Remember(memory.MemoryInput{
		SpaceID: "repo:company/app", Content: "source two", Kind: memory.KindStatic,
		Actor: memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, Confidence: 1.0,
	})
	derived, err := store.Remember(memory.MemoryInput{
		SpaceID: "repo:company/app", Content: "derived synthesis", Kind: memory.KindDerived,
		Actor: memory.Actor{ID: "agent:analyst-1", Kind: memory.ActorAgent}, Confidence: 0.9,
		SourceIDs: []string{source1.ID, source2.ID}, Rationale: "joint inference",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}

	profile, err := store.GetProfile(memory.ProfileRequest{
		SpaceIDs:     []string{"repo:company/app"},
		Query:        "query",
		StaticTopK:   10,
		EpisodicTopK: 10,
	})
	if err != nil {
		t.Fatalf("GetProfile returned unexpected error: %v", err)
	}

	found := false
	for _, item := range profile.Static {
		if item.ID == derived.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected derived memory in static layer, got %#v", profile.Static)
	}
}

func TestGetProfileCanExcludeOrphanedDeriveds(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	source1, _ := store.Remember(memory.MemoryInput{
		SpaceID: "repo:company/app", Content: "source one", Kind: memory.KindStatic,
		Actor: memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, Confidence: 1.0,
	})
	source2, _ := store.Remember(memory.MemoryInput{
		SpaceID: "repo:company/app", Content: "source two", Kind: memory.KindStatic,
		Actor: memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, Confidence: 1.0,
	})
	derived, err := store.Remember(memory.MemoryInput{
		SpaceID: "repo:company/app", Content: "derived synthesis", Kind: memory.KindDerived,
		Actor: memory.Actor{ID: "agent:analyst-1", Kind: memory.ActorAgent}, Confidence: 0.9,
		SourceIDs: []string{source1.ID, source2.ID}, Rationale: "joint inference",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(source1.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	included, err := store.GetProfile(memory.ProfileRequest{
		SpaceIDs:     []string{"repo:company/app"},
		Query:        "query",
		StaticTopK:   10,
		EpisodicTopK: 10,
	})
	if err != nil {
		t.Fatalf("GetProfile returned unexpected error: %v", err)
	}
	foundIncluded := false
	for _, item := range included.Static {
		if item.ID == derived.ID {
			foundIncluded = true
		}
	}
	if !foundIncluded {
		t.Fatalf("expected orphaned derived in default profile, got %#v", included.Static)
	}

	excluded, err := store.GetProfile(memory.ProfileRequest{
		SpaceIDs:               []string{"repo:company/app"},
		Query:                  "query",
		StaticTopK:             10,
		EpisodicTopK:           10,
		ExcludeOrphanedDerives: true,
	})
	if err != nil {
		t.Fatalf("GetProfile returned unexpected error: %v", err)
	}
	for _, item := range excluded.Static {
		if item.ID == derived.ID {
			t.Fatalf("expected orphaned derived to be excluded, got %#v", excluded.Static)
		}
	}
}

func TestGetProfileUsesIndependentLayerLimits(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	for i := 0; i < 3; i++ {
		if _, err := store.Remember(memory.MemoryInput{
			SpaceID:    "repo:company/app",
			Content:    fmt.Sprintf("static fact %d", i),
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
		}); err != nil {
			t.Fatalf("static Remember returned unexpected error: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := store.Remember(memory.MemoryInput{
			SpaceID:    "repo:company/app",
			Content:    fmt.Sprintf("episodic note %d", i),
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
			Confidence: 0.8,
		}); err != nil {
			t.Fatalf("episodic Remember returned unexpected error: %v", err)
		}
	}

	profile, err := store.GetProfile(memory.ProfileRequest{
		SpaceIDs:     []string{"repo:company/app"},
		Query:        "query",
		StaticTopK:   2,
		EpisodicTopK: 1,
	})
	if err != nil {
		t.Fatalf("GetProfile returned unexpected error: %v", err)
	}
	if len(profile.Static) != 2 {
		t.Fatalf("expected static layer limit 2, got %#v", profile.Static)
	}
	if len(profile.Episodic) != 1 {
		t.Fatalf("expected episodic layer limit 1, got %#v", profile.Episodic)
	}
}

func TestGetProfileCapsExplicitLayerLimitsByPolicy(t *testing.T) {
	t.Parallel()

	store := newRememberTestStoreWithEmbedderAndPolicy(t, "repo:company/app", memory.SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters:   memory.WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		StaticWriters:     memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		DerivedWriters:    memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		PromotePolicy:     memory.WritersPolicy{AllowHuman: true},
		MaxStaticMemories: 500, MaxEpisodicMemories: 10000, ProfileMaxStatic: 2, ProfileMaxEpisodic: 1,
	}, testEmbedder{modelID: "test/profile-caps", dimensions: 8})

	for i := 0; i < 3; i++ {
		if _, err := store.Remember(memory.MemoryInput{
			SpaceID:    "repo:company/app",
			Content:    fmt.Sprintf("policy static %d", i),
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
		}); err != nil {
			t.Fatalf("static Remember returned unexpected error: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := store.Remember(memory.MemoryInput{
			SpaceID:    "repo:company/app",
			Content:    fmt.Sprintf("policy episodic %d", i),
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
			Confidence: 1.0,
		}); err != nil {
			t.Fatalf("episodic Remember returned unexpected error: %v", err)
		}
	}

	profile, err := store.GetProfile(memory.ProfileRequest{
		SpaceIDs:     []string{"repo:company/app"},
		Query:        "query",
		StaticTopK:   10,
		EpisodicTopK: 10,
	})
	if err != nil {
		t.Fatalf("GetProfile returned unexpected error: %v", err)
	}
	if len(profile.Static) != 2 {
		t.Fatalf("expected static layer capped by policy at 2, got %#v", profile.Static)
	}
	if len(profile.Episodic) != 1 {
		t.Fatalf("expected episodic layer capped by policy at 1, got %#v", profile.Episodic)
	}
}
