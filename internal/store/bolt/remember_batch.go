package bolt

import (
	"context"
	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	"time"

	bbolt "go.etcd.io/bbolt"
)

type preparedRemember struct {
	input     memory.MemoryInput
	embedding []float32
}

func (s *Store) RememberBatch(inputs []memory.MemoryInput) ([]memory.Memory, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if len(inputs) == 0 {
		return nil, nil
	}

	prepared, err := s.prepareRememberInputs(inputs)
	if err != nil {
		return nil, err
	}

	results := make([]memory.Memory, 0, len(prepared))
	err = s.db.Update(func(tx *bbolt.Tx) error {
		for _, item := range prepared {
			mem, err := s.rememberPreparedInTx(tx, item)
			if err != nil {
				return err
			}
			results = append(results, *mem)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Store) prepareRememberInputs(inputs []memory.MemoryInput) ([]preparedRemember, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	prepared := make([]preparedRemember, 0, len(inputs))
	configs, err := s.loadSpaceConfigsForInputs(inputs)
	if err != nil {
		return nil, err
	}
	for _, input := range inputs {
		if err := validateRememberInput(input); err != nil {
			return nil, err
		}
		cfg := configs[input.SpaceID]
		if err := ensureSpaceWritable(cfg); err != nil {
			return nil, err
		}
		embedder, err := s.lookupEmbedder(cfg.EmbeddingModelID)
		if err != nil {
			return nil, err
		}
		embedding, err := embedder.Embed(context.Background(), input.Content)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, preparedRemember{
			input:     input,
			embedding: cloneEmbedding(embedding),
		})
	}
	return prepared, nil
}

func (s *Store) loadSpaceConfigsForInputs(inputs []memory.MemoryInput) (map[string]*memory.SpaceConfig, error) {
	spaceIDs := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		spaceIDs[input.SpaceID] = struct{}{}
	}

	configs := make(map[string]*memory.SpaceConfig, len(spaceIDs))
	err := s.db.View(func(tx *bbolt.Tx) error {
		for spaceID := range spaceIDs {
			cfg, err := loadSpaceConfig(tx, spaceID)
			if err != nil {
				return err
			}
			configs[spaceID] = cfg
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func validateRememberInput(input memory.MemoryInput) error {
	if err := memory.ValidateSpaceID(input.SpaceID); err != nil {
		return err
	}
	if err := memory.ValidateContent(input.Content); err != nil {
		return err
	}
	if err := memory.ValidateMemoryKind(input.Kind); err != nil {
		return err
	}
	if err := memory.ValidateActor(input.Actor); err != nil {
		return err
	}
	if err := memory.ValidateConfidence(input.Confidence); err != nil {
		return err
	}
	return validateRememberRelations(input.Relations)
}

func (s *Store) rememberPreparedInTx(tx *bbolt.Tx, prepared preparedRemember) (*memory.Memory, error) {
	input := prepared.input
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
		Embedding: cloneEmbedding(prepared.embedding),
	}

	cfg, err := loadSpaceConfig(tx, input.SpaceID)
	if err != nil {
		return nil, err
	}
	if err := ensureSpaceWritable(cfg); err != nil {
		return nil, err
	}
	if !policy.CanWriteKind(cfg.WritePolicy, input.Actor, input.Kind) {
		return nil, memory.ErrPolicyViolation
	}

	liveCounts, err := countLiveKinds(tx, input.SpaceID)
	if err != nil {
		return nil, err
	}

	if len(input.Relations.Updates) == 0 {
		if wouldExceedCorpusLimit(cfg.WritePolicy, liveCounts, input.Kind, memory.KindUnknown) {
			return nil, memory.ErrCorpusLimit
		}
		mem.Version = 1
		mem.IsLatest = true
		if err := validateDerivedInput(tx, mem); err != nil {
			return nil, err
		}
		if err := validateExtendsTargets(tx, mem.ID, input.SpaceID, input.Relations.Extends); err != nil {
			return nil, err
		}
		if err := saveMemory(tx, mem); err != nil {
			return nil, err
		}
		if err := saveEmbedding(tx, mem.ID, mem.Embedding); err != nil {
			return nil, err
		}
		if err := putLatest(tx, mem.ID, mem.ID); err != nil {
			return nil, err
		}
		if err := addSpaceMemoryID(tx, mem.SpaceID, mem.ID); err != nil {
			return nil, err
		}
		if err := incrementLiveKindCount(tx, mem.SpaceID, mem.Kind, 1); err != nil {
			return nil, err
		}
		if err := saveRelations(tx, mem); err != nil {
			return nil, err
		}
		if err := writeAuditEntry(tx, memory.AuditEntry{
			Timestamp: mem.CreatedAt,
			SpaceID:   mem.SpaceID,
			Operation: "remember",
			Actor:     mem.Actor,
			MemoryIDs: []string{mem.ID},
		}); err != nil {
			return nil, err
		}
		return mem, nil
	}

	targetID := input.Relations.Updates[0]
	target, err := loadMemory(tx, targetID)
	if err != nil {
		return nil, err
	}
	if target.SpaceID != input.SpaceID {
		return nil, memory.ErrUpdateAcrossSpaces
	}
	if target.Kind != input.Kind {
		if !(target.Kind == memory.KindEpisodic && input.Kind == memory.KindStatic) {
			return nil, memory.ErrUpdateAcrossKinds
		}
		if !policy.CanPromote(cfg.WritePolicy, input.Actor) {
			return nil, memory.ErrPolicyViolation
		}
	}
	if target.Kind == memory.KindEpisodic && input.Kind == memory.KindStatic && !policy.CanPromote(cfg.WritePolicy, input.Actor) {
		return nil, memory.ErrPolicyViolation
	}
	if target.Kind != input.Kind && !(target.Kind == memory.KindEpisodic && input.Kind == memory.KindStatic) {
		return nil, memory.ErrUpdateAcrossKinds
	}
	rootID := target.ID
	if target.RootID != nil {
		rootID = *target.RootID
	}
	if input.IfLatestID != nil {
		currentLatest := loadLatest(tx, rootID)
		if currentLatest != *input.IfLatestID {
			return nil, memory.ErrConflict
		}
	}
	if target.IsForgotten {
		return nil, memory.ErrUpdateTargetForgotten
	}
	if !target.IsLatest {
		return nil, memory.ErrUpdateTargetNotLatest
	}
	if wouldExceedCorpusLimit(cfg.WritePolicy, liveCounts, input.Kind, target.Kind) {
		return nil, memory.ErrCorpusLimit
	}
	if err := validateDerivedInput(tx, mem); err != nil {
		return nil, err
	}
	if err := validateExtendsTargets(tx, mem.ID, input.SpaceID, input.Relations.Extends); err != nil {
		return nil, err
	}

	target.IsLatest = false
	mem.Version = target.Version + 1
	mem.IsLatest = true
	mem.ParentID = &target.ID
	mem.RootID = &rootID

	if err := saveMemory(tx, target); err != nil {
		return nil, err
	}
	if err := saveMemory(tx, mem); err != nil {
		return nil, err
	}
	if err := saveEmbedding(tx, mem.ID, mem.Embedding); err != nil {
		return nil, err
	}
	if err := putLatest(tx, rootID, mem.ID); err != nil {
		return nil, err
	}
	if err := removeSpaceMemoryID(tx, mem.SpaceID, target.ID); err != nil {
		return nil, err
	}
	if err := addSpaceMemoryID(tx, mem.SpaceID, mem.ID); err != nil {
		return nil, err
	}
	if target.Kind != mem.Kind {
		if err := incrementLiveKindCount(tx, mem.SpaceID, target.Kind, -1); err != nil {
			return nil, err
		}
		if err := incrementLiveKindCount(tx, mem.SpaceID, mem.Kind, 1); err != nil {
			return nil, err
		}
	}
	if err := saveRelations(tx, mem); err != nil {
		return nil, err
	}
	if err := writeAuditEntry(tx, memory.AuditEntry{
		Timestamp: mem.CreatedAt,
		SpaceID:   mem.SpaceID,
		Operation: "remember",
		Actor:     mem.Actor,
		MemoryIDs: []string{target.ID, mem.ID},
	}); err != nil {
		return nil, err
	}
	return mem, nil
}
