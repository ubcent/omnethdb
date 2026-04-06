package bolt

import (
	"cmp"
	"sort"
	"strconv"
	"strings"
	"time"

	"omnethdb/internal/memory"

	bbolt "go.etcd.io/bbolt"
)

const (
	defaultQualityTopKPerMemory      = 5
	defaultQualityMaxDuplicateGroups = 8
	defaultQualityMaxUpdatePairs     = 12
	defaultSynthesisMaxCandidates    = 8
	defaultSynthesisMinClusterSize   = 2
	defaultPromotionMaxSuggestions   = 8
	defaultPromotionMinObservations  = 2
	defaultPromotionMinActors        = 1
	defaultPromotionMinWindows       = 1
	synthesisCandidateScoreThreshold = 0.90
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

func (s *Store) GetSynthesisCandidates(req memory.SynthesisCandidatesRequest) (*memory.SynthesisCandidatesResult, error) {
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
	maxCandidates := req.MaxCandidates
	if maxCandidates <= 0 {
		maxCandidates = defaultSynthesisMaxCandidates
	}
	minClusterSize := req.MinClusterSize
	if minClusterSize <= 0 {
		minClusterSize = defaultSynthesisMinClusterSize
	}
	minPairScore := req.MinPairScore
	if minPairScore <= 0 {
		minPairScore = synthesisCandidateScoreThreshold
	}

	memories, err := s.findCurationMemories(req.SpaceID, req.IncludeSuperseded, req.IncludeForgotten)
	if err != nil {
		return nil, err
	}

	pairs, err := s.findEpisodicSimilarityPairs(req.SpaceID, memories, topK, minPairScore)
	if err != nil {
		return nil, err
	}

	result := &memory.SynthesisCandidatesResult{
		SpaceID:           req.SpaceID,
		GeneratedAt:       time.Now().UTC(),
		LiveEpisodicCount: len(memories),
		Candidates:        buildSynthesisCandidates(memories, pairs, minClusterSize, maxCandidates),
	}
	return result, nil
}

func (s *Store) GetPromotionSuggestions(req memory.PromotionSuggestionsRequest) (*memory.PromotionSuggestionsResult, error) {
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
	maxSuggestions := req.MaxSuggestions
	if maxSuggestions <= 0 {
		maxSuggestions = defaultPromotionMaxSuggestions
	}
	minObservationCount := req.MinObservationCount
	if minObservationCount <= 0 {
		minObservationCount = defaultPromotionMinObservations
	}
	minDistinctActors := req.MinDistinctActors
	if minDistinctActors <= 0 {
		minDistinctActors = defaultPromotionMinActors
	}
	minDistinctWindows := req.MinDistinctWindows
	if minDistinctWindows <= 0 {
		minDistinctWindows = defaultPromotionMinWindows
	}
	minCumulativeScore := req.MinCumulativeScore
	if minCumulativeScore <= 0 {
		minCumulativeScore = float32(minObservationCount) * synthesisCandidateScoreThreshold
	}
	minPairScore := req.MinPairScore
	if minPairScore <= 0 {
		minPairScore = synthesisCandidateScoreThreshold
	}

	memories, err := s.findCurationMemories(req.SpaceID, req.IncludeSuperseded, req.IncludeForgotten)
	if err != nil {
		return nil, err
	}
	pairs, err := s.findEpisodicSimilarityPairs(req.SpaceID, memories, topK, minPairScore)
	if err != nil {
		return nil, err
	}

	result := &memory.PromotionSuggestionsResult{
		SpaceID:           req.SpaceID,
		GeneratedAt:       time.Now().UTC(),
		LiveEpisodicCount: len(memories),
		Suggestions: buildPromotionSuggestions(
			memories,
			pairs,
			minObservationCount,
			minDistinctActors,
			minDistinctWindows,
			minCumulativeScore,
			maxSuggestions,
		),
	}
	return result, nil
}

func (s *Store) findCurationMemories(spaceID string, includeSuperseded, includeForgotten bool) ([]memory.Memory, error) {
	if !includeSuperseded && !includeForgotten {
		return s.ListMemories(memory.ListMemoriesRequest{
			SpaceIDs: []string{spaceID},
			Kinds:    []memory.MemoryKind{memory.KindEpisodic},
		})
	}

	out := make([]memory.Memory, 0)
	now := time.Now().UTC()
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMemories)
		return b.ForEach(func(key, raw []byte) error {
			var mem memory.Memory
			if err := unmarshalMemory(raw, &mem); err != nil {
				return err
			}
			if mem.SpaceID != spaceID {
				return nil
			}
			if mem.Kind != memory.KindEpisodic {
				return nil
			}
			if !includeSuperseded && !mem.IsLatest {
				return nil
			}
			if !includeForgotten {
				if mem.IsForgotten {
					return nil
				}
				if mem.ForgetAfter != nil && !mem.ForgetAfter.After(now) {
					return nil
				}
			}
			out = append(out, mem)
			_ = key
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return cmp.Less(out[i].ID, out[j].ID)
	})
	return out, nil
}

