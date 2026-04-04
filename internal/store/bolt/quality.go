package bolt

import (
	"cmp"
	"sort"
	"strings"
	"time"

	"omnethdb/internal/memory"
)

const (
	defaultQualityTopKPerMemory      = 5
	defaultQualityMaxDuplicateGroups = 8
	defaultQualityMaxUpdatePairs     = 12
)

func (s *Store) GetQualityDiagnostics(req memory.QualityDiagnosticsRequest) (*memory.QualityDiagnosticsResult, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(req.SpaceID); err != nil {
		return nil, err
	}

	topK := req.TopKPerMemory
	if topK <= 0 {
		topK = defaultQualityTopKPerMemory
	}
	maxGroups := req.MaxDuplicateGroups
	if maxGroups <= 0 {
		maxGroups = defaultQualityMaxDuplicateGroups
	}
	maxUpdates := req.MaxUpdatePairs
	if maxUpdates <= 0 {
		maxUpdates = defaultQualityMaxUpdatePairs
	}

	memories, err := s.ListMemories(memory.ListMemoriesRequest{
		SpaceIDs: []string{req.SpaceID},
		Kinds:    []memory.MemoryKind{memory.KindStatic},
	})
	if err != nil {
		return nil, err
	}

	duplicatePairs := make([]memory.MemorySimilarityPair, 0)
	updatePairs := make([]memory.MemorySimilarityPair, 0)
	seenDuplicatePairs := make(map[string]struct{})
	seenUpdatePairs := make(map[string]struct{})

	for _, mem := range memories {
		candidates, err := s.FindCandidates(memory.FindCandidatesRequest{
			SpaceID: req.SpaceID,
			Content: mem.Content,
			TopK:    topK + 1,
		})
		if err != nil {
			return nil, err
		}
		for _, candidate := range candidates {
			if candidate.ID == mem.ID || candidate.Kind != memory.KindStatic {
				continue
			}
			pair := memory.MemorySimilarityPair{
				LeftID:       mem.ID,
				RightID:      candidate.ID,
				LeftContent:  mem.Content,
				RightContent: candidate.Content,
				Score:        candidate.Score,
			}
			pairKey := canonicalPairKey(mem.ID, candidate.ID)
			switch {
			case candidate.Score >= duplicateScoreThreshold:
				if _, ok := seenDuplicatePairs[pairKey]; ok {
					continue
				}
				seenDuplicatePairs[pairKey] = struct{}{}
				duplicatePairs = append(duplicatePairs, normalizePairOrder(pair))
			case candidate.Score >= updateSuggestionScoreThreshold && candidate.Score < duplicateScoreThreshold:
				if _, ok := seenUpdatePairs[pairKey]; ok {
					continue
				}
				seenUpdatePairs[pairKey] = struct{}{}
				updatePairs = append(updatePairs, normalizePairOrder(pair))
			}
		}
	}

	sortSimilarityPairs(duplicatePairs)
	sortSimilarityPairs(updatePairs)

	result := &memory.QualityDiagnosticsResult{
		SpaceID:         req.SpaceID,
		GeneratedAt:     time.Now().UTC(),
		LiveStaticCount: len(memories),
		DuplicateGroups: buildDuplicateGroups(duplicatePairs, maxGroups),
		PossibleUpdates: clampPairs(updatePairs, maxUpdates),
	}
	return result, nil
}

func (s *Store) BuildQualityCleanupPlan(req memory.QualityCleanupPlanRequest) (*memory.QualityCleanupPlanResult, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(req.SpaceID); err != nil {
		return nil, err
	}

	maxActions := req.MaxDuplicateActions
	if maxActions <= 0 {
		maxActions = defaultQualityMaxDuplicateGroups
	}

	memories, err := s.ListMemories(memory.ListMemoriesRequest{
		SpaceIDs: []string{req.SpaceID},
		Kinds:    []memory.MemoryKind{memory.KindStatic},
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]memory.Memory, len(memories))
	for _, mem := range memories {
		byID[mem.ID] = mem
	}

	diagnostics, err := s.GetQualityDiagnostics(memory.QualityDiagnosticsRequest{
		SpaceID:            req.SpaceID,
		MaxDuplicateGroups: maxActions,
	})
	if err != nil {
		return nil, err
	}

	out := &memory.QualityCleanupPlanResult{
		SpaceID:     req.SpaceID,
		GeneratedAt: time.Now().UTC(),
	}
	for _, group := range diagnostics.DuplicateGroups {
		keep, forget := selectCleanupTargets(group.MemoryIDs, byID)
		if keep == "" || len(forget) == 0 {
			continue
		}
		out.DuplicateSuggestions = append(out.DuplicateSuggestions, memory.DuplicateCleanupSuggestion{
			KeepID:         keep,
			ForgetIDs:      forget,
			Rationale:      "advisory keep target chosen by highest confidence, then newest created_at, then stable memory id ordering; apply only after human review",
			DuplicateGroup: group,
		})
	}
	out.SuggestedForgetBatchCommand = buildSuggestedForgetBatchCommand(out.DuplicateSuggestions)
	return out, nil
}

