package bolt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	"strings"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) Remember(input memory.MemoryInput) (*memory.Memory, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(input.SpaceID); err != nil {
		return nil, err
	}
	if err := memory.ValidateContent(input.Content); err != nil {
		return nil, err
	}
	if err := memory.ValidateMemoryKind(input.Kind); err != nil {
		return nil, err
	}
	if err := memory.ValidateActor(input.Actor); err != nil {
		return nil, err
	}
	if err := memory.ValidateConfidence(input.Confidence); err != nil {
		return nil, err
	}
	if err := validateRememberRelations(input.Relations); err != nil {
		return nil, err
	}

	mem := &memory.Memory{
		ID:          newMemoryID(),
		SpaceID:     input.SpaceID,
		Content:     input.Content,
		Kind:        input.Kind,
		Actor:       input.Actor,
		Confidence:  input.Confidence,
		ForgetAfter: input.ForgetAfter,
		Metadata:    cloneMetadata(input.Metadata),
		CreatedAt:   time.Now().UTC(),
		Relations: memory.MemoryRelations{
			Updates: append([]string(nil), input.Relations.Updates...),
			Extends: append([]string(nil), input.Relations.Extends...),
			Derives: append([]string(nil), input.Relations.Derives...),
		},
		SourceIDs: append([]string(nil), input.SourceIDs...),
		Rationale: input.Rationale,
	}

	err := s.db.Update(func(tx *bbolt.Tx) error {
		cfg, err := loadSpaceConfig(tx, input.SpaceID)
		if err != nil {
			return err
		}
		if err := ensureSpaceWritable(cfg); err != nil {
			return err
		}
		embedder, err := s.lookupEmbedder(cfg.EmbeddingModelID)
		if err != nil {
			return err
		}
		if !policy.CanWriteKind(cfg.WritePolicy, input.Actor, input.Kind) {
			return memory.ErrPolicyViolation
		}
		embedding, err := embedder.Embed(context.Background(), input.Content)
		if err != nil {
			return err
		}
		mem.Embedding = cloneEmbedding(embedding)

		liveCounts, err := countLiveKinds(tx, input.SpaceID)
		if err != nil {
			return err
		}

		if len(input.Relations.Updates) == 0 {
			if wouldExceedCorpusLimit(cfg.WritePolicy, liveCounts, input.Kind, memory.KindUnknown) {
				return memory.ErrCorpusLimit
			}
			mem.Version = 1
			mem.IsLatest = true
			if err := validateDerivedInput(tx, mem); err != nil {
				return err
			}
			if err := validateExtendsTargets(tx, mem.ID, input.SpaceID, input.Relations.Extends); err != nil {
				return err
			}
			if err := saveMemory(tx, mem); err != nil {
				return err
			}
			if err := saveEmbedding(tx, mem.ID, mem.Embedding); err != nil {
				return err
			}
			if err := putLatest(tx, mem.ID, mem.ID); err != nil {
				return err
			}
			if err := putSpaceMemoryIDs(tx, mem.SpaceID, appendSpaceMemoryID(loadSpaceMemoryIDs(tx, mem.SpaceID), mem.ID)); err != nil {
				return err
			}
			if err := saveRelations(tx, mem); err != nil {
				return err
			}
			return writeAuditEntry(tx, memory.AuditEntry{
				Timestamp: mem.CreatedAt,
				SpaceID:   mem.SpaceID,
				Operation: "remember",
				Actor:     mem.Actor,
				MemoryIDs: []string{mem.ID},
			})
		}

		targetID := input.Relations.Updates[0]
		target, err := loadMemory(tx, targetID)
		if err != nil {
			return err
		}
		if target.SpaceID != input.SpaceID {
			return memory.ErrUpdateAcrossSpaces
		}
		if target.Kind != input.Kind {
			if !(target.Kind == memory.KindEpisodic && input.Kind == memory.KindStatic) {
				return memory.ErrUpdateAcrossKinds
			}
			if !policy.CanPromote(cfg.WritePolicy, input.Actor) {
				return memory.ErrPolicyViolation
			}
		}
		if target.Kind == memory.KindEpisodic && input.Kind == memory.KindStatic && !policy.CanPromote(cfg.WritePolicy, input.Actor) {
			return memory.ErrPolicyViolation
		}
		if target.Kind != input.Kind && !(target.Kind == memory.KindEpisodic && input.Kind == memory.KindStatic) {
			return memory.ErrUpdateAcrossKinds
		}
		rootID := target.ID
		if target.RootID != nil {
			rootID = *target.RootID
		}
		if input.IfLatestID != nil {
			currentLatest := loadLatest(tx, rootID)
			if currentLatest != *input.IfLatestID {
				return memory.ErrConflict
			}
		}
		if target.IsForgotten {
			return memory.ErrUpdateTargetForgotten
		}
		if !target.IsLatest {
			return memory.ErrUpdateTargetNotLatest
		}
		if wouldExceedCorpusLimit(cfg.WritePolicy, liveCounts, input.Kind, target.Kind) {
			return memory.ErrCorpusLimit
		}
		if err := validateDerivedInput(tx, mem); err != nil {
			return err
		}
		if err := validateExtendsTargets(tx, mem.ID, input.SpaceID, input.Relations.Extends); err != nil {
			return err
		}

		target.IsLatest = false
		mem.Version = target.Version + 1
		mem.IsLatest = true
		mem.ParentID = &target.ID
		mem.RootID = &rootID

		if err := saveMemory(tx, target); err != nil {
			return err
		}
		if err := saveMemory(tx, mem); err != nil {
			return err
		}
		if err := saveEmbedding(tx, mem.ID, mem.Embedding); err != nil {
			return err
		}
		if err := putLatest(tx, rootID, mem.ID); err != nil {
			return err
		}

		spaceIDs := loadSpaceMemoryIDs(tx, mem.SpaceID)
		spaceIDs = replaceSpaceMemoryID(spaceIDs, target.ID, mem.ID)
		if err := putSpaceMemoryIDs(tx, mem.SpaceID, spaceIDs); err != nil {
			return err
		}
		if err := saveRelations(tx, mem); err != nil {
			return err
		}
		return writeAuditEntry(tx, memory.AuditEntry{
			Timestamp: mem.CreatedAt,
			SpaceID:   mem.SpaceID,
			Operation: "remember",
			Actor:     mem.Actor,
			MemoryIDs: []string{target.ID, mem.ID},
		})
	})
	if err != nil {
		return nil, err
	}

	return mem, nil
}

