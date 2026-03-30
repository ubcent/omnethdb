package bolt

import (
	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	"slices"
)

func (s *Store) GetProfile(req memory.ProfileRequest) (*memory.MemoryProfile, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if len(req.SpaceIDs) == 0 {
		return &memory.MemoryProfile{}, nil
	}
	for _, spaceID := range req.SpaceIDs {
		if err := memory.ValidateSpaceID(spaceID); err != nil {
			return nil, err
		}
	}
	if req.StaticTopK < 0 || req.EpisodicTopK < 0 {
		return nil, memory.ErrInvalidContent
	}

	staticTopK := req.StaticTopK
	episodicTopK := req.EpisodicTopK
	if staticTopK == 0 || episodicTopK == 0 {
		defaultStatic, defaultEpisodic, err := s.resolveProfileLimits(req.SpaceIDs)
		if err != nil {
			return nil, err
		}
		if staticTopK == 0 {
			staticTopK = defaultStatic
		}
		if episodicTopK == 0 {
			episodicTopK = defaultEpisodic
		}
	}

	staticResults, err := s.collectScoredMemories(
		req.SpaceIDs,
		req.SpaceWeights,
		req.Query,
		kindSet([]memory.MemoryKind{memory.KindStatic, memory.KindDerived}),
		req.ExcludeOrphanedDerives,
	)
	if err != nil {
		return nil, err
	}
	episodicResults, err := s.collectScoredMemories(
		req.SpaceIDs,
		req.SpaceWeights,
		req.Query,
		kindSet([]memory.MemoryKind{memory.KindEpisodic}),
		req.ExcludeOrphanedDerives,
	)
	if err != nil {
		return nil, err
	}

	if staticTopK > 0 && len(staticResults) > staticTopK {
		staticResults = slices.Clone(staticResults[:staticTopK])
	}
	if episodicTopK > 0 && len(episodicResults) > episodicTopK {
		episodicResults = slices.Clone(episodicResults[:episodicTopK])
	}

	return &memory.MemoryProfile{
		Static:   staticResults,
		Episodic: episodicResults,
	}, nil
}

func (s *Store) resolveProfileLimits(spaceIDs []string) (int, int, error) {
	maxStatic := 0
	maxEpisodic := 0
	for _, spaceID := range spaceIDs {
		cfg, err := s.GetSpaceConfig(spaceID)
		if err != nil {
			return 0, 0, err
		}
		normalized := policy.NormalizeSpaceWritePolicy(cfg.WritePolicy)
		if normalized.ProfileMaxStatic > maxStatic {
			maxStatic = normalized.ProfileMaxStatic
		}
		if normalized.ProfileMaxEpisodic > maxEpisodic {
			maxEpisodic = normalized.ProfileMaxEpisodic
		}
	}
	return maxStatic, maxEpisodic, nil
}
