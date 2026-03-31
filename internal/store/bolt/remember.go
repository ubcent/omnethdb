package bolt

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"omnethdb/internal/memory"
	"strings"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) Remember(input memory.MemoryInput) (*memory.Memory, error) {
	prepared, err := s.prepareRememberInputs([]memory.MemoryInput{input})
	if err != nil {
		return nil, err
	}

	var mem *memory.Memory
	err = s.db.Update(func(tx *bbolt.Tx) error {
		written, err := s.rememberPreparedInTx(tx, prepared[0])
		if err != nil {
			return err
		}
		mem = written
		return nil
	})
	if err != nil {
		return nil, err
	}
	return mem, nil
}

func countLiveKinds(tx *bbolt.Tx, spaceID string) (map[memory.MemoryKind]int, error) {
	return loadLiveKindCounts(tx, spaceID)
}

func wouldExceedCorpusLimit(policyConfig memory.SpaceWritePolicy, counts map[memory.MemoryKind]int, newKind memory.MemoryKind, replacedKind memory.MemoryKind) bool {
	if replacedKind == newKind {
		return false
	}

	nextStatic := counts[memory.KindStatic] + counts[memory.KindDerived]
	nextEpisodic := counts[memory.KindEpisodic]

	if replacedKind == memory.KindStatic || replacedKind == memory.KindDerived {
		nextStatic--
	}
	if replacedKind == memory.KindEpisodic {
		nextEpisodic--
	}

	if newKind == memory.KindStatic || newKind == memory.KindDerived {
		nextStatic++
	}
	if newKind == memory.KindEpisodic {
		nextEpisodic++
	}

	return nextStatic > policyConfig.MaxStaticMemories || nextEpisodic > policyConfig.MaxEpisodicMemories
}

func validateRememberRelations(rel memory.MemoryRelations) error {
	if len(rel.Updates) > 1 {
		return memory.ErrInvalidRelations
	}
	return nil
}

func validateDerivedInput(tx *bbolt.Tx, mem *memory.Memory) error {
	if mem.Kind != memory.KindDerived {
		return nil
	}
	if mem.Actor.Kind != memory.ActorHuman && mem.Actor.Kind != memory.ActorAgent {
		return memory.ErrDerivedActorKind
	}
	if strings.TrimSpace(mem.Rationale) == "" {
		return memory.ErrDerivedRationale
	}

	distinct := distinctStrings(mem.SourceIDs)
	if len(distinct) < 2 {
		return memory.ErrDerivedSourceCount
	}
	mem.SourceIDs = distinct
	mem.Relations.Derives = append([]string(nil), distinct...)

	for _, sourceID := range distinct {
		source, err := loadMemoryForDerived(tx, sourceID)
		if err != nil {
			return err
		}
		if source.SpaceID != mem.SpaceID {
			return memory.ErrDerivedAcrossSpaces
		}
		if source.IsForgotten {
			return memory.ErrDerivedSourceForgotten
		}
		if !source.IsLatest {
			return memory.ErrDerivedSourceNotLatest
		}
		if source.Kind != memory.KindEpisodic && source.Kind != memory.KindStatic {
			return memory.ErrDerivedSourceKind
		}
	}

	return nil
}

func saveMemory(tx *bbolt.Tx, mem *memory.Memory) error {
	encoded, err := marshalJSON(mem)
	if err != nil {
		return err
	}
	return tx.Bucket(bucketMemories).Put([]byte(mem.ID), encoded)
}

func saveEmbedding(tx *bbolt.Tx, id string, embedding []float32) error {
	encoded, err := marshalJSON(embedding)
	if err != nil {
		return err
	}
	return tx.Bucket(bucketEmbeddings).Put([]byte(id), encoded)
}