func countLiveKinds(tx *bbolt.Tx, spaceID string) (map[memory.MemoryKind]int, error) {
	counts := map[memory.MemoryKind]int{
		memory.KindEpisodic: 0,
		memory.KindStatic:   0,
		memory.KindDerived:  0,
	}

	for _, id := range loadSpaceMemoryIDs(tx, spaceID) {
		mem, err := loadMemory(tx, id)
		if err != nil {
			return nil, err
		}
		counts[mem.Kind]++
	}

	return counts, nil
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
		if err := putRelation(tx, mem.ID, "updates", toID); err != nil {
			return err
		}
	}
	for _, toID := range mem.Relations.Extends {
		if err := putRelation(tx, mem.ID, "extends", toID); err != nil {
			return err
		}
	}
	for _, toID := range mem.Relations.Derives {
		if err := putRelation(tx, mem.ID, "derives", toID); err != nil {
			return err
		}
	}
	return nil
}

func putRelation(tx *bbolt.Tx, fromID, relationType, toID string) error {
	key := []byte(fmt.Sprintf("%s:%s", fromID, relationType))

	b := tx.Bucket(bucketRelations)
	raw := b.Get(key)
	var ids []string
	if raw != nil {
		if err := json.Unmarshal(raw, &ids); err != nil {
			return err
		}
	}
	ids = append(ids, toID)

	encoded, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return b.Put(key, encoded)
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

func loadSpaceMemoryIDs(tx *bbolt.Tx, spaceID string) []string {
	raw := tx.Bucket(bucketSpaces).Get([]byte(spaceID))
	if raw == nil {
		return nil
	}

	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil
	}
	return ids
}

func putSpaceMemoryIDs(tx *bbolt.Tx, spaceID string, ids []string) error {
	encoded, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return tx.Bucket(bucketSpaces).Put([]byte(spaceID), encoded)
}

func appendSpaceMemoryID(ids []string, id string) []string {
	return append(ids, id)
}

func replaceSpaceMemoryID(ids []string, oldID string, newID string) []string {
	out := make([]string, 0, len(ids))
	replaced := false
	for _, id := range ids {
		if id == oldID {
			out = append(out, newID)
			replaced = true
			continue
		}
		out = append(out, id)
	}
	if !replaced {
		out = append(out, newID)
	}
	return out
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
