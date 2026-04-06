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

type SynthesisCandidatesRequest struct {
	SpaceID string

	TopKPerMemory  int
	MaxCandidates  int
	MinClusterSize int
	MinPairScore   float32

	IncludeSuperseded bool
	IncludeForgotten  bool
}

type PromotionSuggestionsRequest struct {
	SpaceID string

	TopKPerMemory       int
	MaxSuggestions      int
	MinObservationCount int
	MinDistinctActors   int
	MinDistinctWindows  int
	MinCumulativeScore  float32
	MinPairScore        float32

	IncludeSuperseded bool
	IncludeForgotten  bool
}

type QualityDiagnosticsResult struct {
	SpaceID         string                     `json:"space_id"`
	GeneratedAt     time.Time                  `json:"generated_at"`
	LiveStaticCount int                        `json:"live_static_count"`
	DuplicateGroups []DuplicateDiagnosticGroup `json:"duplicate_groups"`
	PossibleUpdates []MemorySimilarityPair     `json:"possible_updates"`
}

type SynthesisCandidatesResult struct {
	SpaceID           string               `json:"space_id"`
	GeneratedAt       time.Time            `json:"generated_at"`
	LiveEpisodicCount int                  `json:"live_episodic_count"`
	Candidates        []SynthesisCandidate `json:"candidates"`
}

type PromotionSuggestionsResult struct {
	SpaceID           string                `json:"space_id"`
	GeneratedAt       time.Time             `json:"generated_at"`
	LiveEpisodicCount int                   `json:"live_episodic_count"`
	Suggestions       []PromotionSuggestion `json:"suggestions"`
}

type SynthesisCandidate struct {
	CandidateID     string                     `json:"candidate_id"`
	AdvisoryOnly    bool                       `json:"advisory_only"`
	ReasonCodes     []string                   `json:"reason_codes,omitempty"`
	SuggestedAction string                     `json:"suggested_action"`
	ReviewScore     float32                    `json:"review_score"`
	ClusterSize     int                        `json:"cluster_size"`
	DistinctActors  int                        `json:"distinct_actors"`
	DistinctSpaces  int                        `json:"distinct_spaces"`
	TimeSpanStart   time.Time                  `json:"time_span_start"`
	TimeSpanEnd     time.Time                  `json:"time_span_end"`
	MeanSimilarity  float32                    `json:"mean_similarity"`
	MinSimilarity   float32                    `json:"min_similarity"`
	MaxSimilarity   float32                    `json:"max_similarity"`
	Members         []SynthesisCandidateMember `json:"members"`
}

type SynthesisCandidateMember struct {
	MemoryID    string    `json:"memory_id"`
	SpaceID     string    `json:"space_id"`
	ActorID     string    `json:"actor_id"`
	CreatedAt   time.Time `json:"created_at"`
	Content     string    `json:"content"`
	IsLatest    bool      `json:"is_latest"`
	IsForgotten bool      `json:"is_forgotten"`
}

type PromotionSuggestion struct {
	MemoryID            string                    `json:"memory_id"`
	AdvisoryOnly        bool                      `json:"advisory_only"`
	ReasonCodes         []string                  `json:"reason_codes,omitempty"`
	SuggestedAction     string                    `json:"suggested_action"`
	ObservationCount    int                       `json:"observation_count"`
	DistinctTimeWindows int                       `json:"distinct_time_windows"`
	DistinctActors      int                       `json:"distinct_actors"`
	CumulativeScore     float32                   `json:"cumulative_score"`
	LatestScore         float32                   `json:"latest_score"`
	FirstSeenAt         time.Time                 `json:"first_seen_at"`
	LastSeenAt          time.Time                 `json:"last_seen_at"`
	ChurnRisk           float32                   `json:"churn_risk"`
	ContradictionRisk   float32                   `json:"contradiction_risk"`
	Explanation         string                    `json:"explanation"`
	Memory              PromotionSuggestionMemory `json:"memory"`
}

type PromotionSuggestionMemory struct {
	MemoryID    string    `json:"memory_id"`
	SpaceID     string    `json:"space_id"`
	ActorID     string    `json:"actor_id"`
	CreatedAt   time.Time `json:"created_at"`
	Content     string    `json:"content"`
	IsLatest    bool      `json:"is_latest"`
	IsForgotten bool      `json:"is_forgotten"`
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
	SpaceID                     string                       `json:"space_id"`
	GeneratedAt                 time.Time                    `json:"generated_at"`
	DuplicateSuggestions        []DuplicateCleanupSuggestion `json:"duplicate_suggestions"`
	SuggestedForgetBatchCommand string                       `json:"suggested_forget_batch_command,omitempty"`
}

type DuplicateCleanupSuggestion struct {
	KeepID         string                   `json:"keep_id"`
	ForgetIDs      []string                 `json:"forget_ids"`
	Rationale      string                   `json:"rationale"`
	DuplicateGroup DuplicateDiagnosticGroup `json:"duplicate_group"`
}
