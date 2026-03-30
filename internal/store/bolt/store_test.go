package bolt

import (
	"context"
	"errors"
	"hash/fnv"
	. "omnethdb/internal/memory"
	. "omnethdb/internal/policy"
	"path/filepath"
	"testing"

	bbolt "go.etcd.io/bbolt"
)

type testEmbedder struct {
	modelID    string
	dimensions int
}

func (e testEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, e.dimensions)
	if e.dimensions == 0 {
		return vec, nil
	}

	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(text))
	hash := hasher.Sum64()
	for i := 0; i < e.dimensions; i++ {
		shift := uint((i % 8) * 8)
		value := float32((hash>>shift)&0xff) + 1
		vec[i] = value / 255
	}
	return vec, nil
}
func (e testEmbedder) Dimensions() int { return e.dimensions }
func (e testEmbedder) ModelID() string { return e.modelID }

func TestOpenInitializesBuckets(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	err = store.db.View(func(tx *bbolt.Tx) error {
		for _, bucket := range [][]byte{
			bucketSpaces,
			bucketSpaceStats,
			bucketLatest,
			bucketEmbeddings,
			bucketMemories,
			bucketRelations,
			bucketRelationRefs,
			bucketSpacesConfig,
			bucketAuditLog,
			bucketForgetLog,
		} {
			if tx.Bucket(bucket) == nil {
				t.Fatalf("expected bucket %q to exist", bucket)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("bucket verification returned unexpected error: %v", err)
	}
}

func TestEnsureSpaceBootstrapsAndPersistsConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}

	init := SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   DefaultSpaceWritePolicy(),
	}

	cfg, err := store.EnsureSpace("repo:company/app", testEmbedder{
		modelID:    "openai/text-embedding-3-small",
		dimensions: 1536,
	}, init)
	if err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}

	if cfg.EmbeddingModelID != "openai/text-embedding-3-small" {
		t.Fatalf("expected persisted model id, got %q", cfg.EmbeddingModelID)
	}
	if cfg.Dimension != 1536 {
		t.Fatalf("expected persisted dimension 1536, got %d", cfg.Dimension)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	persisted, err := reopened.GetSpaceConfig("repo:company/app")
	if err != nil {
		t.Fatalf("GetSpaceConfig returned unexpected error: %v", err)
	}
	if persisted.EmbeddingModelID != "openai/text-embedding-3-small" || persisted.Dimension != 1536 {
		t.Fatalf("unexpected persisted config: %+v", *persisted)
	}
}

func TestEnsureSpaceRejectsEmbedderMismatch(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	init := SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   DefaultSpaceWritePolicy(),
	}

	if _, err := store.EnsureSpace("repo:company/app", testEmbedder{
		modelID:    "openai/text-embedding-3-small",
		dimensions: 1536,
	}, init); err != nil {
		t.Fatalf("initial EnsureSpace returned unexpected error: %v", err)
	}

	_, err = store.EnsureSpace("repo:company/app", testEmbedder{
		modelID:    "local/bge-m3-v1.2",
		dimensions: 1024,
	}, init)
	if !errors.Is(err, ErrEmbeddingModelMismatch) {
		t.Fatalf("expected ErrEmbeddingModelMismatch, got %v", err)
	}
}

func TestEnsureSpaceOnlyPersistsSpaceConfigDuringBootstrap(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	init := SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   DefaultSpaceWritePolicy(),
	}

	if _, err := store.EnsureSpace("repo:company/app", testEmbedder{
		modelID:    "openai/text-embedding-3-small",
		dimensions: 1536,
	}, init); err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}

	err = store.db.View(func(tx *bbolt.Tx) error {
		if stats := tx.Bucket(bucketSpacesConfig).Stats(); stats.KeyN != 1 {
			t.Fatalf("expected spaces_config to contain one key, got %d", stats.KeyN)
		}

		for _, bucket := range [][]byte{
			bucketSpaces,
			bucketSpaceStats,
			bucketLatest,
			bucketEmbeddings,
			bucketMemories,
			bucketRelations,
			bucketRelationRefs,
			bucketAuditLog,
			bucketForgetLog,
		} {
			if stats := tx.Bucket(bucket).Stats(); stats.KeyN != 0 {
				t.Fatalf("expected bucket %q to remain empty after bootstrap, got %d keys", bucket, stats.KeyN)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("storage invariant verification returned unexpected error: %v", err)
	}
}

func TestGetSpaceConfigReturnsErrSpaceNotFound(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	_, err = store.GetSpaceConfig("repo:company/missing")
	if !errors.Is(err, ErrSpaceNotFound) {
		t.Fatalf("expected ErrSpaceNotFound, got %v", err)
	}
}

func TestLiveKindCountsTrackRememberForgetAndRevive(t *testing.T) {
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
	episodicMem, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "episodic fact",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("episodic Remember returned unexpected error: %v", err)
	}

	assertCounts := func(wantStatic int, wantEpisodic int) {
		t.Helper()
		err := store.db.View(func(tx *bbolt.Tx) error {
			counts, err := loadLiveKindCounts(tx, "repo:company/app")
			if err != nil {
				return err
			}
			if counts[KindStatic] != wantStatic || counts[KindDerived] != 0 || counts[KindEpisodic] != wantEpisodic {
				t.Fatalf("unexpected live counts: %#v", counts)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("count verification returned unexpected error: %v", err)
		}
	}

	assertCounts(1, 1)

	if err := store.Forget(episodicMem.ID, Actor{ID: "user:alice", Kind: ActorHuman}, "cleanup"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}
	assertCounts(1, 0)

	revivedEpisodic, err := store.Revive(episodicMem.ID, ReviveInput{
		Content:    "episodic fact revived",
		Kind:       KindEpisodic,
		Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Revive returned unexpected error: %v", err)
	}
	assertCounts(1, 1)

	if _, err := store.Remember(MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "promoted fact",
		Kind:       KindStatic,
		Actor:      Actor{ID: "user:alice", Kind: ActorHuman},
		Confidence: 1.0,
		Relations:  MemoryRelations{Updates: []string{revivedEpisodic.ID}},
	}); err != nil {
		t.Fatalf("promotion update returned unexpected error: %v", err)
	}
	assertCounts(2, 0)
}
