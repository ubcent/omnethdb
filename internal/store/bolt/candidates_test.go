package bolt

import (
	"testing"

	"omnethdb/internal/memory"
)

func TestFindCandidatesUsesRawCosineOnly(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/candidates",
		dimensions: 2,
		vectors: map[string][]float32{
			"query":              {1, 0},
			"high-cos-low-trust": {1, 0},
			"mid-cos-high-trust": {0.8, 0.2},
		},
	}
	store := newRememberTestStoreWithEmbedderAndPolicy(t, "repo:company/app", memory.SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.2,
		EpisodicWriters:   memory.WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		StaticWriters:     memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		DerivedWriters:    memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		PromotePolicy:     memory.WritersPolicy{AllowHuman: true},
	}, embedder)

	lowTrust, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "high-cos-low-trust",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
		Confidence: 0.2,
	})
	if err != nil {
		t.Fatalf("lowTrust Remember returned unexpected error: %v", err)
	}
	highTrust, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "mid-cos-high-trust",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("highTrust Remember returned unexpected error: %v", err)
	}

	got, err := store.FindCandidates(memory.FindCandidatesRequest{
		SpaceID: "repo:company/app",
		Content: "query",
		TopK:    10,
	})
	if err != nil {
		t.Fatalf("FindCandidates returned unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", got)
	}
	if got[0].ID != lowTrust.ID {
		t.Fatalf("expected raw-cosine candidate to win despite low trust/confidence, got %#v", got)
	}
	if got[1].ID != highTrust.ID {
		t.Fatalf("expected lower-cosine candidate second, got %#v", got)
	}
}

func TestFindCandidatesDefaultsToLiveOnly(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	v1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "candidate v1",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("v1 Remember returned unexpected error: %v", err)
	}
	v2, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "candidate v2",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
		Relations:  memory.MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("v2 Remember returned unexpected error: %v", err)
	}
	forgotten, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "forgotten candidate",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("forgotten Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(forgotten.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	got, err := store.FindCandidates(memory.FindCandidatesRequest{
		SpaceID: "repo:company/app",
		Content: "candidate v2",
		TopK:    10,
	})
	if err != nil {
		t.Fatalf("FindCandidates returned unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != v2.ID {
		t.Fatalf("expected only live latest candidate, got %#v", got)
	}
}

func TestFindCandidatesCanIncludeHistoricalAndForgotten(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	v1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "candidate v1",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("v1 Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "candidate v2",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
		Relations:  memory.MemoryRelations{Updates: []string{v1.ID}},
	}); err != nil {
		t.Fatalf("v2 Remember returned unexpected error: %v", err)
	}
	forgotten, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "forgotten candidate",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("forgotten Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(forgotten.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	got, err := store.FindCandidates(memory.FindCandidatesRequest{
		SpaceID:           "repo:company/app",
		Content:           "candidate v1",
		TopK:              10,
		IncludeSuperseded: true,
		IncludeForgotten:  true,
	})
	if err != nil {
		t.Fatalf("FindCandidates returned unexpected error: %v", err)
	}
	if len(got) < 3 {
		t.Fatalf("expected historical and forgotten candidates to be included, got %#v", got)
	}
}

func TestFindCandidatesAndRecallHaveDifferentSemanticsOnSameCorpus(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/divergence",
		dimensions: 2,
		vectors: map[string][]float32{
			"query":                {1, 0},
			"high-cos-low-trust":   {1, 0},
			"lower-cos-high-trust": {0.8, 0.2},
		},
	}
	store := newRememberTestStoreWithEmbedderAndPolicy(t, "repo:company/app", memory.SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.2,
		EpisodicWriters:   memory.WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		StaticWriters:     memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		DerivedWriters:    memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		PromotePolicy:     memory.WritersPolicy{AllowHuman: true},
	}, embedder)

	agentMem, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "high-cos-low-trust",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("agent Remember returned unexpected error: %v", err)
	}
	humanMem, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "lower-cos-high-trust",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("human Remember returned unexpected error: %v", err)
	}

	recall, err := store.Recall(memory.RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "query",
		TopK:     10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	candidates, err := store.FindCandidates(memory.FindCandidatesRequest{
		SpaceID: "repo:company/app",
		Content: "query",
		TopK:    10,
	})
	if err != nil {
		t.Fatalf("FindCandidates returned unexpected error: %v", err)
	}

	if len(recall) != 2 || len(candidates) != 2 {
		t.Fatalf("expected both APIs to return two records, got recall=%#v candidates=%#v", recall, candidates)
	}
	if recall[0].ID != humanMem.ID {
		t.Fatalf("expected Recall to prefer high-trust memory, got %#v", recall)
	}
	if candidates[0].ID != agentMem.ID {
		t.Fatalf("expected FindCandidates to prefer highest cosine memory, got %#v", candidates)
	}
}
