package bolt

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"omnethdb/internal/memory"
)

const (
	defaultRememberLintTopK        = 3
	duplicateScoreThreshold        = 0.97
	updateSuggestionScoreThreshold = 0.82
	mixedFactBlobRuneThreshold     = 220
)

func (s *Store) LintRemember(req memory.RememberLintRequest) (*memory.RememberLintResult, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(req.Input.SpaceID); err != nil {
		return nil, err
	}
	if err := memory.ValidateContent(req.Input.Content); err != nil {
		return nil, err
	}
	if req.TopK < 0 {
		return nil, memory.ErrInvalidContent
	}

	topK := req.TopK
	if topK == 0 {
		topK = defaultRememberLintTopK
	}

	candidates, err := s.FindCandidates(memory.FindCandidatesRequest{
		SpaceID: req.Input.SpaceID,
		Content: req.Input.Content,
		TopK:    topK,
	})
	if err != nil {
		return nil, err
	}

	out := &memory.RememberLintResult{
		Candidates: candidates,
	}

	duplicateWarning := detectPossibleDuplicate(req.Input, candidates)
	if duplicateWarning != nil {
		out.Warnings = append(out.Warnings, *duplicateWarning)
	}
	updateWarning := detectPossibleUpdateTarget(req.Input, candidates)
	if updateWarning != nil {
		out.Warnings = append(out.Warnings, *updateWarning)
	}
	blobWarning := detectMixedFactBlob(req.Input)
	if blobWarning != nil {
		out.Warnings = append(out.Warnings, *blobWarning)
	}

	if suggestion := suggestSkipDuplicate(duplicateWarning); suggestion != nil {
		out.Suggestions = append(out.Suggestions, *suggestion)
	}
	if suggestion := suggestUpdateExisting(updateWarning); suggestion != nil {
		out.Suggestions = append(out.Suggestions, *suggestion)
	}
	if suggestion := suggestSplitCandidate(req.Input, blobWarning); suggestion != nil {
		out.Suggestions = append(out.Suggestions, *suggestion)
	}

	return out, nil
}

func detectPossibleDuplicate(input memory.MemoryInput, candidates []memory.ScoredMemory) *memory.RememberLintWarning {
	if input.Kind != memory.KindStatic || len(candidates) == 0 {
		return nil
	}
	top := candidates[0]
	if top.Kind != memory.KindStatic || top.Score < duplicateScoreThreshold {
		return nil
	}
	normalizedInput := normalizeLintText(input.Content)
	normalizedTop := normalizeLintText(top.Content)
	if normalizedInput != normalizedTop && top.Score < 0.995 {
		return nil
	}
	return &memory.RememberLintWarning{
		Code:         memory.RememberLintPossibleDuplicate,
		Message:      fmt.Sprintf("candidate is very close to existing live static memory %s", top.ID),
		CandidateIDs: []string{top.ID},
	}
}

func detectPossibleUpdateTarget(input memory.MemoryInput, candidates []memory.ScoredMemory) *memory.RememberLintWarning {
	if input.Kind != memory.KindStatic || len(candidates) == 0 || len(input.Relations.Updates) > 0 {
		return nil
	}
	top := candidates[0]
	if top.Kind != memory.KindStatic || top.Score < updateSuggestionScoreThreshold {
		return nil
	}
	if top.Score >= duplicateScoreThreshold {
		return nil
	}
	return &memory.RememberLintWarning{
		Code:         memory.RememberLintPossibleUpdateTarget,
		Message:      fmt.Sprintf("candidate may be updating existing live static memory %s instead of creating a new root", top.ID),
		CandidateIDs: []string{top.ID},
	}
}

func detectMixedFactBlob(input memory.MemoryInput) *memory.RememberLintWarning {
	if utf8.RuneCountInString(strings.TrimSpace(input.Content)) < mixedFactBlobRuneThreshold {
		return nil
	}
	clauseCount := strings.Count(input.Content, ". ") +
		strings.Count(input.Content, ";") +
		strings.Count(input.Content, "\n") +
		strings.Count(strings.ToLower(input.Content), "phase ")
	if clauseCount < 3 {
		return nil
	}
	return &memory.RememberLintWarning{
		Code:    memory.RememberLintMixedFactBlob,
		Message: "candidate appears to mix multiple facts or plan fragments into one memory; consider splitting it into sharper memories",
	}
}

func suggestSkipDuplicate(warning *memory.RememberLintWarning) *memory.RememberLintSuggestion {
	if warning == nil || warning.Code != memory.RememberLintPossibleDuplicate {
		return nil
	}
	return &memory.RememberLintSuggestion{
		Code:         memory.RememberLintSuggestSkipDuplicate,
		Message:      "consider reusing the existing memory instead of creating another root with the same claim",
		CandidateIDs: append([]string(nil), warning.CandidateIDs...),
	}
}

func suggestUpdateExisting(warning *memory.RememberLintWarning) *memory.RememberLintSuggestion {
	if warning == nil || warning.Code != memory.RememberLintPossibleUpdateTarget {
		return nil
	}
	return &memory.RememberLintSuggestion{
		Code:         memory.RememberLintSuggestUpdateExisting,
		Message:      "consider writing this as an explicit update to the existing lineage instead of a new root",
		CandidateIDs: append([]string(nil), warning.CandidateIDs...),
	}
}

func suggestSplitCandidate(input memory.MemoryInput, warning *memory.RememberLintWarning) *memory.RememberLintSuggestion {
	if warning == nil || warning.Code != memory.RememberLintMixedFactBlob {
		return nil
	}
	proposed := splitLintCandidate(input.Content)
	if len(proposed) < 2 {
		return nil
	}
	return &memory.RememberLintSuggestion{
		Code:             memory.RememberLintSuggestSplitCandidate,
		Message:          "candidate looks decomposable into sharper memories; consider writing separate memories for these atomic claims",
		ProposedContents: proposed,
	}
}

func splitLintCandidate(content string) []string {
	parts := strings.FieldsFunc(content, func(r rune) bool {
		return r == '\n' || r == ';'
	})
	if len(parts) <= 1 {
		parts = strings.Split(content, ". ")
	}
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		trimmed = strings.Trim(trimmed, ".")
		if utf8.RuneCountInString(trimmed) < 24 {
			continue
		}
		normalized := normalizeLintText(trimmed)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, trimmed)
		if len(out) == 4 {
			break
		}
	}
	return out
}

func normalizeLintText(v string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(v))), " ")
}
