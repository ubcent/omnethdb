package bolt

import (
	"reflect"
	"testing"
	"time"

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
	if plan.SuggestedForgetBatchCommand == "" {
		t.Fatalf("expected suggested forget-batch command, got %#v", plan)
	}
}

func TestGetSynthesisCandidatesReturnsLiveEpisodicClustersOnly(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/synthesis",
		dimensions: 2,
		vectors: map[string][]float32{
			"job run failed with cache timeout":            {1, 0},
			"another job run failed with cache timeout":    {0.98, 0.02},
			"job run failed with cache timeout yesterday":  {0.97, 0.03},
			"repo policy requires reviewed migrations":     {0, 1},
			"historical cache timeout incident superseded": {1, 0},
			"forgotten cache timeout incident":             {1, 0},
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
		Content:    "job run failed with cache timeout",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "another job run failed with cache timeout",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:two", Kind: memory.ActorAgent},
		Confidence: 0.85,
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "job run failed with cache timeout yesterday",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 0.9,
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "repo policy requires reviewed migrations",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "historical cache timeout incident superseded",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
		Confidence: 0.7,
		Relations:  memory.MemoryRelations{Updates: []string{first.ID}},
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	forgotten, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "forgotten cache timeout incident",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:three", Kind: memory.ActorAgent},
		Confidence: 0.75,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(forgotten.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	result, err := store.GetSynthesisCandidates(memory.SynthesisCandidatesRequest{
		SpaceID: "repo:company/app",
	})
	if err != nil {
		t.Fatalf("GetSynthesisCandidates returned unexpected error: %v", err)
	}
	if result.LiveEpisodicCount != 3 {
		t.Fatalf("expected 3 live episodic memories, got %#v", result)
	}
	if len(result.Candidates) == 0 {
		t.Fatalf("expected synthesis candidates, got %#v", result)
	}
	if result.Candidates[0].ClusterSize != 3 {
		t.Fatalf("expected 3-member synthesis cluster, got %#v", result.Candidates[0])
	}
	if result.Candidates[0].SuggestedAction != "review_for_derived" {
		t.Fatalf("expected review_for_derived action, got %#v", result.Candidates[0])
	}
}

func TestGetPromotionSuggestionsReturnsAdvisoryReviewCandidate(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/promotion",
		dimensions: 2,
		vectors: map[string][]float32{
			"team requires reviewed migrations":          {1, 0},
			"repository requires reviewed migrations":    {0.98, 0.02},
			"migration policy requires review approval":  {0.97, 0.03},
			"transient deploy failure during smoke test": {0, 1},
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

	for _, input := range []memory.MemoryInput{
		{
			SpaceID:    "repo:company/app",
			Content:    "team requires reviewed migrations",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
			Confidence: 0.75,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "repository requires reviewed migrations",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:two", Kind: memory.ActorAgent},
			Confidence: 0.8,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "migration policy requires review approval",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 0.95,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "transient deploy failure during smoke test",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:three", Kind: memory.ActorAgent},
			Confidence: 0.6,
		},
	} {
		if _, err := store.Remember(input); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}

	result, err := store.GetPromotionSuggestions(memory.PromotionSuggestionsRequest{
		SpaceID:             "repo:company/app",
		MinObservationCount: 2,
		MinDistinctActors:   2,
		MinDistinctWindows:  1,
		MinCumulativeScore:  2.5,
	})
	if err != nil {
		t.Fatalf("GetPromotionSuggestions returned unexpected error: %v", err)
	}
	if result.LiveEpisodicCount != 4 {
		t.Fatalf("expected 4 live episodic memories, got %#v", result)
	}
	if len(result.Suggestions) == 0 {
		t.Fatalf("expected promotion suggestions, got %#v", result)
	}
	if result.Suggestions[0].SuggestedAction != "review_for_promotion" {
		t.Fatalf("expected review_for_promotion, got %#v", result.Suggestions[0])
	}
	if result.Suggestions[0].ObservationCount < 3 {
		t.Fatalf("expected clustered observation count, got %#v", result.Suggestions[0])
	}
	if result.Suggestions[0].DistinctActors < 2 {
		t.Fatalf("expected multi-actor support, got %#v", result.Suggestions[0])
	}
}

func TestAdvisoryCurationAPIsDefaultToLiveOnlyButCanIncludeHistoricalAndForgotten(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/curation-scope",
		dimensions: 2,
		vectors: map[string][]float32{
			"cache timeout incident root":         {1, 0},
			"cache timeout incident latest":       {1, 0},
			"cache timeout incident forgotten":    {1, 0},
			"unrelated live episodic observation": {0, 1},
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

	root, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "cache timeout incident root",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "cache timeout incident latest",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
		Confidence: 0.85,
		Relations:  memory.MemoryRelations{Updates: []string{root.ID}},
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	forgotten, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "cache timeout incident forgotten",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:two", Kind: memory.ActorAgent},
		Confidence: 0.75,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(forgotten.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "unrelated live episodic observation",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:three", Kind: memory.ActorAgent},
		Confidence: 0.7,
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	defaultResult, err := store.GetSynthesisCandidates(memory.SynthesisCandidatesRequest{
		SpaceID: "repo:company/app",
	})
	if err != nil {
		t.Fatalf("GetSynthesisCandidates returned unexpected error: %v", err)
	}
	if defaultResult.LiveEpisodicCount != 2 {
		t.Fatalf("expected only live latest episodics by default, got %#v", defaultResult)
	}

	inclusiveResult, err := store.GetSynthesisCandidates(memory.SynthesisCandidatesRequest{
		SpaceID:           "repo:company/app",
		IncludeSuperseded: true,
		IncludeForgotten:  true,
	})
	if err != nil {
		t.Fatalf("GetSynthesisCandidates returned unexpected error: %v", err)
	}
	if inclusiveResult.LiveEpisodicCount != 4 {
		t.Fatalf("expected explicit include flags to widen episodic scope, got %#v", inclusiveResult)
	}
}

func TestAdvisoryCurationAPIsDoNotChangeRecallOrProfileBehavior(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/curation-isolation",
		dimensions: 2,
		vectors: map[string][]float32{
			"repository requires reviewed migrations": {1, 0},
			"reviewed migrations are required":        {0.98, 0.02},
			"migration policy requires review":        {0.97, 0.03},
			"payments use signed cursor pagination":   {0, 1},
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

	for _, input := range []memory.MemoryInput{
		{
			SpaceID:    "repo:company/app",
			Content:    "repository requires reviewed migrations",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
			Confidence: 0.8,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "reviewed migrations are required",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:two", Kind: memory.ActorAgent},
			Confidence: 0.85,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "migration policy requires review",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 0.9,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "payments use signed cursor pagination",
			Kind:       memory.KindStatic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 1.0,
		},
	} {
		if _, err := store.Remember(input); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}

	beforeRecall, err := store.Recall(memory.RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "reviewed migrations",
		TopK:     10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	beforeProfile, err := store.GetProfile(memory.ProfileRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "reviewed migrations",
	})
	if err != nil {
		t.Fatalf("GetProfile returned unexpected error: %v", err)
	}

	if _, err := store.GetSynthesisCandidates(memory.SynthesisCandidatesRequest{
		SpaceID: "repo:company/app",
	}); err != nil {
		t.Fatalf("GetSynthesisCandidates returned unexpected error: %v", err)
	}
	if _, err := store.GetPromotionSuggestions(memory.PromotionSuggestionsRequest{
		SpaceID: "repo:company/app",
	}); err != nil {
		t.Fatalf("GetPromotionSuggestions returned unexpected error: %v", err)
	}

	afterRecall, err := store.Recall(memory.RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "reviewed migrations",
		TopK:     10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	afterProfile, err := store.GetProfile(memory.ProfileRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "reviewed migrations",
	})
	if err != nil {
		t.Fatalf("GetProfile returned unexpected error: %v", err)
	}

	if !reflect.DeepEqual(scoredMemoryIDs(beforeRecall), scoredMemoryIDs(afterRecall)) {
		t.Fatalf("expected Recall results to stay unchanged, before=%v after=%v", scoredMemoryIDs(beforeRecall), scoredMemoryIDs(afterRecall))
	}
	if !reflect.DeepEqual(scoredMemoryIDs(beforeProfile.Static), scoredMemoryIDs(afterProfile.Static)) {
		t.Fatalf("expected GetProfile static layer to stay unchanged, before=%v after=%v", scoredMemoryIDs(beforeProfile.Static), scoredMemoryIDs(afterProfile.Static))
	}
	if !reflect.DeepEqual(scoredMemoryIDs(beforeProfile.Episodic), scoredMemoryIDs(afterProfile.Episodic)) {
		t.Fatalf("expected GetProfile episodic layer to stay unchanged, before=%v after=%v", scoredMemoryIDs(beforeProfile.Episodic), scoredMemoryIDs(afterProfile.Episodic))
	}
}

func TestAdvisoryCurationAPIsDoNotMutateAuditRelationsOrLineageState(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/curation-non-mutation",
		dimensions: 2,
		vectors: map[string][]float32{
			"payments use cursor pagination":          {1, 0},
			"payments use signed cursor pagination":   {0.98, 0.02},
			"repository requires reviewed migrations": {0, 1},
			"team requires reviewed migrations":       {0.02, 0.98},
			"migration policy requires review":        {0.03, 0.97},
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

	root, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use cursor pagination",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	updated, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use signed cursor pagination",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 0.9,
		Relations:  memory.MemoryRelations{Updates: []string{root.ID}},
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	for _, input := range []memory.MemoryInput{
		{
			SpaceID:    "repo:company/app",
			Content:    "repository requires reviewed migrations",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
			Confidence: 0.8,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "team requires reviewed migrations",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:two", Kind: memory.ActorAgent},
			Confidence: 0.85,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "migration policy requires review",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 0.9,
		},
	} {
		if _, err := store.Remember(input); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}

	beforeAudit, err := store.GetAuditLog("repo:company/app", time.Time{})
	if err != nil {
		t.Fatalf("GetAuditLog returned unexpected error: %v", err)
	}
	beforeLineage, err := store.GetLineage(root.ID)
	if err != nil {
		t.Fatalf("GetLineage returned unexpected error: %v", err)
	}
	beforeRelated, err := store.GetRelated(updated.ID, memory.RelationUpdates, 1)
	if err != nil {
		t.Fatalf("GetRelated returned unexpected error: %v", err)
	}

	if _, err := store.GetSynthesisCandidates(memory.SynthesisCandidatesRequest{
		SpaceID: "repo:company/app",
	}); err != nil {
		t.Fatalf("GetSynthesisCandidates returned unexpected error: %v", err)
	}
	if _, err := store.GetPromotionSuggestions(memory.PromotionSuggestionsRequest{
		SpaceID: "repo:company/app",
	}); err != nil {
		t.Fatalf("GetPromotionSuggestions returned unexpected error: %v", err)
	}

	afterAudit, err := store.GetAuditLog("repo:company/app", time.Time{})
	if err != nil {
		t.Fatalf("GetAuditLog returned unexpected error: %v", err)
	}
	afterLineage, err := store.GetLineage(root.ID)
	if err != nil {
		t.Fatalf("GetLineage returned unexpected error: %v", err)
	}
	afterRelated, err := store.GetRelated(updated.ID, memory.RelationUpdates, 1)
	if err != nil {
		t.Fatalf("GetRelated returned unexpected error: %v", err)
	}

	if !reflect.DeepEqual(beforeAudit, afterAudit) {
		t.Fatalf("expected advisory APIs to leave audit log unchanged, before=%#v after=%#v", beforeAudit, afterAudit)
	}
	if !reflect.DeepEqual(lineageSignature(beforeLineage), lineageSignature(afterLineage)) {
		t.Fatalf("expected advisory APIs to leave lineage state unchanged, before=%v after=%v", lineageSignature(beforeLineage), lineageSignature(afterLineage))
	}
	if !reflect.DeepEqual(memoryIDs(beforeRelated), memoryIDs(afterRelated)) {
		t.Fatalf("expected advisory APIs to leave relation traversal unchanged, before=%v after=%v", memoryIDs(beforeRelated), memoryIDs(afterRelated))
	}
}

func TestSynthesisCandidatesDowngradeBlobLikeClustersToReviewOnly(t *testing.T) {
	t.Parallel()

	blobA := "Platform rollout incident mixed cache timeout notes; mitigation draft; follow-up owners; retry strategy; migration cleanup and feature flag status; staging fallback; owner checklist; temporary rollout notes; exception handling for cache shard drift."
	blobB := "Platform rollout incident mixed cache timeout notes; mitigation alternatives; on-call summary; retry strategy; migration cleanup and feature flag status; staging fallback; owner checklist; temporary rollout notes; exception handling for cache shard drift."

	embedder := mapEmbedder{
		modelID:    "test/synthesis-blob",
		dimensions: 2,
		vectors: map[string][]float32{
			blobA:                      {1, 0},
			blobB:                      {0.99, 0.01},
			"clean unrelated episodic": {0, 1},
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

	for _, input := range []memory.MemoryInput{
		{
			SpaceID:    "repo:company/app",
			Content:    blobA,
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
			Confidence: 0.8,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    blobB,
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:two", Kind: memory.ActorAgent},
			Confidence: 0.85,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "clean unrelated episodic",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 0.9,
		},
	} {
		if _, err := store.Remember(input); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}

	result, err := store.GetSynthesisCandidates(memory.SynthesisCandidatesRequest{
		SpaceID: "repo:company/app",
	})
	if err != nil {
		t.Fatalf("GetSynthesisCandidates returned unexpected error: %v", err)
	}
	if len(result.Candidates) == 0 {
		t.Fatalf("expected synthesis candidate, got %#v", result)
	}
	if result.Candidates[0].SuggestedAction != "review_only" {
		t.Fatalf("expected blob-like synthesis candidate to downgrade to review_only, got %#v", result.Candidates[0])
	}
	if !hasReasonCode(result.Candidates[0].ReasonCodes, "mixed_fact_blob") {
		t.Fatalf("expected mixed_fact_blob reason code, got %#v", result.Candidates[0])
	}
}

func TestPromotionSuggestionsDowngradeChurnySupportClustersToReviewOnly(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/promotion-churn",
		dimensions: 2,
		vectors: map[string][]float32{
			"repository requires reviewed migrations":         {1, 0},
			"team requires reviewed migrations":               {0.98, 0.02},
			"migration approvals are optional in emergencies": {0.97, 0.03},
			"unrelated deploy note":                           {0, 1},
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

	for _, input := range []memory.MemoryInput{
		{
			SpaceID:    "repo:company/app",
			Content:    "repository requires reviewed migrations",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:one", Kind: memory.ActorAgent},
			Confidence: 0.8,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "team requires reviewed migrations",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:two", Kind: memory.ActorAgent},
			Confidence: 0.85,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "migration approvals are optional in emergencies",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
			Confidence: 0.9,
		},
		{
			SpaceID:    "repo:company/app",
			Content:    "unrelated deploy note",
			Kind:       memory.KindEpisodic,
			Actor:      memory.Actor{ID: "agent:three", Kind: memory.ActorAgent},
			Confidence: 0.7,
		},
	} {
		if _, err := store.Remember(input); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}

	result, err := store.GetPromotionSuggestions(memory.PromotionSuggestionsRequest{
		SpaceID:             "repo:company/app",
		MinObservationCount: 2,
		MinDistinctActors:   2,
		MinCumulativeScore:  2.5,
	})
	if err != nil {
		t.Fatalf("GetPromotionSuggestions returned unexpected error: %v", err)
	}
	if len(result.Suggestions) == 0 {
		t.Fatalf("expected promotion suggestion, got %#v", result)
	}
	if result.Suggestions[0].SuggestedAction != "review_only" {
		t.Fatalf("expected churny promotion suggestion to downgrade to review_only, got %#v", result.Suggestions[0])
	}
	if !hasReasonCode(result.Suggestions[0].ReasonCodes, "high_churn") {
		t.Fatalf("expected high_churn reason code, got %#v", result.Suggestions[0])
	}
}

func scoredMemoryIDs(items []memory.ScoredMemory) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func hasReasonCode(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func memoryIDs(items []memory.Memory) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func lineageSignature(items []memory.Memory) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		rootID := item.ID
		if item.RootID != nil {
			rootID = *item.RootID
		}
		out = append(out, item.ID+":"+rootID+":"+boolString(item.IsLatest)+":"+boolString(item.IsForgotten))
	}
	return out
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
