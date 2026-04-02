package bolt

import (
	"testing"

	"omnethdb/internal/memory"
)

func TestGetQualityDiagnosticsReturnsDuplicateGroupsAndUpdatePairs(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/quality",
		dimensions: 2,
		vectors: map[string][]float32{
			"payments use signed cursor pagination":       {1, 0},
			"payments use signed cursor pagination again": {1, 0},
			"payments use HMAC signed cursor pagination":  {0.88, 0.3},
			"ledger entries are append-only":              {0, 1},
		},
	}
	store := newRememberTestStoreWithEmbedderAndPolicy(t, "repo:company/app", memory.SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters:   memory.WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		StaticWriters:     memory.WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		DerivedWriters:    memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		PromotePolicy:     memory.WritersPolicy{AllowHuman: true},
	}, embedder)

	for _, content := range []string{
		"payments use signed cursor pagination",
		"payments use signed cursor pagination again",
		"payments use HMAC signed cursor pagination",
		"ledger entries are append-only",
	} {
		if _, err := store.Remember(memory.MemoryInput{
			SpaceID:    "repo:company/app",
			Content:    content,
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
		}); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}

	result, err := store.GetQualityDiagnostics(memory.QualityDiagnosticsRequest{
		SpaceID: "repo:company/app",
	})
	if err != nil {
		t.Fatalf("GetQualityDiagnostics returned unexpected error: %v", err)
	}
	if result.LiveStaticCount != 4 {
		t.Fatalf("expected 4 live static memories, got %d", result.LiveStaticCount)
	}
	if len(result.DuplicateGroups) == 0 {
		t.Fatalf("expected duplicate groups, got %#v", result)
	}
	if len(result.DuplicateGroups[0].MemoryIDs) != 2 {
		t.Fatalf("expected 2-memory duplicate group, got %#v", result.DuplicateGroups[0])
	}
	if len(result.PossibleUpdates) == 0 {
		t.Fatalf("expected possible update pairs, got %#v", result)
	}
}

func TestBuildQualityCleanupPlanSuggestsKeepAndForgetIDs(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/quality-plan",
		dimensions: 2,
		vectors: map[string][]float32{
			"payments use signed cursor pagination":       {1, 0},
			"payments use signed cursor pagination again": {1, 0},
		},
	}
	store := newRememberTestStoreWithEmbedderAndPolicy(t, "repo:company/app", memory.SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters:   memory.WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		StaticWriters:     memory.WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		DerivedWriters:    memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		PromotePolicy:     memory.WritersPolicy{AllowHuman: true},
	}, embedder)

	first, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use signed cursor pagination",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	second, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use signed cursor pagination again",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	plan, err := store.BuildQualityCleanupPlan(memory.QualityCleanupPlanRequest{
		SpaceID: "repo:company/app",
	})
	if err != nil {
		t.Fatalf("BuildQualityCleanupPlan returned unexpected error: %v", err)
	}
	if len(plan.DuplicateSuggestions) == 0 {
		t.Fatalf("expected duplicate suggestions, got %#v", plan)
	}
	if plan.DuplicateSuggestions[0].KeepID != second.ID {
		t.Fatalf("expected higher-confidence memory to be kept, got %#v", plan.DuplicateSuggestions[0])
	}
	if len(plan.DuplicateSuggestions[0].ForgetIDs) != 1 || plan.DuplicateSuggestions[0].ForgetIDs[0] != first.ID {
		t.Fatalf("expected lower-confidence memory to be forgotten, got %#v", plan.DuplicateSuggestions[0])
	}
}
