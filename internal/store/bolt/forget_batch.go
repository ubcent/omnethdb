package bolt

import (
	"sort"

	"omnethdb/internal/memory"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) ForgetBatch(ids []string, actor memory.Actor, reason string) error {
	if s == nil || s.db == nil {
		return memory.ErrStoreClosed
	}
	if err := memory.ValidateActor(actor); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	deduped := distinctStrings(ids)
	sort.Strings(deduped)

	return s.db.Update(func(tx *bbolt.Tx) error {
		for _, id := range deduped {
			if err := memory.ValidateMemoryID(id); err != nil {
				return err
			}
			if err := forgetInTx(tx, id, actor, reason); err != nil {
				return err
			}
		}
		return nil
	})
}
