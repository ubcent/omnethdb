package memory

import "time"

type MemoryInput struct {
	SpaceID     string
	Content     string
	Kind        MemoryKind
	Actor       Actor
	Confidence  float32
	ForgetAfter *time.Time
	Metadata    map[string]any

	SourceIDs  []string
	Rationale  string
	IfLatestID *string

	Relations MemoryRelations
}

type ReviveInput struct {
	Content    string
	Kind       MemoryKind
	Actor      Actor
	Confidence float32
	Metadata   map[string]any
}

type RecallRequest struct {
	SpaceIDs     []string
	SpaceWeights map[string]float32
	Query        string
	TopK         int
	Kinds        []MemoryKind

	ExcludeOrphanedDerives bool
}

type ListMemoriesRequest struct {
	SpaceIDs []string
	Kinds    []MemoryKind

	ExcludeOrphanedDerives bool
}

type ScoredMemory struct {
	Memory
	Score float32
}

type ProfileRequest struct {
	SpaceIDs     []string
	SpaceWeights map[string]float32
	Query        string
	StaticTopK   int
	EpisodicTopK int

	ExcludeOrphanedDerives bool
}

type MemoryProfile struct {
	Static   []ScoredMemory
	Episodic []ScoredMemory
}

type FindCandidatesRequest struct {
	SpaceID string
	Content string
	TopK    int

	IncludeSuperseded bool
	IncludeForgotten  bool
}

type QualityDiagnosticsRequest struct {
	SpaceID            string
	TopKPerMemory      int
	MaxDuplicateGroups int
	MaxUpdatePairs     int
}

type QualityDiagnosticsResult struct {
	SpaceID         string                     `json:"space_id"`
	GeneratedAt     time.Time                  `json:"generated_at"`
	LiveStaticCount int                        `json:"live_static_count"`
	DuplicateGroups []DuplicateDiagnosticGroup `json:"duplicate_groups"`
	PossibleUpdates []MemorySimilarityPair     `json:"possible_updates"`
}

type DuplicateDiagnosticGroup struct {
	MemoryIDs []string               `json:"memory_ids"`
	Pairs     []MemorySimilarityPair `json:"pairs"`
}

type MemorySimilarityPair struct {
	LeftID       string  `json:"left_id"`
	RightID      string  `json:"right_id"`
	LeftContent  string  `json:"left_content"`
	RightContent string  `json:"right_content"`
	Score        float32 `json:"score"`
}

type QualityCleanupPlanRequest struct {
	SpaceID             string
	MaxDuplicateActions int
}

type QualityCleanupPlanResult struct {
	SpaceID              string                       `json:"space_id"`
	GeneratedAt          time.Time                    `json:"generated_at"`
	DuplicateSuggestions []DuplicateCleanupSuggestion `json:"duplicate_suggestions"`
}

type DuplicateCleanupSuggestion struct {
	KeepID         string                   `json:"keep_id"`
	ForgetIDs      []string                 `json:"forget_ids"`
	Rationale      string                   `json:"rationale"`
	DuplicateGroup DuplicateDiagnosticGroup `json:"duplicate_group"`
}
