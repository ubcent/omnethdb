package memory

import "time"

type Memory struct {
	ID        string
	SpaceID   string
	Content   string
	Embedding []float32

	Kind MemoryKind

	Actor      Actor
	Confidence float32

	Version  int
	IsLatest bool
	ParentID *string
	RootID   *string

	SourceIDs []string
	Rationale string

	IsForgotten        bool
	ForgetAfter        *time.Time
	HasOrphanedSources bool

	Relations MemoryRelations
	Metadata  map[string]any
	CreatedAt time.Time
}

type MemoryKind uint8

const (
	KindEpisodic MemoryKind = iota
	KindStatic
	KindDerived
	KindUnknown MemoryKind = 255
)

type Actor struct {
	ID   string
	Kind ActorKind
}

type ActorKind uint8

const (
	ActorHuman ActorKind = iota
	ActorAgent
	ActorSystem
)

type MemoryRelations struct {
	Updates []string
	Extends []string
	Derives []string
}

type RelationType string

const (
	RelationUpdates RelationType = "updates"
	RelationExtends RelationType = "extends"
	RelationDerives RelationType = "derives"
)

type SpaceWritePolicy struct {
	HumanTrust        float32
	SystemTrust       float32
	DefaultAgentTrust float32

	TrustLevels map[string]float32

	EpisodicWriters WritersPolicy
	StaticWriters   WritersPolicy
	DerivedWriters  WritersPolicy
	PromotePolicy   WritersPolicy

	MaxStaticMemories   int
	MaxEpisodicMemories int

	ProfileMaxStatic   int
	ProfileMaxEpisodic int
}

type WritersPolicy struct {
	AllowHuman      bool
	AllowSystem     bool
	AllowAllAgents  bool
	AllowedAgentIDs []string
	MinTrustLevel   float32
}

type SpaceConfig struct {
	EmbeddingModelID string
	Dimension        int
	DefaultWeight    float32
	HalfLifeDays     float32
	WritePolicy      SpaceWritePolicy

	Migrating                bool
	MigrationTargetModelID   string
	MigrationTargetDimension int
	MigrationStartedAt       *time.Time
}

type AuditEntry struct {
	Timestamp time.Time
	SpaceID   string
	Operation string
	Actor     Actor
	MemoryIDs []string
	Reason    string
}

type ForgetRecord struct {
	MemoryID  string
	SpaceID   string
	Actor     Actor
	Reason    string
	Timestamp time.Time
}