func loadMemory(tx *bbolt.Tx, id string) (*memory.Memory, error) {
	raw := tx.Bucket(bucketMemories).Get([]byte(id))
	if raw == nil {
		return nil, memory.ErrUpdateTargetNotFound
	}

	var mem memory.Memory
	if err := unmarshalMemory(raw, &mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

func loadLatest(tx *bbolt.Tx, rootID string) string {
	return string(tx.Bucket(bucketLatest).Get([]byte(rootID)))
}

func validateExtendsTargets(tx *bbolt.Tx, sourceID string, spaceID string, targetIDs []string) error {
	for _, targetID := range targetIDs {
		if targetID == sourceID {
			return memory.ErrInvalidRelations
		}

		target, err := loadMemoryForRelation(tx, targetID)
		if err != nil {
			return err
		}
		if target.SpaceID != spaceID {
			return memory.ErrExtendsAcrossSpaces
		}
		if target.IsForgotten {
			return memory.ErrExtendsTargetForgotten
		}
		if !target.IsLatest {
			return memory.ErrExtendsTargetNotLatest
		}
	}
	return nil
}

func loadMemoryForRelation(tx *bbolt.Tx, id string) (*memory.Memory, error) {
	raw := tx.Bucket(bucketMemories).Get([]byte(id))
	if raw == nil {
		return nil, memory.ErrExtendsTargetNotFound
	}

	var mem memory.Memory
	if err := json.Unmarshal(raw, &mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

func loadMemoryForDerived(tx *bbolt.Tx, id string) (*memory.Memory, error) {
	raw := tx.Bucket(bucketMemories).Get([]byte(id))
	if raw == nil {
		return nil, memory.ErrDerivedSourceNotFound
	}

	var mem memory.Memory
	if err := json.Unmarshal(raw, &mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

func saveRelations(tx *bbolt.Tx, mem *memory.Memory) error {
	for _, toID := range mem.Relations.Updates {
		if err := putRelation(tx, mem.ID, memory.RelationUpdates, toID); err != nil {
			return err
		}
	}
	for _, toID := range mem.Relations.Extends {
		if err := putRelation(tx, mem.ID, memory.RelationExtends, toID); err != nil {
			return err
		}
	}
	for _, toID := range mem.Relations.Derives {
		if err := putRelation(tx, mem.ID, memory.RelationDerives, toID); err != nil {
			return err
		}
	}
	return nil
}

func putRelation(tx *bbolt.Tx, fromID string, relationType memory.RelationType, toID string) error {
	if err := appendRelationIDs(tx.Bucket(bucketRelations), relationBucketKey(fromID, relationType), toID); err != nil {
		return err
	}
	return appendRelationIDs(tx.Bucket(bucketRelationRefs), relationReverseBucketKey(toID, relationType), fromID)
}

func putLatest(tx *bbolt.Tx, rootID string, latestID string) error {
	return tx.Bucket(bucketLatest).Put([]byte(rootID), []byte(latestID))
}

func loadSpaceConfig(tx *bbolt.Tx, spaceID string) (*memory.SpaceConfig, error) {
	raw := tx.Bucket(bucketSpacesConfig).Get([]byte(spaceID))
	if raw == nil {
		return nil, memory.ErrSpaceNotFound
	}

	var cfg memory.SpaceConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func appendRelationIDs(bucket *bbolt.Bucket, key []byte, ids ...string) error {
	existing, err := loadRelationIDsFromBucket(bucket, key)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if !containsSourceID(existing, id) {
			existing = append(existing, id)
		}
	}
	encoded, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	return bucket.Put(key, encoded)
}

func loadRelationIDsFromBucket(bucket *bbolt.Bucket, key []byte) ([]string, error) {
	raw := bucket.Get(key)
	if raw == nil {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func newMemoryID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}

func cloneMetadata(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneEmbedding(in []float32) []float32 {
	if in == nil {
		return nil
	}
	out := make([]float32, len(in))
	copy(out, in)
	return out
}

func distinctStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func marshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func unmarshalMemory(raw []byte, mem *memory.Memory) error {
	return json.Unmarshal(raw, mem)
}