func (s *Store) findEpisodicSimilarityPairs(spaceID string, memories []memory.Memory, topK int, minPairScore float32) ([]memory.MemorySimilarityPair, error) {
	pairs := make([]memory.MemorySimilarityPair, 0)
	seen := make(map[string]struct{})
	for _, mem := range memories {
		candidates, err := s.FindCandidates(memory.FindCandidatesRequest{
			SpaceID:           spaceID,
			Content:           mem.Content,
			TopK:              topK + 1,
			IncludeSuperseded: false,
			IncludeForgotten:  false,
		})
		if err != nil {
			return nil, err
		}
		for _, candidate := range candidates {
			if candidate.ID == mem.ID || candidate.Kind != memory.KindEpisodic || candidate.Score < minPairScore {
				continue
			}
			pairKey := canonicalPairKey(mem.ID, candidate.ID)
			if _, ok := seen[pairKey]; ok {
				continue
			}
			seen[pairKey] = struct{}{}
			pairs = append(pairs, normalizePairOrder(memory.MemorySimilarityPair{
				LeftID:       mem.ID,
				RightID:      candidate.ID,
				LeftContent:  mem.Content,
				RightContent: candidate.Content,
				Score:        candidate.Score,
			}))
		}
	}
	sortSimilarityPairs(pairs)
	return pairs, nil
}

func buildSynthesisCandidates(memories []memory.Memory, pairs []memory.MemorySimilarityPair, minClusterSize, maxCandidates int) []memory.SynthesisCandidate {
	groups := buildSimilarityGroups(memories, pairs, minClusterSize)
	if len(groups) == 0 {
		return nil
	}

	out := make([]memory.SynthesisCandidate, 0, len(groups))
	for idx, group := range groups {
		reasonCodes := []string{"high_similarity_cluster"}
		if group.distinctActors > 1 {
			reasonCodes = append(reasonCodes, "cross_actor_confirmation")
		}
		if !group.start.Equal(group.end) {
			reasonCodes = append(reasonCodes, "stable_across_time")
		}
		if group.clusterSize >= 3 {
			reasonCodes = append(reasonCodes, "repeated_observation")
		}
		if group.hasBlob {
			reasonCodes = append(reasonCodes, "mixed_fact_blob")
		}
		if group.hasContradiction {
			reasonCodes = append(reasonCodes, "possible_contradiction")
		}
		if group.hasEpisodicChatter {
			reasonCodes = append(reasonCodes, "episodic_chatter")
		}

		action := "review_for_derived"
		if group.hasBlob || group.hasContradiction || group.hasEpisodicChatter {
			action = "review_only"
		}

		out = append(out, memory.SynthesisCandidate{
			CandidateID:     synthesisCandidateID(idx, group.ids),
			AdvisoryOnly:    true,
			ReasonCodes:     reasonCodes,
			SuggestedAction: action,
			ReviewScore:     group.meanScore * float32(group.clusterSize),
			ClusterSize:     group.clusterSize,
			DistinctActors:  group.distinctActors,
			DistinctSpaces:  group.distinctSpaces,
			TimeSpanStart:   group.start,
			TimeSpanEnd:     group.end,
			MeanSimilarity:  group.meanScore,
			MinSimilarity:   group.minScore,
			MaxSimilarity:   group.maxScore,
			Members:         group.members,
		})
	}
	if len(out) > maxCandidates {
		return append([]memory.SynthesisCandidate(nil), out[:maxCandidates]...)
	}
	return out
}

