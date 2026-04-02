package bolt

import (
	"encoding/json"

	"omnethdb/internal/memory"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) UpdateSpaceConfig(spaceID string, next memory.SpaceConfig) (*memory.SpaceConfig, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(spaceID); err != nil {
		return nil, err
	}
	if err := memory.ValidateSpaceConfig(next); err != nil {
		return nil, err
	}

	var persisted memory.SpaceConfig
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSpacesConfig)
		raw := b.Get([]byte(spaceID))
		if raw == nil {
			return memory.ErrSpaceNotFound
		}
		if err := json.Unmarshal(raw, &persisted); err != nil {
			return err
		}
		encoded, err := json.Marshal(next)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(spaceID), encoded); err != nil {
			return err
		}
		persisted = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &persisted, nil
}