func buildDuplicateGroups(pairs []memory.MemorySimilarityPair, maxGroups int) []memory.DuplicateDiagnosticGroup {
	if len(pairs) == 0 {
		return nil
	}

	adj := make(map[string]map[string]struct{})
	nodes := make(map[string]struct{})
	for _, pair := range pairs {
		nodes[pair.LeftID] = struct{}{}
		nodes[pair.RightID] = struct{}{}
		if adj[pair.LeftID] == nil {
			adj[pair.LeftID] = make(map[string]struct{})
		}
		if adj[pair.RightID] == nil {
			adj[pair.RightID] = make(map[string]struct{})
		}
		adj[pair.LeftID][pair.RightID] = struct{}{}
		adj[pair.RightID][pair.LeftID] = struct{}{}
	}

	visited := make(map[string]struct{}, len(nodes))
	groups := make([]memory.DuplicateDiagnosticGroup, 0)
	for node := range nodes {
		if _, ok := visited[node]; ok {
			continue
		}
		stack := []string{node}
		componentSet := make(map[string]struct{})
		componentIDs := make([]string, 0)
		for len(stack) > 0 {
			last := len(stack) - 1
			current := stack[last]
			stack = stack[:last]
			if _, ok := visited[current]; ok {
				continue
			}
			visited[current] = struct{}{}
			componentSet[current] = struct{}{}
			componentIDs = append(componentIDs, current)
			for neighbor := range adj[current] {
				if _, ok := visited[neighbor]; !ok {
					stack = append(stack, neighbor)
				}
			}
		}
		if len(componentIDs) < 2 {
			continue
		}
		sort.Strings(componentIDs)
		componentPairs := make([]memory.MemorySimilarityPair, 0)
		for _, pair := range pairs {
			if _, ok := componentSet[pair.LeftID]; !ok {
				continue
			}
			if _, ok := componentSet[pair.RightID]; !ok {
				continue
			}
			componentPairs = append(componentPairs, pair)
		}
		sortSimilarityPairs(componentPairs)
		groups = append(groups, memory.DuplicateDiagnosticGroup{
			MemoryIDs: componentIDs,
			Pairs:     componentPairs,
		})
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if len(groups[i].MemoryIDs) != len(groups[j].MemoryIDs) {
			return len(groups[i].MemoryIDs) > len(groups[j].MemoryIDs)
		}
		leftScore := float32(0)
		rightScore := float32(0)
		if len(groups[i].Pairs) > 0 {
			leftScore = groups[i].Pairs[0].Score
		}
		if len(groups[j].Pairs) > 0 {
			rightScore = groups[j].Pairs[0].Score
		}
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		return cmp.Less(groups[i].MemoryIDs[0], groups[j].MemoryIDs[0])
	})

	if len(groups) > maxGroups {
		return append([]memory.DuplicateDiagnosticGroup(nil), groups[:maxGroups]...)
	}
	return groups
}

func clampPairs(pairs []memory.MemorySimilarityPair, max int) []memory.MemorySimilarityPair {
	if len(pairs) > max {
		return append([]memory.MemorySimilarityPair(nil), pairs[:max]...)
	}
	return append([]memory.MemorySimilarityPair(nil), pairs...)
}

func sortSimilarityPairs(pairs []memory.MemorySimilarityPair) {
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].Score != pairs[j].Score {
			return pairs[i].Score > pairs[j].Score
		}
		if pairs[i].LeftID != pairs[j].LeftID {
			return cmp.Less(pairs[i].LeftID, pairs[j].LeftID)
		}
		return cmp.Less(pairs[i].RightID, pairs[j].RightID)
	})
}

func canonicalPairKey(leftID, rightID string) string {
	if leftID > rightID {
		leftID, rightID = rightID, leftID
	}
	return leftID + "|" + rightID
}

func normalizePairOrder(pair memory.MemorySimilarityPair) memory.MemorySimilarityPair {
	if pair.LeftID <= pair.RightID {
		return pair
	}
	return memory.MemorySimilarityPair{
		LeftID:       pair.RightID,
		RightID:      pair.LeftID,
		LeftContent:  pair.RightContent,
		RightContent: pair.LeftContent,
		Score:        pair.Score,
	}
}

func selectCleanupTargets(ids []string, byID map[string]memory.Memory) (string, []string) {
	if len(ids) < 2 {
		return "", nil
	}
	items := make([]memory.Memory, 0, len(ids))
	for _, id := range ids {
		mem, ok := byID[id]
		if !ok {
			continue
		}
		items = append(items, mem)
	}
	if len(items) < 2 {
		return "", nil
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Confidence != items[j].Confidence {
			return items[i].Confidence > items[j].Confidence
		}
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return cmp.Less(items[i].ID, items[j].ID)
	})
	keep := items[0].ID
	forget := make([]string, 0, len(items)-1)
	for _, mem := range items[1:] {
		forget = append(forget, mem.ID)
	}
	return keep, forget
}

func buildSuggestedForgetBatchCommand(items []memory.DuplicateCleanupSuggestion) string {
	if len(items) == 0 {
		return ""
	}
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	for _, item := range items {
		for _, id := range item.ForgetIDs {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return ""
	}
	sort.Strings(ids)
	return `omnethdb forget-batch --workspace . --ids ` + strings.Join(ids, ",") + ` --reason "duplicate cleanup"`
}