func buildPromotionSuggestions(memories []memory.Memory, pairs []memory.MemorySimilarityPair, minObservationCount, minDistinctActors, minDistinctWindows int, minCumulativeScore float32, maxSuggestions int) []memory.PromotionSuggestion {
	groups := buildSimilarityGroups(memories, pairs, minObservationCount)
	if len(groups) == 0 {
		return nil
	}

	out := make([]memory.PromotionSuggestion, 0, len(groups))
	for _, group := range groups {
		if group.clusterSize < minObservationCount || group.distinctActors < minDistinctActors || group.distinctWindows < minDistinctWindows {
			continue
		}
		cumulative := group.meanScore * float32(group.clusterSize)
		if cumulative < minCumulativeScore {
			continue
		}
		representative := selectPromotionRepresentative(group.sourceMemories)
		if representative.ID == "" {
			continue
		}

		normalizedCount := len(group.normalizedContents)
		diversityRisk := float32(0)
		if group.clusterSize > 0 {
			diversityRisk = float32(normalizedCount-1) / float32(group.clusterSize)
			if diversityRisk < 0 {
				diversityRisk = 0
			}
		}
		churnRisk := diversityRisk * maxFloat32(0, 1-group.meanScore)
		contradictionRisk := churnRisk * 0.5
		if group.hasContradiction {
			churnRisk = maxFloat32(churnRisk, diversityRisk)
			contradictionRisk = maxFloat32(contradictionRisk, diversityRisk*0.75)
		}
		reasonCodes := []string{"high_cumulative_score", "repeated_occurrence"}
		if group.distinctActors > 1 {
			reasonCodes = append(reasonCodes, "cross_actor_confirmation")
		}
		if group.distinctWindows > 1 {
			reasonCodes = append(reasonCodes, "stable_across_time")
		}
		if group.hasBlob {
			reasonCodes = append(reasonCodes, "mixed_fact_blob")
		}
		if group.hasEpisodicChatter {
			reasonCodes = append(reasonCodes, "episodic_chatter")
		}
		if churnRisk > 0.25 {
			reasonCodes = append(reasonCodes, "high_churn")
		}
		if contradictionRisk > 0.20 {
			reasonCodes = append(reasonCodes, "possible_contradiction")
		}
		if group.distinctActors < 2 && group.distinctWindows < 2 {
			reasonCodes = append(reasonCodes, "weak_independent_support")
		}

		action := "review_for_promotion"
		if group.hasBlob || group.hasContradiction || group.hasEpisodicChatter || churnRisk > 0.35 || (group.distinctActors < 2 && group.distinctWindows < 2) {
			action = "review_only"
		}

		out = append(out, memory.PromotionSuggestion{
			MemoryID:            representative.ID,
			AdvisoryOnly:        true,
			ReasonCodes:         reasonCodes,
			SuggestedAction:     action,
			ObservationCount:    group.clusterSize,
			DistinctTimeWindows: group.distinctWindows,
			DistinctActors:      group.distinctActors,
			CumulativeScore:     cumulative,
			LatestScore:         group.maxScore,
			FirstSeenAt:         group.start,
			LastSeenAt:          group.end,
			ChurnRisk:           churnRisk,
			ContradictionRisk:   contradictionRisk,
			Explanation:         buildPromotionExplanation(representative, group, action),
			Memory: memory.PromotionSuggestionMemory{
				MemoryID:    representative.ID,
				SpaceID:     representative.SpaceID,
				ActorID:     representative.Actor.ID,
				CreatedAt:   representative.CreatedAt,
				Content:     representative.Content,
				IsLatest:    representative.IsLatest,
				IsForgotten: representative.IsForgotten,
			},
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CumulativeScore != out[j].CumulativeScore {
			return out[i].CumulativeScore > out[j].CumulativeScore
		}
		if !out[i].LastSeenAt.Equal(out[j].LastSeenAt) {
			return out[i].LastSeenAt.After(out[j].LastSeenAt)
		}
		return cmp.Less(out[i].MemoryID, out[j].MemoryID)
	})
	if len(out) > maxSuggestions {
		return append([]memory.PromotionSuggestion(nil), out[:maxSuggestions]...)
	}
	return out
}

type similarityGroup struct {
	ids                []string
	sourceMemories     []memory.Memory
	members            []memory.SynthesisCandidateMember
	normalizedContents map[string]struct{}
	clusterSize        int
	distinctActors     int
	distinctSpaces     int
	distinctWindows    int
	start              time.Time
	end                time.Time
	meanScore          float32
	minScore           float32
	maxScore           float32
	hasBlob            bool
	hasContradiction   bool
	hasEpisodicChatter bool
}

func buildSimilarityGroups(memories []memory.Memory, pairs []memory.MemorySimilarityPair, minClusterSize int) []similarityGroup {
	if len(pairs) == 0 || len(memories) == 0 {
		return nil
	}

	byID := make(map[string]memory.Memory, len(memories))
	for _, mem := range memories {
		byID[mem.ID] = mem
	}

	adj := make(map[string]map[string]struct{})
	nodes := make(map[string]struct{})
	for _, pair := range pairs {
		if _, ok := byID[pair.LeftID]; !ok {
			continue
		}
		if _, ok := byID[pair.RightID]; !ok {
			continue
		}
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
	groups := make([]similarityGroup, 0)
	for node := range nodes {
		if _, ok := visited[node]; ok {
			continue
		}
		stack := []string{node}
		componentSet := make(map[string]struct{})
		componentIDs := make([]string, 0)
		componentMemories := make([]memory.Memory, 0)
		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if _, ok := visited[current]; ok {
				continue
			}
			visited[current] = struct{}{}
			componentSet[current] = struct{}{}
			componentIDs = append(componentIDs, current)
			componentMemories = append(componentMemories, byID[current])
			for neighbor := range adj[current] {
				if _, ok := visited[neighbor]; !ok {
					stack = append(stack, neighbor)
				}
			}
		}
		if len(componentIDs) < minClusterSize {
			continue
		}
		sort.Strings(componentIDs)
		sort.SliceStable(componentMemories, func(i, j int) bool {
			if !componentMemories[i].CreatedAt.Equal(componentMemories[j].CreatedAt) {
				return componentMemories[i].CreatedAt.Before(componentMemories[j].CreatedAt)
			}
			return cmp.Less(componentMemories[i].ID, componentMemories[j].ID)
		})

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
		groups = append(groups, summarizeSimilarityGroup(componentIDs, componentMemories, componentPairs))
	}

	sort.SliceStable(groups, func(i, j int) bool {
		left := groups[i].meanScore * float32(groups[i].clusterSize)
		right := groups[j].meanScore * float32(groups[j].clusterSize)
		if left != right {
			return left > right
		}
		return cmp.Less(groups[i].ids[0], groups[j].ids[0])
	})
	return groups
}

func summarizeSimilarityGroup(ids []string, memories []memory.Memory, pairs []memory.MemorySimilarityPair) similarityGroup {
	group := similarityGroup{
		ids:                append([]string(nil), ids...),
		sourceMemories:     append([]memory.Memory(nil), memories...),
		clusterSize:        len(memories),
		normalizedContents: make(map[string]struct{}),
		minScore:           1.0,
	}
	actors := make(map[string]struct{})
	spaces := make(map[string]struct{})
	windows := make(map[string]struct{})
	scoreSum := float32(0)

	for _, mem := range memories {
		actors[mem.Actor.ID] = struct{}{}
		spaces[mem.SpaceID] = struct{}{}
		windows[mem.CreatedAt.UTC().Format("2006-01-02")] = struct{}{}
		group.normalizedContents[normalizeLintText(mem.Content)] = struct{}{}
		group.hasBlob = group.hasBlob || detectMixedFactBlob(memory.MemoryInput{Content: mem.Content}) != nil
		group.hasContradiction = group.hasContradiction || contentHasContradictionCue(mem.Content)
		group.hasEpisodicChatter = group.hasEpisodicChatter || contentHasEpisodicChatterCue(mem.Content)
		if group.start.IsZero() || mem.CreatedAt.Before(group.start) {
			group.start = mem.CreatedAt
		}
		if group.end.IsZero() || mem.CreatedAt.After(group.end) {
			group.end = mem.CreatedAt
		}
		group.members = append(group.members, memory.SynthesisCandidateMember{
			MemoryID:    mem.ID,
			SpaceID:     mem.SpaceID,
			ActorID:     mem.Actor.ID,
			CreatedAt:   mem.CreatedAt,
			Content:     mem.Content,
			IsLatest:    mem.IsLatest,
			IsForgotten: mem.IsForgotten,
		})
	}
	group.distinctActors = len(actors)
	group.distinctSpaces = len(spaces)
	group.distinctWindows = len(windows)

	if len(pairs) == 0 {
		group.minScore = 0
		return group
	}
	for _, pair := range pairs {
		scoreSum += pair.Score
		if pair.Score > group.maxScore {
			group.maxScore = pair.Score
		}
		if pair.Score < group.minScore {
			group.minScore = pair.Score
		}
	}
	group.meanScore = scoreSum / float32(len(pairs))
	return group
}

func selectPromotionRepresentative(memories []memory.Memory) memory.Memory {
	best := memory.Memory{}
	for _, mem := range memories {
		if best.ID == "" {
			best = mem
			continue
		}
		if mem.Confidence != best.Confidence {
			if mem.Confidence > best.Confidence {
				best = mem
			}
			continue
		}
		if !mem.CreatedAt.Equal(best.CreatedAt) {
			if mem.CreatedAt.After(best.CreatedAt) {
				best = mem
			}
			continue
		}
		if cmp.Less(mem.ID, best.ID) {
			best = mem
		}
	}
	return best
}

func buildPromotionExplanation(mem memory.Memory, group similarityGroup, action string) string {
	parts := []string{
		"episodic memory recurs across similar live observations",
	}
	if group.distinctActors > 1 {
		parts = append(parts, "multiple actors reinforced the claim")
	}
	if group.distinctWindows > 1 {
		parts = append(parts, "the claim remained present across separate time windows")
	}
	if action == "review_only" {
		parts = append(parts, "review should stay cautious because the support cluster is noisy")
	}
	return strings.Join(parts, "; ") + ": " + mem.ID
}

func synthesisCandidateID(index int, ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return "synth-" + strings.Join([]string{ids[0], strconv.Itoa(index + 1)}, "-")
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func contentHasContradictionCue(content string) bool {
	normalized := " " + normalizeLintText(content) + " "
	for _, marker := range []string{
		" optional ",
		" optionally ",
		" except ",
		" unless ",
		" emergency ",
		" emergencies ",
		" temporary ",
		" temporarily ",
		" not required ",
		" no longer ",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func contentHasEpisodicChatterCue(content string) bool {
	normalized := " " + normalizeLintText(content) + " "
	for _, marker := range []string{
		" investigating ",
		" investigation ",
		" triage ",
		" triaged ",
		" mitigation ",
		" mitigated ",
		" retry ",
		" retried ",
		" retrying ",
		" monitoring ",
		" noisy ",
		" on call ",
		" workaround ",
		" rollback ",
		" rolled back ",
		" temporary ",
		" unblock ",
		" debugging ",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
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
