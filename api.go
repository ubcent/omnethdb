package omnethdb

import (
	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	"omnethdb/internal/runtime"
	storebolt "omnethdb/internal/store/bolt"
)

type (
	Store                      = storebolt.Store
	SpaceInit                  = storebolt.SpaceInit
	SpaceConfig                = memory.SpaceConfig
	WorkspaceLayout            = runtime.Layout
	RuntimeConfig              = runtime.Config
	RuntimeSpaceSettings       = runtime.SpaceSettings
	RuntimeEmbedderConfig      = runtime.RuntimeEmbedderConfig
	SpaceConfigChange          = runtime.SpaceConfigChange
	SpaceConfigReconcile       = runtime.SpaceConfigReconcile
	AuditEntry                 = memory.AuditEntry
	ForgetRecord               = memory.ForgetRecord
	ExportEdge                 = memory.ExportEdge
	ExportDiff                 = memory.ExportDiff
	ExportDiffRequest          = memory.ExportDiffRequest
	ExportFormat               = memory.ExportFormat
	ExportLineage              = memory.ExportLineage
	ExportLineageDiff          = memory.ExportLineageDiff
	ExportRequest              = memory.ExportRequest
	ExportSnapshot             = memory.ExportSnapshot
	RememberLintRequest        = memory.RememberLintRequest
	RememberLintResult         = memory.RememberLintResult
	RememberLintWarning        = memory.RememberLintWarning
	RememberLintSuggestion     = memory.RememberLintSuggestion
	Memory                     = memory.Memory
	MemoryProfile              = memory.MemoryProfile
	FindCandidatesRequest      = memory.FindCandidatesRequest
	QualityDiagnosticsRequest  = memory.QualityDiagnosticsRequest
	QualityDiagnosticsResult   = memory.QualityDiagnosticsResult
	DuplicateDiagnosticGroup   = memory.DuplicateDiagnosticGroup
	MemorySimilarityPair       = memory.MemorySimilarityPair
	QualityCleanupPlanRequest  = memory.QualityCleanupPlanRequest
	QualityCleanupPlanResult   = memory.QualityCleanupPlanResult
	DuplicateCleanupSuggestion = memory.DuplicateCleanupSuggestion
	ListMemoriesRequest        = memory.ListMemoriesRequest
	MemoryInput                = memory.MemoryInput
	ProfileRequest             = memory.ProfileRequest
	RecallRequest              = memory.RecallRequest
	ReviveInput                = memory.ReviveInput
	MemoryRelations            = memory.MemoryRelations
	RelationType               = memory.RelationType
	ScoredMemory               = memory.ScoredMemory
	MemoryKind                 = memory.MemoryKind
	Actor                      = memory.Actor
	ActorKind                  = memory.ActorKind
	WritersPolicy              = memory.WritersPolicy
	SpaceWritePolicy           = memory.SpaceWritePolicy
	Embedder                   = memory.Embedder
)

const (
	ExportFormatSnapshotJSON ExportFormat = memory.ExportFormatSnapshotJSON
	ExportFormatSummaryMD    ExportFormat = memory.ExportFormatSummaryMD
	ExportFormatGraphMermaid ExportFormat = memory.ExportFormatGraphMermaid
)

const (
	KindEpisodic MemoryKind = memory.KindEpisodic
	KindStatic   MemoryKind = memory.KindStatic
	KindDerived  MemoryKind = memory.KindDerived
	KindUnknown  MemoryKind = memory.KindUnknown
)

const (
	ActorHuman  ActorKind = memory.ActorHuman
	ActorAgent  ActorKind = memory.ActorAgent
	ActorSystem ActorKind = memory.ActorSystem
)

const (
	RelationUpdates RelationType = memory.RelationUpdates
	RelationExtends RelationType = memory.RelationExtends
	RelationDerives RelationType = memory.RelationDerives
)

const (
	RememberLintPossibleDuplicate     = memory.RememberLintPossibleDuplicate
	RememberLintPossibleUpdateTarget  = memory.RememberLintPossibleUpdateTarget
	RememberLintMixedFactBlob         = memory.RememberLintMixedFactBlob
	RememberLintSuggestSkipDuplicate  = memory.RememberLintSuggestSkipDuplicate
	RememberLintSuggestUpdateExisting = memory.RememberLintSuggestUpdateExisting
	RememberLintSuggestSplitCandidate = memory.RememberLintSuggestSplitCandidate
)

