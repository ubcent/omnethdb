package bolt

import (
	"cmp"
	"context"
	"math"
	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	"slices"
	"sort"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) Recall(req memory.RecallRequest) ([]memory.ScoredMemory, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if len(req.SpaceIDs) == 0 {
		return nil, nil
	}
	for _, spaceID := range req.SpaceIDs {
		if err := memory.ValidateSpaceID(spaceID); err != nil {
			return nil, err
		}
	}
	for _, kind := range req.Kinds {
		if err := memory.ValidateMemoryKind(kind); err != nil {
			return nil, err
		}
	}
	if req.TopK < 0 {
		return nil, memory.ErrInvalidContent
	}

	results, err := s.collectScoredMemories(req.SpaceIDs, req.SpaceWeights, req.Query, kindSet(req.Kinds), req.ExcludeOrphanedDerives)
	if err != nil {
		return nil, err
	}

	if req.TopK > 0 && len(results) > req.TopK {
		results = slices.Clone(results[:req.TopK])
	}

	return results, nil
}

func (s *Store) collectScoredMemories(spaceIDs []string, spaceWeights map[string]float32, query string, kinds map[memory.MemoryKind]struct{}, excludeOrphanedDerives bool) ([]memory.ScoredMemory, error) {
	now := time.Now().UTC()
	results := make([]memory.ScoredMemory, 0)
	configs, queryEmbeddings, err := s.prepareRecallInputs(spaceIDs, query)
	if err != nil {
		return nil, err
	}

	err = s.db.View(func(tx *bbolt.Tx) error {
		for _, spaceID := range spaceIDs {
			cfg := configs[spaceID]
			queryEmbedding := queryEmbeddings[cfg.EmbeddingModelID]

			for _, id := range loadSpaceMemoryIDs(tx, spaceID) {
				mem, err := loadMemory(tx, id)
				if err != nil {
					return err
				}
				if !isRecallEligible(mem, now, kinds, excludeOrphanedDerives) {
					continue
				}
				score := scoreMemory(*mem, queryEmbedding, *cfg, spaceWeights, now)
				results = append(results, memory.ScoredMemory{Memory: *mem, Score: score})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].CreatedAt.Equal(results[j].CreatedAt) {
			return cmp.Less(results[i].ID, results[j].ID)
		}
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	return results, nil
}

func (s *Store) prepareRecallInputs(spaceIDs []string, query string) (map[string]*memory.SpaceConfig, map[string][]float32, error) {
	configs := make(map[string]*memory.SpaceConfig, len(spaceIDs))
	if err := s.db.View(func(tx *bbolt.Tx) error {
		for _, spaceID := range spaceIDs {
			cfg, err := loadSpaceConfig(tx, spaceID)
			if err != nil {
				return err
			}
			configs[spaceID] = cfg
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}

	queryEmbeddings := make(map[string][]float32, len(configs))
	for _, cfg := range configs {
		if _, ok := queryEmbeddings[cfg.EmbeddingModelID]; ok {
			continue
		}
		embedder, err := s.lookupEmbedder(cfg.EmbeddingModelID)
		if err != nil {
			return nil, nil, err
		}
		queryEmbedding, err := embedder.Embed(context.Background(), query)
		if err != nil {
			return nil, nil, err
		}
		queryEmbeddings[cfg.EmbeddingModelID] = queryEmbedding
	}

	return configs, queryEmbeddings, nil
}

func scoreMemory(mem memory.Memory, queryEmbedding []float32, cfg memory.SpaceConfig, overrides map[string]float32, now time.Time) float32 {
	return cosine(queryEmbedding, mem.Embedding) *
		mem.Confidence *
		policy.ResolveActorTrust(cfg.WritePolicy, mem.Actor) *
		recencyFactor(mem, cfg.HalfLifeDays, now) *
		effectiveSpaceWeight(mem.SpaceID, cfg.DefaultWeight, overrides)
}

func recencyFactor(mem memory.Memory, halfLifeDays float32, now time.Time) float32 {
	if mem.Kind != memory.KindEpisodic {
		return 1.0
	}
	if halfLifeDays <= 0 {
		return 1.0
	}
	ageHours := now.Sub(mem.CreatedAt).Hours()
	if ageHours <= 0 {
		return 1.0
	}
	ageDays := ageHours / 24
	return float32(math.Exp((-math.Ln2 / float64(halfLifeDays)) * ageDays))
}

func effectiveSpaceWeight(spaceID string, defaultWeight float32, overrides map[string]float32) float32 {
	if overrides != nil {
		if weight, ok := overrides[spaceID]; ok {
			return weight
		}
	}
	if defaultWeight > 0 {
		return defaultWeight
	}
	return 1.0
}

func cosine(a []float32, b []float32) float32 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := min(len(a), len(b))
	var dot float64
	var magA float64
	var magB float64
	for i := 0; i < n; i++ {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		magA += av * av
		magB += bv * bv
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(magA) * math.Sqrt(magB)))
}

func (s *Store) lookupEmbedder(modelID string) (memory.Embedder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	embedder := s.embedders[modelID]
	if embedder == nil {
		return nil, memory.ErrEmbedderUnavailable
	}
	return embedder, nil
}

func isRecallEligible(mem *memory.Memory, now time.Time, kinds map[memory.MemoryKind]struct{}, excludeOrphanedDerives bool) bool {
	if !mem.IsLatest || mem.IsForgotten {
		return false
	}
	if mem.ForgetAfter != nil && !mem.ForgetAfter.After(now) {
		return false
	}
	if len(kinds) > 0 {
		if _, ok := kinds[mem.Kind]; !ok {
			return false
		}
	}
	if excludeOrphanedDerives && mem.Kind == memory.KindDerived && mem.HasOrphanedSources {
		return false
	}
	return true
}

func kindSet(kinds []memory.MemoryKind) map[memory.MemoryKind]struct{} {
	if len(kinds) == 0 {
		return nil
	}
	out := make(map[memory.MemoryKind]struct{}, len(kinds))
	for _, kind := range kinds {
		out[kind] = struct{}{}
	}
	return out
}
