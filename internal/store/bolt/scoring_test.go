package bolt

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"omnethdb/internal/memory"
	. "omnethdb/internal/policy"

	bbolt "go.etcd.io/bbolt"
)

type mapEmbedder struct {
	modelID    string
	dimensions int
	vectors    map[string][]float32
}

func (e mapEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := e.vectors[text]
	out := make([]float32, len(vec))
	copy(out, vec)
	return out, nil
}

func (e mapEmbedder) Dimensions() int { return e.dimensions }
func (e mapEmbedder) ModelID() string { return e.modelID }

func TestRecallScoresByConfidenceAndTrust(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/scoring",
		dimensions: 2,
		vectors: map[string][]float32{
			"query":               {1, 0},
			"same-signal-human":   {1, 0},
			"same-signal-agent":   {1, 0},
			"same-signal-lowconf": {1, 0},
		},
	}
	store := newRememberTestStoreWithEmbedderAndPolicy(t, "repo:company/app", memory.SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.4,
		EpisodicWriters: memory.WritersPolicy{
			AllowHuman:     true,
			AllowSystem:    true,
			AllowAllAgents: true,
		},
		StaticWriters: memory.WritersPolicy{
			AllowHuman:     true,
			AllowAllAgents: true,
		},
		DerivedWriters: memory.WritersPolicy{
			AllowHuman:     true,
			AllowAllAgents: true,
		},
		PromotePolicy: memory.WritersPolicy{AllowHuman: true},
	}, embedder)

	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "same-signal-human",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("human Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "same-signal-agent",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("agent Remember returned unexpected error: %v", err)
	}
	lowConf, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "same-signal-lowconf",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 0.5,
	})
	if err != nil {
		t.Fatalf("lowConf Remember returned unexpected error: %v", err)
	}

	got, err := store.Recall(memory.RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "query",
		TopK:     10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 scored memories, got %#v", got)
	}
	if got[0].Actor.Kind != memory.ActorHuman {
		t.Fatalf("expected human-authored memory to win on trust, got %#v", got[0])
	}
	if got[1].ID != lowConf.ID {
		t.Fatalf("expected low-confidence human memory to rank between full-trust and low-trust candidates, got %#v", got)
	}
	if got[2].Actor.Kind != memory.ActorAgent {
		t.Fatalf("expected low-trust agent memory to rank lowest, got %#v", got)
	}
}

func TestRecallUsesCurrentPolicyTrustAtQueryTime(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/scoring",
		dimensions: 2,
		vectors: map[string][]float32{
			"query":        {1, 0},
			"human-memory": {1, 0},
			"agent-memory": {1, 0},
		},
	}
	store := newRememberTestStoreWithEmbedderAndPolicy(t, "repo:company/app", memory.SpaceWritePolicy{
		HumanTrust:        0.6,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.4,
		EpisodicWriters:   memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true, AllowSystem: true},
		StaticWriters:     memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		DerivedWriters:    memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		PromotePolicy:     memory.WritersPolicy{AllowHuman: true},
	}, embedder)

	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "human-memory",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("human Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "agent-memory",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("agent Remember returned unexpected error: %v", err)
	}

	if err := store.db.Update(func(tx *bbolt.Tx) error {
		cfg, err := loadSpaceConfig(tx, "repo:company/app")
		if err != nil {
			return err
		}
		cfg.WritePolicy.DefaultAgentTrust = 0.95
		encoded, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketSpacesConfig).Put([]byte("repo:company/app"), encoded)
	}); err != nil {
		t.Fatalf("policy mutation returned unexpected error: %v", err)
	}

	got, err := store.Recall(memory.RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "query",
		TopK:     10,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 scored memories, got %#v", got)
	}
	if got[0].Actor.Kind != memory.ActorAgent {
		t.Fatalf("expected agent memory to win after policy trust change, got %#v", got)
	}
}

func TestRecallAppliesEpisodicRecencyAndSpaceWeights(t *testing.T) {
	t.Parallel()

	embedder := mapEmbedder{
		modelID:    "test/scoring",
		dimensions: 2,
		vectors: map[string][]float32{
			"query":     {1, 0},
			"old-ep":    {1, 0},
			"new-ep":    {1, 0},
			"space-one": {1, 0},
			"space-two": {1, 0},
		},
	}
	store := newRememberTestStoreWithEmbedderAndPolicy(t, "repo:company/app", DefaultSpaceWritePolicy(), embedder)
	ensureRememberTestSpaceWithEmbedderAndPolicy(t, store, "repo:company/other", memory.SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		EpisodicWriters:   memory.WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		StaticWriters:     memory.WritersPolicy{AllowHuman: true},
		DerivedWriters:    memory.WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		PromotePolicy:     memory.WritersPolicy{AllowHuman: true},
		MaxStaticMemories: 500, MaxEpisodicMemories: 10000, ProfileMaxStatic: 50, ProfileMaxEpisodic: 10,
	}, embedder, 0.25, 30)

	oldEp, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "old-ep",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("oldEp Remember returned unexpected error: %v", err)
	}
	newEp, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "new-ep",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:scout-1", Kind: memory.ActorAgent},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("newEp Remember returned unexpected error: %v", err)
	}
	if err := store.db.Update(func(tx *bbolt.Tx) error {
		mem, err := loadMemory(tx, oldEp.ID)
		if err != nil {
			return err
		}
		mem.CreatedAt = time.Now().UTC().Add(-45 * 24 * time.Hour)
		return saveMemory(tx, mem)
	}); err != nil {
		t.Fatalf("age mutation returned unexpected error: %v", err)
	}

	got, err := store.Recall(memory.RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "query",
		TopK:     10,
		Kinds:    []memory.MemoryKind{memory.KindEpisodic},
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	if got[0].ID != newEp.ID {
		t.Fatalf("expected newer episodic memory to outrank older one, got %#v", got)
	}

	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "space-one",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("space one Remember returned unexpected error: %v", err)
	}
	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/other",
		Content:    "space-two",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("space two Remember returned unexpected error: %v", err)
	}

	weighted, err := store.Recall(memory.RecallRequest{
		SpaceIDs: []string{"repo:company/app", "repo:company/other"},
		Query:    "query",
		TopK:     10,
		Kinds:    []memory.MemoryKind{memory.KindStatic},
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	if weighted[0].SpaceID != "repo:company/app" {
		t.Fatalf("expected default space weight to favor app space, got %#v", weighted)
	}

	overridden, err := store.Recall(memory.RecallRequest{
		SpaceIDs:     []string{"repo:company/app", "repo:company/other"},
		SpaceWeights: map[string]float32{"repo:company/other": 2.0, "repo:company/app": 0.2},
		Query:        "query",
		TopK:         10,
		Kinds:        []memory.MemoryKind{memory.KindStatic},
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error: %v", err)
	}
	if overridden[0].SpaceID != "repo:company/other" {
		t.Fatalf("expected request override to favor other space, got %#v", overridden)
	}
}
