package bolt

import (
	"context"
	"errors"
	"hash/fnv"
	"path/filepath"
	"strings"
	"testing"

	"omnethdb/internal/memory"
	"omnethdb/internal/policy"

	bbolt "go.etcd.io/bbolt"
)

type modelAwareEmbedder struct {
	modelID    string
	dimensions int
}

func (e modelAwareEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, e.dimensions)
	if e.dimensions == 0 {
		return vec, nil
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(e.modelID + "::" + text))
	hash := hasher.Sum64()
	for i := 0; i < e.dimensions; i++ {
		shift := uint((i % 8) * 8)
		value := float32((hash>>shift)&0xff) + 1
		vec[i] = value / 255
	}
	return vec, nil
}

func (e modelAwareEmbedder) Dimensions() int { return e.dimensions }
func (e modelAwareEmbedder) ModelID() string { return e.modelID }

func TestBeginEmbeddingMigrationRejectsWritesButKeepsReadsAvailable(t *testing.T) {
	t.Parallel()

	store := newMigrationTestStore(t)

	mem, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~5 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	target := modelAwareEmbedder{modelID: "local/bge-m3-v1.2", dimensions: 8}
	if err := store.BeginEmbeddingMigration(mem.SpaceID, target); err != nil {
		t.Fatalf("BeginEmbeddingMigration returned unexpected error: %v", err)
	}

	cfg, err := store.GetSpaceConfig(mem.SpaceID)
	if err != nil {
		t.Fatalf("GetSpaceConfig returned unexpected error: %v", err)
	}
	if !cfg.Migrating {
		t.Fatal("expected space to be marked migrating")
	}
	if cfg.MigrationTargetModelID != target.ModelID() || cfg.MigrationTargetDimension != target.Dimensions() {
		t.Fatalf("unexpected migration target in config: %+v", *cfg)
	}

	_, err = store.Remember(memory.MemoryInput{
		SpaceID:    mem.SpaceID,
		Content:    "new write during migration",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if !errors.Is(err, memory.ErrSpaceMigrating) {
		t.Fatalf("expected ErrSpaceMigrating from Remember, got %v", err)
	}

	err = store.Forget(mem.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "cleanup")
	if !errors.Is(err, memory.ErrSpaceMigrating) {
		t.Fatalf("expected ErrSpaceMigrating from Forget, got %v", err)
	}

	results, err := store.Recall(memory.RecallRequest{
		SpaceIDs: []string{mem.SpaceID},
		Query:    "deploy",
		TopK:     5,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error during migration: %v", err)
	}
	if len(results) != 1 || results[0].ID != mem.ID {
		t.Fatalf("expected reads to remain available during migration, got %#v", results)
	}
}

func TestMigrateEmbeddingsReembedsLiveAndHistoricalMemories(t *testing.T) {
	t.Parallel()

	store := newMigrationTestStore(t)

	v1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~5 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("root Remember returned unexpected error: %v", err)
	}
	v2, err := store.Remember(memory.MemoryInput{
		SpaceID:    v1.SpaceID,
		Content:    "deploy takes ~9 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
		Relations:  memory.MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("update Remember returned unexpected error: %v", err)
	}

	beforeV1 := cloneEmbedding(v1.Embedding)
	beforeV2 := cloneEmbedding(v2.Embedding)

	target := modelAwareEmbedder{modelID: "local/bge-m3-v1.2", dimensions: 8}
	if err := store.MigrateEmbeddings(v1.SpaceID, target); err != nil {
		t.Fatalf("MigrateEmbeddings returned unexpected error: %v", err)
	}

	cfg, err := store.GetSpaceConfig(v1.SpaceID)
	if err != nil {
		t.Fatalf("GetSpaceConfig returned unexpected error: %v", err)
	}
	if cfg.Migrating {
		t.Fatal("expected migration to complete and clear migrating flag")
	}
	if cfg.EmbeddingModelID != target.ModelID() || cfg.Dimension != target.Dimensions() {
		t.Fatalf("expected migrated config to point at target embedder, got %+v", *cfg)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		persistedV1, err := loadMemory(tx, v1.ID)
		if err != nil {
			return err
		}
		persistedV2, err := loadMemory(tx, v2.ID)
		if err != nil {
			return err
		}
		if len(persistedV1.Embedding) != target.Dimensions() || len(persistedV2.Embedding) != target.Dimensions() {
			t.Fatalf("expected migrated embeddings to have target dimension %d", target.Dimensions())
		}
		if sameEmbedding(beforeV1, persistedV1.Embedding) {
			t.Fatal("expected historical memory embedding to change during migration")
		}
		if sameEmbedding(beforeV2, persistedV2.Embedding) {
			t.Fatal("expected live memory embedding to change during migration")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view returned unexpected error: %v", err)
	}

	audit, err := store.GetAuditLog(v1.SpaceID, v1.CreatedAt.Add(-1))
	if err != nil {
		t.Fatalf("GetAuditLog returned unexpected error: %v", err)
	}
	var sawStart bool
	var sawComplete bool
	for _, entry := range audit {
		switch entry.Operation {
		case "migration_start":
			sawStart = strings.Contains(entry.Reason, target.ModelID())
		case "migration_complete":
			sawComplete = strings.Contains(entry.Reason, target.ModelID())
		}
	}
	if !sawStart || !sawComplete {
		t.Fatalf("expected migration audit entries, got %#v", audit)
	}
}

func TestMigrateEmbeddingsCanResumeFromPersistedMigratingState(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}

	old := modelAwareEmbedder{modelID: "openai/text-embedding-3-small", dimensions: 6}
	ensureRememberTestSpaceWithEmbedderAndPolicy(t, store, "repo:company/app", policy.DefaultSpaceWritePolicy(), old, 1.0, 30)

	mem, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy takes ~5 min",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}

	target := modelAwareEmbedder{modelID: "local/bge-m3-v1.2", dimensions: 8}
	if err := store.BeginEmbeddingMigration(mem.SpaceID, target); err != nil {
		t.Fatalf("BeginEmbeddingMigration returned unexpected error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	_, err = reopened.Remember(memory.MemoryInput{
		SpaceID:    mem.SpaceID,
		Content:    "write during resumed migration",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if !errors.Is(err, memory.ErrSpaceMigrating) {
		t.Fatalf("expected ErrSpaceMigrating after reopen, got %v", err)
	}

	if err := reopened.MigrateEmbeddings(mem.SpaceID, target); err != nil {
		t.Fatalf("rerun MigrateEmbeddings returned unexpected error: %v", err)
	}

	cfg, err := reopened.GetSpaceConfig(mem.SpaceID)
	if err != nil {
		t.Fatalf("GetSpaceConfig returned unexpected error: %v", err)
	}
	if cfg.Migrating || cfg.EmbeddingModelID != target.ModelID() || cfg.Dimension != target.Dimensions() {
		t.Fatalf("expected resumed migration to complete cleanly, got %+v", *cfg)
	}
}

func newMigrationTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ensureRememberTestSpaceWithEmbedderAndPolicy(t, store, "repo:company/app", policy.DefaultSpaceWritePolicy(), modelAwareEmbedder{
		modelID:    "openai/text-embedding-3-small",
		dimensions: 6,
	}, 1.0, 30)

	return store
}

func sameEmbedding(a []float32, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