var (
	Open                      = storebolt.Open
	DefaultSpaceWritePolicy   = policy.DefaultSpaceWritePolicy
	NormalizeSpaceWritePolicy = policy.NormalizeSpaceWritePolicy
	ResolveActorTrust         = policy.ResolveActorTrust
	CanWriteKind              = policy.CanWriteKind
	CanPromote                = policy.CanPromote
	CompareExportSnapshots    = memory.CompareExportSnapshots
	ResolveWorkspaceLayout    = runtime.ResolveLayout
	OpenWorkspace             = runtime.OpenWorkspace
	LoadRuntimeConfig         = runtime.LoadConfig
	ValidateSpaceID           = memory.ValidateSpaceID
	ValidateMemoryID          = memory.ValidateMemoryID
	ValidateContent           = memory.ValidateContent
	ValidateMemoryKind        = memory.ValidateMemoryKind
	ValidateActorKind         = memory.ValidateActorKind
	ValidateActor             = memory.ValidateActor
	ValidateConfidence        = memory.ValidateConfidence
	ValidateWritersPolicy     = memory.ValidateWritersPolicy
	ValidateSpaceWritePolicy  = memory.ValidateSpaceWritePolicy
	ValidateSpaceConfig       = memory.ValidateSpaceConfig
	ValidateSpaceInit         = storebolt.ValidateSpaceInit

	ErrInvalidSpaceID           = memory.ErrInvalidSpaceID
	ErrInvalidMemoryID          = memory.ErrInvalidMemoryID
	ErrInvalidContent           = memory.ErrInvalidContent
	ErrInvalidActorID           = memory.ErrInvalidActorID
	ErrInvalidActorKind         = memory.ErrInvalidActorKind
	ErrInvalidMemoryKind        = memory.ErrInvalidMemoryKind
	ErrInvalidConfidence        = memory.ErrInvalidConfidence
	ErrInvalidTrustLevel        = memory.ErrInvalidTrustLevel
	ErrInvalidDimension         = memory.ErrInvalidDimension
	ErrInvalidDefaultWeight     = memory.ErrInvalidDefaultWeight
	ErrInvalidHalfLife          = memory.ErrInvalidHalfLife
	ErrInvalidMemoryVersion     = memory.ErrInvalidMemoryVersion
	ErrInvalidWritersPolicy     = memory.ErrInvalidWritersPolicy
	ErrInvalidSpaceConfig       = memory.ErrInvalidSpaceConfig
	ErrInvalidSpaceWritePolicy  = memory.ErrInvalidSpaceWritePolicy
	ErrInvalidSpaceInit         = memory.ErrInvalidSpaceInit
	ErrNilEmbedder              = memory.ErrNilEmbedder
	ErrEmbedderUnavailable      = memory.ErrEmbedderUnavailable
	ErrEmbeddingModelMismatch   = memory.ErrEmbeddingModelMismatch
	ErrSpaceMigrating           = memory.ErrSpaceMigrating
	ErrStoreClosed              = memory.ErrStoreClosed
	ErrConflict                 = memory.ErrConflict
	ErrMemoryNotFound           = memory.ErrMemoryNotFound
	ErrSpaceNotFound            = memory.ErrSpaceNotFound
	ErrLineageActive            = memory.ErrLineageActive
	ErrReviveDerivedUnsupported = memory.ErrReviveDerivedUnsupported
	ErrInvalidRelations         = memory.ErrInvalidRelations
	ErrPolicyViolation          = memory.ErrPolicyViolation
	ErrCorpusLimit              = memory.ErrCorpusLimit
	ErrUpdateTargetNotFound     = memory.ErrUpdateTargetNotFound
	ErrUpdateTargetNotLatest    = memory.ErrUpdateTargetNotLatest
	ErrUpdateTargetForgotten    = memory.ErrUpdateTargetForgotten
	ErrUpdateAcrossSpaces       = memory.ErrUpdateAcrossSpaces
	ErrUpdateAcrossKinds        = memory.ErrUpdateAcrossKinds
	ErrExtendsTargetNotFound    = memory.ErrExtendsTargetNotFound
	ErrExtendsTargetNotLatest   = memory.ErrExtendsTargetNotLatest
	ErrExtendsTargetForgotten   = memory.ErrExtendsTargetForgotten
	ErrExtendsAcrossSpaces      = memory.ErrExtendsAcrossSpaces
	ErrDerivedSourceCount       = memory.ErrDerivedSourceCount
	ErrDerivedSourceNotFound    = memory.ErrDerivedSourceNotFound
	ErrDerivedSourceNotLatest   = memory.ErrDerivedSourceNotLatest
	ErrDerivedSourceForgotten   = memory.ErrDerivedSourceForgotten
	ErrDerivedAcrossSpaces      = memory.ErrDerivedAcrossSpaces
	ErrDerivedSourceKind        = memory.ErrDerivedSourceKind
	ErrDerivedRationale         = memory.ErrDerivedRationale
	ErrDerivedActorKind         = memory.ErrDerivedActorKind
)
