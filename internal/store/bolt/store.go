package bolt

import (
	"encoding/json"
	"errors"
	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	"os"
	"path/filepath"
	"sync"

	bbolt "go.etcd.io/bbolt"
)

var (
	bucketSpaces       = []byte("spaces")
	bucketSpaceStats   = []byte("space_stats")
	bucketLatest       = []byte("latest")
	bucketEmbeddings   = []byte("embeddings")
	bucketMemories     = []byte("memories")
	bucketRelations    = []byte("relations")
	bucketRelationRefs = []byte("relation_refs")
	bucketSpacesConfig = []byte("spaces_config")
	bucketAuditLog     = []byte("audit_log")
	bucketForgetLog    = []byte("forget_log")
)

type Store struct {
	db        *bbolt.DB
	embedders map[string]memory.Embedder
	mu        sync.RWMutex
}

type SpaceInit struct {
	DefaultWeight float32
	HalfLifeDays  float32
	WritePolicy   memory.SpaceWritePolicy
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := bbolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, err
	}

	store := &Store{
		db:        db,
		embedders: map[string]memory.Embedder{},
	}
	if err := store.initBuckets(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	err := s.db.Close()
	s.db = nil
	return err
}

func (s *Store) EnsureSpace(spaceID string, embedder memory.Embedder, init SpaceInit) (*memory.SpaceConfig, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(spaceID); err != nil {
		return nil, err
	}
	if embedder == nil {
		return nil, memory.ErrNilEmbedder
	}
	s.RegisterEmbedder(embedder)
	if err := ValidateSpaceInit(init); err != nil {
		return nil, err
	}

	persisted := memory.SpaceConfig{
		EmbeddingModelID: embedder.ModelID(),
		Dimension:        embedder.Dimensions(),
		DefaultWeight:    init.DefaultWeight,
		HalfLifeDays:     init.HalfLifeDays,
		WritePolicy:      policy.NormalizeSpaceWritePolicy(init.WritePolicy),
	}
	if err := memory.ValidateSpaceConfig(persisted); err != nil {
		return nil, err
	}

	var cfg memory.SpaceConfig
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSpacesConfig)
		raw := b.Get([]byte(spaceID))
		if raw == nil {
			encoded, err := json.Marshal(persisted)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(spaceID), encoded); err != nil {
				return err
			}
			if err := saveLiveKindCounts(tx, spaceID, map[memory.MemoryKind]int{
				memory.KindEpisodic: 0,
				memory.KindStatic:   0,
				memory.KindDerived:  0,
			}); err != nil {
				return err
			}
			cfg = persisted
			return nil
		}

		if err := json.Unmarshal(raw, &cfg); err != nil {
			return err
		}
		if cfg.EmbeddingModelID != embedder.ModelID() || cfg.Dimension != embedder.Dimensions() {
			return memory.ErrEmbeddingModelMismatch
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (s *Store) RegisterEmbedder(embedder memory.Embedder) {
	if s == nil || embedder == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.embedders == nil {
		s.embedders = map[string]memory.Embedder{}
	}
	s.embedders[embedder.ModelID()] = embedder
}

func (s *Store) GetSpaceConfig(spaceID string) (*memory.SpaceConfig, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(spaceID); err != nil {
		return nil, err
	}

	var cfg memory.SpaceConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		raw := tx.Bucket(bucketSpacesConfig).Get([]byte(spaceID))
		if raw == nil {
			return memory.ErrSpaceNotFound
		}
		return json.Unmarshal(raw, &cfg)
	})
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (s *Store) initBuckets() error {
	if s == nil || s.db == nil {
		return memory.ErrStoreClosed
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		for _, name := range [][]byte{
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
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}
		return nil
	})
}

func ValidateSpaceInit(init SpaceInit) error {
	cfg := memory.SpaceConfig{
		EmbeddingModelID: "bootstrap-placeholder",
		Dimension:        1,
		DefaultWeight:    init.DefaultWeight,
		HalfLifeDays:     init.HalfLifeDays,
		WritePolicy:      init.WritePolicy,
	}
	if err := memory.ValidateSpaceConfig(cfg); err != nil {
		return errors.Join(memory.ErrInvalidSpaceInit, err)
	}
	return nil
}
