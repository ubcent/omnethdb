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
