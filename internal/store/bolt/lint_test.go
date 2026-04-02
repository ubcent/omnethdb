package bolt

import (
	"testing"

	"omnethdb/internal/memory"
)

func TestLintRememberFlagsPossibleDuplicateForStaticCandidate(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/lint",
		dimensions: 2,
		vectors: map[string][]float32{
			"existing fact": {1, 0},
			"new fact":      {1, 0},
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

	existing, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "existing fact",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	result, err := store.LintRemember(memory.RememberLintRequest{
		Input: memory.MemoryInput{
			SpaceID:    "repo:company/app",
			Content:    "existing fact",
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "agent:claude", Kind: memory.ActorAgent},
			Confidence: 0.9,
		},
	})
	if err != nil {
		t.Fatalf("LintRemember returned unexpected error: %v", err)
	}
	if len(result.Candidates) == 0 || result.Candidates[0].ID != existing.ID {
		t.Fatalf("expected existing candidate to be suggested, got %#v", result.Candidates)
	}
	if !hasLintWarning(result.Warnings, memory.RememberLintPossibleDuplicate) {
		t.Fatalf("expected possible_duplicate warning, got %#v", result.Warnings)
	}
	if !hasLintSuggestion(result.Suggestions, memory.RememberLintSuggestSkipDuplicate) {
		t.Fatalf("expected suggest_skip_duplicate suggestion, got %#v", result.Suggestions)
	}
}

func TestLintRememberFlagsPossibleUpdateTargetForNearStaticCandidate(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/lint-update",
		dimensions: 2,
		vectors: map[string][]float32{
			"existing fact": {1, 0},
			"updated fact":  {0.88, 0.3},
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

	existing, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "existing fact",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	result, err := store.LintRemember(memory.RememberLintRequest{
		Input: memory.MemoryInput{
			SpaceID:    "repo:company/app",
			Content:    "updated fact",
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "agent:claude", Kind: memory.ActorAgent},
			Confidence: 0.9,
		},
	})
	if err != nil {
		t.Fatalf("LintRemember returned unexpected error: %v", err)
	}
	if len(result.Candidates) == 0 || result.Candidates[0].ID != existing.ID {
		t.Fatalf("expected existing candidate to be suggested, got %#v", result.Candidates)
	}
	if !hasLintWarning(result.Warnings, memory.RememberLintPossibleUpdateTarget) {
		t.Fatalf("expected possible_update_target warning, got %#v", result.Warnings)
	}
	if !hasLintSuggestion(result.Suggestions, memory.RememberLintSuggestUpdateExisting) {
		t.Fatalf("expected suggest_update_existing suggestion, got %#v", result.Suggestions)
	}
}

func TestLintRememberFlagsMixedFactBlob(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	result, err := store.LintRemember(memory.RememberLintRequest{
		Input: memory.MemoryInput{
			SpaceID: "repo:company/app",
			Content: "Flow A uses service one to compute candidates. Flow B renders results in a different client surface. Phase 2 introduces denormalized fields for performance. Stories are split across backend, notifications, and UI rollout tasks; this is not one atomic fact.",
			Kind:    memory.KindStatic,
			Actor:   memory.Actor{ID: "agent:claude", Kind: memory.ActorAgent},
		},
	})
	if err != nil {
		t.Fatalf("LintRemember returned unexpected error: %v", err)
	}
	if !hasLintWarning(result.Warnings, memory.RememberLintMixedFactBlob) {
		t.Fatalf("expected mixed_fact_blob warning, got %#v", result.Warnings)
	}
	if !hasLintSuggestion(result.Suggestions, memory.RememberLintSuggestSplitCandidate) {
		t.Fatalf("expected suggest_split_candidate suggestion, got %#v", result.Suggestions)
	}
	if len(result.Suggestions) == 0 || len(result.Suggestions[0].ProposedContents) < 2 {
		t.Fatalf("expected proposed split contents, got %#v", result.Suggestions)
	}
}

func hasLintWarning(warnings []memory.RememberLintWarning, code memory.RememberLintWarningCode) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

func hasLintSuggestion(suggestions []memory.RememberLintSuggestion, code memory.RememberLintSuggestionCode) bool {
	for _, suggestion := range suggestions {
		if suggestion.Code == code {
			return true
		}
	}
	return false
}
