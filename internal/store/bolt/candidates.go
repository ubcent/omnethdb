package bolt

import (
	"cmp"
	"context"
	"encoding/json"
	"omnethdb/internal/memory"
	"slices"
	"sort"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) FindCandidates(req memory.FindCandidatesRequest) ([]memory.ScoredMemory, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(req.SpaceID); err != nil {
		return nil, err
	}
	if err := memory.ValidateContent(req.Content); err != nil {
		return nil, err
	}
	if req.TopK < 0 {
		return nil, memory.ErrInvalidContent
	}

	now := time.Now().UTC()
	results := make([]memory.ScoredMemory, 0)

	err := s.db.View(func(tx *bbolt.Tx) error {
		cfg, err := loadSpaceConfig(tx, req.SpaceID)
		if err != nil {
			return err
		}
		embedder, err := s.lookupEmbedder(cfg.EmbeddingModelID)
		if err != nil {
			return err
		}
		queryEmbedding, err := embedder.Embed(context.Background(), req.Content)
		if err != nil {
			return err
		}

		b := tx.Bucket(bucketMemories)
		c := b.Cursor()
		for _, raw := c.First(); raw != nil; _, raw = c.Next() {
			var mem memory.Memory
			if err := json.Unmarshal(raw, &mem); err != nil {
				return err
			}
			if mem.SpaceID != req.SpaceID {
				continue
			}
			if !isCandidateEligible(mem, now, req.IncludeSuperseded, req.IncludeForgotten) {
				continue
			}
			results = append(results, memory.ScoredMemory{
				Memory: mem,
				Score:  cosine(queryEmbedding, mem.Embedding),
			})
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

	if req.TopK > 0 && len(results) > req.TopK {
		results = slices.Clone(results[:req.TopK])
	}

	return results, nil
}

func isCandidateEligible(mem memory.Memory, now time.Time, includeSuperseded bool, includeForgotten bool) bool {
	if !includeSuperseded && !mem.IsLatest {
		return false
	}
	if !includeForgotten && mem.IsForgotten {
		return false
	}
	if mem.ForgetAfter != nil && !mem.ForgetAfter.After(now) && !includeForgotten {
		return false
	}
	return true
}
