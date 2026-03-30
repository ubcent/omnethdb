package bolt

import (
	"context"
	"fmt"
	"time"

	"omnethdb/internal/memory"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) BeginEmbeddingMigration(spaceID string, embedder memory.Embedder) error {
	if s == nil || s.db == nil {
		return memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(spaceID); err != nil {
		return err
	}
	if embedder == nil {
		return memory.ErrNilEmbedder
	}

	s.RegisterEmbedder(embedder)

	return s.db.Update(func(tx *bbolt.Tx) error {
		cfg, err := loadSpaceConfig(tx, spaceID)
		if err != nil {
			return err
		}

		if cfg.Migrating {
			if cfg.MigrationTargetModelID == embedder.ModelID() && cfg.MigrationTargetDimension == embedder.Dimensions() {
				return nil
			}
			return memory.ErrSpaceMigrating
		}
		if cfg.EmbeddingModelID == embedder.ModelID() && cfg.Dimension == embedder.Dimensions() {
			return nil
		}

		now := time.Now().UTC()
		cfg.Migrating = true
		cfg.MigrationTargetModelID = embedder.ModelID()
		cfg.MigrationTargetDimension = embedder.Dimensions()
		cfg.MigrationStartedAt = &now
		if err := saveSpaceConfig(tx, spaceID, *cfg); err != nil {
			return err
		}

		return writeAuditEntry(tx, memory.AuditEntry{
			Timestamp: now,
			SpaceID:   spaceID,
			Operation: "migration_start",
			Actor:     migrationActor(),
			Reason:    fmt.Sprintf("%s:%d", embedder.ModelID(), embedder.Dimensions()),
		})
	})
}

func (s *Store) MigrateEmbeddings(spaceID string, embedder memory.Embedder) error {
	if s == nil || s.db == nil {
		return memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(spaceID); err != nil {
		return err
	}
	if embedder == nil {
		return memory.ErrNilEmbedder
	}

	if err := s.BeginEmbeddingMigration(spaceID, embedder); err != nil {
		return err
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		cfg, err := loadSpaceConfig(tx, spaceID)
		if err != nil {
			return err
		}
		if !cfg.Migrating {
			return nil
		}
		if cfg.MigrationTargetModelID != embedder.ModelID() || cfg.MigrationTargetDimension != embedder.Dimensions() {
			return memory.ErrSpaceMigrating
		}

		b := tx.Bucket(bucketMemories)
		c := b.Cursor()
		for key, raw := c.First(); key != nil; key, raw = c.Next() {
			var mem memory.Memory
			if err := unmarshalMemory(raw, &mem); err != nil {
				return err
			}
			if mem.SpaceID != spaceID {
				continue
			}

			embedding, err := embedder.Embed(context.Background(), mem.Content)
			if err != nil {
				return err
			}
			mem.Embedding = cloneEmbedding(embedding)
			if err := saveMemory(tx, &mem); err != nil {
				return err
			}
			if err := saveEmbedding(tx, mem.ID, mem.Embedding); err != nil {
				return err
			}
		}

		now := time.Now().UTC()
		cfg.EmbeddingModelID = embedder.ModelID()
		cfg.Dimension = embedder.Dimensions()
		cfg.Migrating = false
		cfg.MigrationTargetModelID = ""
		cfg.MigrationTargetDimension = 0
		cfg.MigrationStartedAt = nil
		if err := saveSpaceConfig(tx, spaceID, *cfg); err != nil {
			return err
		}

		return writeAuditEntry(tx, memory.AuditEntry{
			Timestamp: now,
			SpaceID:   spaceID,
			Operation: "migration_complete",
			Actor:     migrationActor(),
			Reason:    fmt.Sprintf("%s:%d", embedder.ModelID(), embedder.Dimensions()),
		})
	})
}

func ensureSpaceWritable(cfg *memory.SpaceConfig) error {
	if cfg != nil && cfg.Migrating {
		return memory.ErrSpaceMigrating
	}
	return nil
}

func saveSpaceConfig(tx *bbolt.Tx, spaceID string, cfg memory.SpaceConfig) error {
	if err := memory.ValidateSpaceConfig(cfg); err != nil {
		return err
	}
	encoded, err := marshalJSON(cfg)
	if err != nil {
		return err
	}
	return tx.Bucket(bucketSpacesConfig).Put([]byte(spaceID), encoded)
}

func migrationActor() memory.Actor {
	return memory.Actor{ID: "system:migrator", Kind: memory.ActorSystem}
}
