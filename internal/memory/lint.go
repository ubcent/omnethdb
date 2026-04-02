package memory

type RememberLintRequest struct {
	Input MemoryInput
	TopK  int
}

type RememberLintWarningCode string

const (
	RememberLintPossibleDuplicate    RememberLintWarningCode = "possible_duplicate"
	RememberLintPossibleUpdateTarget RememberLintWarningCode = "possible_update_target"
	RememberLintMixedFactBlob        RememberLintWarningCode = "mixed_fact_blob"
)

type RememberLintSuggestionCode string

const (
	RememberLintSuggestSkipDuplicate  RememberLintSuggestionCode = "suggest_skip_duplicate"
	RememberLintSuggestUpdateExisting RememberLintSuggestionCode = "suggest_update_existing"
	RememberLintSuggestSplitCandidate RememberLintSuggestionCode = "suggest_split_candidate"
)

type RememberLintWarning struct {
	Code         RememberLintWarningCode `json:"code"`
	Message      string                  `json:"message"`
	CandidateIDs []string                `json:"candidate_ids,omitempty"`
}

type RememberLintSuggestion struct {
	Code             RememberLintSuggestionCode `json:"code"`
	Message          string                     `json:"message"`
	CandidateIDs     []string                   `json:"candidate_ids,omitempty"`
	ProposedContents []string                   `json:"proposed_contents,omitempty"`
}

type RememberLintResult struct {
	Candidates  []ScoredMemory           `json:"candidates"`
	Warnings    []RememberLintWarning    `json:"warnings"`
	Suggestions []RememberLintSuggestion `json:"suggestions,omitempty"`
}
