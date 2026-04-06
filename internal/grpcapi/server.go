package grpcapi

import (
	"context"
	"errors"
	"strings"
	"time"

	omnethdb "omnethdb"
	hashembedder "omnethdb/embedders/hash"
	omnethdbv1 "omnethdb/gen/omnethdb/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultModelID   = "builtin/hash-embedder-v1"
	defaultDimension = 256
)

type Server struct {
	omnethdbv1.UnimplementedOmnethDBServer
	store *omnethdb.Store
	cfg   *omnethdb.RuntimeConfig
}

func NewServer(store *omnethdb.Store, cfg *omnethdb.RuntimeConfig) *Server {
	return &Server{store: store, cfg: cfg}
}

func (s *Server) Health(context.Context, *omnethdbv1.HealthRequest) (*omnethdbv1.HealthResponse, error) {
	return &omnethdbv1.HealthResponse{Status: "ok"}, nil
}

func (s *Server) GetRuntimeConfig(context.Context, *omnethdbv1.GetRuntimeConfigRequest) (*omnethdbv1.GetRuntimeConfigResponse, error) {
	return &omnethdbv1.GetRuntimeConfigResponse{Config: runtimeConfigToProto(s.cfg)}, nil
}

func (s *Server) InitSpace(_ context.Context, req *omnethdbv1.InitSpaceRequest) (*omnethdbv1.SpaceConfig, error) {
	embedder := s.embedderForBootstrap(req.GetSpaceId())
	init := s.configOrEmpty().SpaceInit(req.GetSpaceId(), omnethdb.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	})
	cfg, err := s.store.EnsureSpace(req.GetSpaceId(), embedder, init)
	if err != nil {
		return nil, mapError(err)
	}
	return spaceConfigToProto(cfg), nil
}

func (s *Server) GetSpaceConfig(_ context.Context, req *omnethdbv1.GetSpaceConfigRequest) (*omnethdbv1.SpaceConfig, error) {
	cfg, err := s.store.GetSpaceConfig(req.GetSpaceId())
	if err != nil {
		return nil, mapError(err)
	}
	return spaceConfigToProto(cfg), nil
}

func (s *Server) Remember(_ context.Context, req *omnethdbv1.RememberRequest) (*omnethdbv1.Memory, error) {
	s.ensureEmbedderForSpace(req.GetSpaceId())
	input, err := rememberRequestToDomain(req)
	if err != nil {
		return nil, mapError(err)
	}
	mem, err := s.store.Remember(input)
	if err != nil {
		return nil, mapError(err)
	}
	return memoryToProto(mem), nil
}

func (s *Server) Recall(_ context.Context, req *omnethdbv1.RecallRequest) (*omnethdbv1.RecallResponse, error) {
	s.ensureEmbeddersForSpaces(req.GetSpaceIds())
	results, err := s.store.Recall(omnethdb.RecallRequest{
		SpaceIDs:               append([]string(nil), req.GetSpaceIds()...),
		SpaceWeights:           req.GetSpaceWeights(),
		Query:                  req.GetQuery(),
		TopK:                   int(req.GetTopK()),
		Kinds:                  memoryKindsFromProto(req.GetKinds()),
		ExcludeOrphanedDerives: req.GetExcludeOrphanedDerives(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &omnethdbv1.RecallResponse{Memories: scoredMemoriesToProto(results)}, nil
}

func (s *Server) GetProfile(_ context.Context, req *omnethdbv1.GetProfileRequest) (*omnethdbv1.MemoryProfile, error) {
	s.ensureEmbeddersForSpaces(req.GetSpaceIds())
	profile, err := s.store.GetProfile(omnethdb.ProfileRequest{
		SpaceIDs:               append([]string(nil), req.GetSpaceIds()...),
		SpaceWeights:           req.GetSpaceWeights(),
		Query:                  req.GetQuery(),
		StaticTopK:             int(req.GetStaticTopK()),
		EpisodicTopK:           int(req.GetEpisodicTopK()),
		ExcludeOrphanedDerives: req.GetExcludeOrphanedDerives(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &omnethdbv1.MemoryProfile{
		Static:   scoredMemoriesToProto(profile.Static),
		Episodic: scoredMemoriesToProto(profile.Episodic),
	}, nil
}

func (s *Server) FindCandidates(_ context.Context, req *omnethdbv1.FindCandidatesRequest) (*omnethdbv1.RecallResponse, error) {
	s.ensureEmbedderForSpace(req.GetSpaceId())
	results, err := s.store.FindCandidates(omnethdb.FindCandidatesRequest{
		SpaceID:           req.GetSpaceId(),
		Content:           req.GetContent(),
		TopK:              int(req.GetTopK()),
		IncludeSuperseded: req.GetIncludeSuperseded(),
		IncludeForgotten:  req.GetIncludeForgotten(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &omnethdbv1.RecallResponse{Memories: scoredMemoriesToProto(results)}, nil
}

func (s *Server) SynthesisCandidates(_ context.Context, req *omnethdbv1.SynthesisCandidatesRequest) (*omnethdbv1.SynthesisCandidatesResponse, error) {
	s.ensureEmbedderForSpace(req.GetSpaceId())
	result, err := s.store.GetSynthesisCandidates(omnethdb.SynthesisCandidatesRequest{
		SpaceID:           req.GetSpaceId(),
		TopKPerMemory:     int(req.GetTopKPerMemory()),
		MaxCandidates:     int(req.GetMaxCandidates()),
		MinClusterSize:    int(req.GetMinClusterSize()),
		MinPairScore:      req.GetMinPairScore(),
		IncludeSuperseded: req.GetIncludeSuperseded(),
		IncludeForgotten:  req.GetIncludeForgotten(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return synthesisCandidatesResultToProto(result), nil
}

func (s *Server) PromotionSuggestions(_ context.Context, req *omnethdbv1.PromotionSuggestionsRequest) (*omnethdbv1.PromotionSuggestionsResponse, error) {
	s.ensureEmbedderForSpace(req.GetSpaceId())
	result, err := s.store.GetPromotionSuggestions(omnethdb.PromotionSuggestionsRequest{
		SpaceID:             req.GetSpaceId(),
		TopKPerMemory:       int(req.GetTopKPerMemory()),
		MaxSuggestions:      int(req.GetMaxSuggestions()),
		MinObservationCount: int(req.GetMinObservationCount()),
		MinDistinctActors:   int(req.GetMinDistinctActors()),
		MinDistinctWindows:  int(req.GetMinDistinctWindows()),
		MinCumulativeScore:  req.GetMinCumulativeScore(),
		MinPairScore:        req.GetMinPairScore(),
		IncludeSuperseded:   req.GetIncludeSuperseded(),
		IncludeForgotten:    req.GetIncludeForgotten(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return promotionSuggestionsResultToProto(result), nil
}

func (s *Server) Forget(_ context.Context, req *omnethdbv1.ForgetRequest) (*omnethdbv1.StatusResponse, error) {
	if err := s.store.Forget(req.GetId(), actorFromProto(req.GetActor()), req.GetReason()); err != nil {
		return nil, mapError(err)
	}
	return &omnethdbv1.StatusResponse{Status: "ok"}, nil
}

func (s *Server) Revive(_ context.Context, req *omnethdbv1.ReviveRequest) (*omnethdbv1.Memory, error) {
	mem, err := s.store.Revive(req.GetRootId(), omnethdb.ReviveInput{
		Content:    req.GetContent(),
		Kind:       memoryKindFromProto(req.GetKind()),
		Actor:      actorFromProto(req.GetActor()),
		Confidence: req.GetConfidence(),
		Metadata:   structToMap(req.GetMetadata()),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return memoryToProto(mem), nil
}

func (s *Server) GetLineage(_ context.Context, req *omnethdbv1.GetLineageRequest) (*omnethdbv1.LineageResponse, error) {
	memories, err := s.store.GetLineage(req.GetRootId())
	if err != nil {
		return nil, mapError(err)
	}
	return &omnethdbv1.LineageResponse{Memories: memoriesToProto(memories)}, nil
}

func (s *Server) GetRelated(_ context.Context, req *omnethdbv1.GetRelatedRequest) (*omnethdbv1.RelatedResponse, error) {
	memories, err := s.store.GetRelated(req.GetMemoryId(), relationTypeFromProto(req.GetRelation()), int(req.GetDepth()))
	if err != nil {
		return nil, mapError(err)
	}
	return &omnethdbv1.RelatedResponse{Memories: memoriesToProto(memories)}, nil
}

func (s *Server) GetAuditLog(_ context.Context, req *omnethdbv1.GetAuditLogRequest) (*omnethdbv1.AuditLogResponse, error) {
	var since time.Time
	if req.GetSince() != nil {
		since = req.GetSince().AsTime()
	}
	entries, err := s.store.GetAuditLog(req.GetSpaceId(), since)
	if err != nil {
		return nil, mapError(err)
	}
	return &omnethdbv1.AuditLogResponse{Entries: auditEntriesToProto(entries)}, nil
}

func (s *Server) MigrateEmbeddings(_ context.Context, req *omnethdbv1.MigrateEmbeddingsRequest) (*omnethdbv1.SpaceConfig, error) {
	embedder := s.embedderForMigration(req.GetSpaceId(), req.GetModelId(), int(req.GetDimensions()))
	if err := s.store.MigrateEmbeddings(req.GetSpaceId(), embedder); err != nil {
		return nil, mapError(err)
	}
	cfg, err := s.store.GetSpaceConfig(req.GetSpaceId())
	if err != nil {
		return nil, mapError(err)
	}
	return spaceConfigToProto(cfg), nil
}

func (s *Server) ensureEmbeddersForSpaces(spaceIDs []string) {
	for _, spaceID := range spaceIDs {
		s.ensureEmbedderForSpace(spaceID)
	}
}

func (s *Server) ensureEmbedderForSpace(spaceID string) {
	if strings.TrimSpace(spaceID) == "" || s.store == nil {
		return
	}
	if cfg := s.configOrEmpty(); cfg != nil {
		if settings, ok := cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				s.store.RegisterEmbedder(hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions))
				return
			}
		}
	}
	if persisted, err := s.store.GetSpaceConfig(spaceID); err == nil {
		s.store.RegisterEmbedder(hashembedder.New(persisted.EmbeddingModelID, persisted.Dimension))
	}
}

func (s *Server) embedderForBootstrap(spaceID string) omnethdb.Embedder {
	if cfg := s.configOrEmpty(); cfg != nil {
		if settings, ok := cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				return hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions)
			}
		}
	}
	return hashembedder.New(defaultModelID, defaultDimension)
}

func (s *Server) embedderForMigration(spaceID string, modelID string, dimensions int) omnethdb.Embedder {
	if strings.TrimSpace(modelID) != "" && dimensions > 0 {
		return hashembedder.New(modelID, dimensions)
	}
	if cfg := s.configOrEmpty(); cfg != nil {
		if settings, ok := cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				return hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions)
			}
		}
	}
	if persisted, err := s.store.GetSpaceConfig(spaceID); err == nil {
		return hashembedder.New(persisted.EmbeddingModelID, persisted.Dimension)
	}
	return hashembedder.New(defaultModelID, defaultDimension)
}

func (s *Server) configOrEmpty() *omnethdb.RuntimeConfig {
	if s.cfg != nil {
		return s.cfg
	}
	return &omnethdb.RuntimeConfig{}
}

func rememberRequestToDomain(req *omnethdbv1.RememberRequest) (omnethdb.MemoryInput, error) {
	input := omnethdb.MemoryInput{
		SpaceID:    req.GetSpaceId(),
		Content:    req.GetContent(),
		Kind:       memoryKindFromProto(req.GetKind()),
		Actor:      actorFromProto(req.GetActor()),
		Confidence: req.GetConfidence(),
		Metadata:   structToMap(req.GetMetadata()),
		SourceIDs:  append([]string(nil), req.GetSourceIds()...),
		Rationale:  req.GetRationale(),
		Relations: omnethdb.MemoryRelations{
			Updates: append([]string(nil), req.GetRelations().GetUpdates()...),
			Extends: append([]string(nil), req.GetRelations().GetExtends()...),
			Derives: append([]string(nil), req.GetRelations().GetDerives()...),
		},
	}
	if ts := req.GetForgetAfter(); ts != nil {
		t := ts.AsTime()
		input.ForgetAfter = &t
	}
	if id := strings.TrimSpace(req.GetIfLatestId()); id != "" {
		input.IfLatestID = &id
	}
	return input, nil
}

func memoryKindFromProto(kind omnethdbv1.MemoryKind) omnethdb.MemoryKind {
	switch kind {
	case omnethdbv1.MemoryKind_MEMORY_KIND_EPISODIC:
		return omnethdb.KindEpisodic
	case omnethdbv1.MemoryKind_MEMORY_KIND_STATIC:
		return omnethdb.KindStatic
	case omnethdbv1.MemoryKind_MEMORY_KIND_DERIVED:
		return omnethdb.KindDerived
	default:
		return omnethdb.KindUnknown
	}
}

func memoryKindsFromProto(kinds []omnethdbv1.MemoryKind) []omnethdb.MemoryKind {
	if len(kinds) == 0 {
		return nil
	}
	out := make([]omnethdb.MemoryKind, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, memoryKindFromProto(kind))
	}
	return out
}

func memoryKindToProto(kind omnethdb.MemoryKind) omnethdbv1.MemoryKind {
	switch kind {
	case omnethdb.KindEpisodic:
		return omnethdbv1.MemoryKind_MEMORY_KIND_EPISODIC
	case omnethdb.KindStatic:
		return omnethdbv1.MemoryKind_MEMORY_KIND_STATIC
	case omnethdb.KindDerived:
		return omnethdbv1.MemoryKind_MEMORY_KIND_DERIVED
	default:
		return omnethdbv1.MemoryKind_MEMORY_KIND_EPISODIC
	}
}

func actorFromProto(actor *omnethdbv1.Actor) omnethdb.Actor {
	if actor == nil {
		return omnethdb.Actor{}
	}
	return omnethdb.Actor{
		ID:   actor.GetId(),
		Kind: actorKindFromProto(actor.GetKind()),
	}
}

func actorKindFromProto(kind omnethdbv1.ActorKind) omnethdb.ActorKind {
	switch kind {
	case omnethdbv1.ActorKind_ACTOR_KIND_HUMAN:
		return omnethdb.ActorHuman
	case omnethdbv1.ActorKind_ACTOR_KIND_AGENT:
		return omnethdb.ActorAgent
	case omnethdbv1.ActorKind_ACTOR_KIND_SYSTEM:
		return omnethdb.ActorSystem
	default:
		return omnethdb.ActorSystem + 99
	}
}

func actorToProto(actor omnethdb.Actor) *omnethdbv1.Actor {
	return &omnethdbv1.Actor{
		Id:   actor.ID,
		Kind: actorKindToProto(actor.Kind),
	}
}

func actorKindToProto(kind omnethdb.ActorKind) omnethdbv1.ActorKind {
	switch kind {
	case omnethdb.ActorHuman:
		return omnethdbv1.ActorKind_ACTOR_KIND_HUMAN
	case omnethdb.ActorAgent:
		return omnethdbv1.ActorKind_ACTOR_KIND_AGENT
	case omnethdb.ActorSystem:
		return omnethdbv1.ActorKind_ACTOR_KIND_SYSTEM
	default:
		return omnethdbv1.ActorKind_ACTOR_KIND_SYSTEM
	}
}

func relationTypeFromProto(kind omnethdbv1.RelationType) omnethdb.RelationType {
	switch kind {
	case omnethdbv1.RelationType_RELATION_TYPE_UPDATES:
		return omnethdb.RelationUpdates
	case omnethdbv1.RelationType_RELATION_TYPE_EXTENDS:
		return omnethdb.RelationExtends
	case omnethdbv1.RelationType_RELATION_TYPE_DERIVES:
		return omnethdb.RelationDerives
	default:
		return omnethdb.RelationExtends
	}
}

func relationTypeToProto(kind omnethdb.RelationType) omnethdbv1.RelationType {
	switch kind {
	case omnethdb.RelationUpdates:
		return omnethdbv1.RelationType_RELATION_TYPE_UPDATES
	case omnethdb.RelationExtends:
		return omnethdbv1.RelationType_RELATION_TYPE_EXTENDS
	case omnethdb.RelationDerives:
		return omnethdbv1.RelationType_RELATION_TYPE_DERIVES
	default:
		return omnethdbv1.RelationType_RELATION_TYPE_EXTENDS
	}
}

func memoryToProto(mem *omnethdb.Memory) *omnethdbv1.Memory {
	if mem == nil {
		return nil
	}
	return &omnethdbv1.Memory{
		Id:                 mem.ID,
		SpaceId:            mem.SpaceID,
		Content:            mem.Content,
		Embedding:          append([]float32(nil), mem.Embedding...),
		Kind:               memoryKindToProto(mem.Kind),
		Actor:              actorToProto(mem.Actor),
		Confidence:         mem.Confidence,
		Version:            int32(mem.Version),
		IsLatest:           mem.IsLatest,
		ParentId:           derefString(mem.ParentID),
		RootId:             derefString(mem.RootID),
		SourceIds:          append([]string(nil), mem.SourceIDs...),
		Rationale:          mem.Rationale,
		IsForgotten:        mem.IsForgotten,
		ForgetAfter:        timestampPtr(mem.ForgetAfter),
		HasOrphanedSources: mem.HasOrphanedSources,
		Relations: &omnethdbv1.MemoryRelations{
			Updates: append([]string(nil), mem.Relations.Updates...),
			Extends: append([]string(nil), mem.Relations.Extends...),
			Derives: append([]string(nil), mem.Relations.Derives...),
		},
		Metadata:  structFromMap(mem.Metadata),
		CreatedAt: timestamppb.New(mem.CreatedAt),
	}
}

func memoriesToProto(memories []omnethdb.Memory) []*omnethdbv1.Memory {
	out := make([]*omnethdbv1.Memory, 0, len(memories))
	for i := range memories {
		mem := memories[i]
		out = append(out, memoryToProto(&mem))
	}
	return out
}

func scoredMemoriesToProto(memories []omnethdb.ScoredMemory) []*omnethdbv1.ScoredMemory {
	out := make([]*omnethdbv1.ScoredMemory, 0, len(memories))
	for i := range memories {
		mem := memories[i]
		out = append(out, &omnethdbv1.ScoredMemory{
			Memory: memoryToProto(&mem.Memory),
			Score:  mem.Score,
		})
	}
	return out
}

func synthesisCandidatesResultToProto(result *omnethdb.SynthesisCandidatesResult) *omnethdbv1.SynthesisCandidatesResponse {
	if result == nil {
		return nil
	}
	return &omnethdbv1.SynthesisCandidatesResponse{
		SpaceId:           result.SpaceID,
		GeneratedAt:       timestamppb.New(result.GeneratedAt),
		LiveEpisodicCount: int32(result.LiveEpisodicCount),
		Candidates:        synthesisCandidatesToProto(result.Candidates),
	}
}

func synthesisCandidatesToProto(items []omnethdb.SynthesisCandidate) []*omnethdbv1.SynthesisCandidate {
	out := make([]*omnethdbv1.SynthesisCandidate, 0, len(items))
	for _, item := range items {
		out = append(out, &omnethdbv1.SynthesisCandidate{
			CandidateId:     item.CandidateID,
			AdvisoryOnly:    item.AdvisoryOnly,
			ReasonCodes:     append([]string(nil), item.ReasonCodes...),
			SuggestedAction: item.SuggestedAction,
			ReviewScore:     item.ReviewScore,
			ClusterSize:     int32(item.ClusterSize),
			DistinctActors:  int32(item.DistinctActors),
			DistinctSpaces:  int32(item.DistinctSpaces),
			TimeSpanStart:   timestamppb.New(item.TimeSpanStart),
			TimeSpanEnd:     timestamppb.New(item.TimeSpanEnd),
			MeanSimilarity:  item.MeanSimilarity,
			MinSimilarity:   item.MinSimilarity,
			MaxSimilarity:   item.MaxSimilarity,
			Members:         synthesisCandidateMembersToProto(item.Members),
		})
	}
	return out
}

func synthesisCandidateMembersToProto(items []omnethdb.SynthesisCandidateMember) []*omnethdbv1.SynthesisCandidateMember {
	out := make([]*omnethdbv1.SynthesisCandidateMember, 0, len(items))
	for _, item := range items {
		out = append(out, &omnethdbv1.SynthesisCandidateMember{
			MemoryId:    item.MemoryID,
			SpaceId:     item.SpaceID,
			ActorId:     item.ActorID,
			CreatedAt:   timestamppb.New(item.CreatedAt),
			Content:     item.Content,
			IsLatest:    item.IsLatest,
			IsForgotten: item.IsForgotten,
		})
	}
	return out
}

func promotionSuggestionsResultToProto(result *omnethdb.PromotionSuggestionsResult) *omnethdbv1.PromotionSuggestionsResponse {
	if result == nil {
		return nil
	}
	return &omnethdbv1.PromotionSuggestionsResponse{
		SpaceId:           result.SpaceID,
		GeneratedAt:       timestamppb.New(result.GeneratedAt),
		LiveEpisodicCount: int32(result.LiveEpisodicCount),
		Suggestions:       promotionSuggestionsToProto(result.Suggestions),
	}
}

func promotionSuggestionsToProto(items []omnethdb.PromotionSuggestion) []*omnethdbv1.PromotionSuggestion {
	out := make([]*omnethdbv1.PromotionSuggestion, 0, len(items))
	for _, item := range items {
		out = append(out, &omnethdbv1.PromotionSuggestion{
			MemoryId:            item.MemoryID,
			AdvisoryOnly:        item.AdvisoryOnly,
			ReasonCodes:         append([]string(nil), item.ReasonCodes...),
			SuggestedAction:     item.SuggestedAction,
			ObservationCount:    int32(item.ObservationCount),
			DistinctTimeWindows: int32(item.DistinctTimeWindows),
			DistinctActors:      int32(item.DistinctActors),
			CumulativeScore:     item.CumulativeScore,
			LatestScore:         item.LatestScore,
			FirstSeenAt:         timestamppb.New(item.FirstSeenAt),
			LastSeenAt:          timestamppb.New(item.LastSeenAt),
			ChurnRisk:           item.ChurnRisk,
			ContradictionRisk:   item.ContradictionRisk,
			Explanation:         item.Explanation,
			Memory:              promotionSuggestionMemoryToProto(item.Memory),
		})
	}
	return out
}

func promotionSuggestionMemoryToProto(item omnethdb.PromotionSuggestionMemory) *omnethdbv1.PromotionSuggestionMemory {
	return &omnethdbv1.PromotionSuggestionMemory{
		MemoryId:    item.MemoryID,
		SpaceId:     item.SpaceID,
		ActorId:     item.ActorID,
		CreatedAt:   timestamppb.New(item.CreatedAt),
		Content:     item.Content,
		IsLatest:    item.IsLatest,
		IsForgotten: item.IsForgotten,
	}
}

func spaceConfigToProto(cfg *omnethdb.SpaceConfig) *omnethdbv1.SpaceConfig {
	if cfg == nil {
		return nil
	}
	return &omnethdbv1.SpaceConfig{
		EmbeddingModelId:         cfg.EmbeddingModelID,
		Dimension:                int32(cfg.Dimension),
		DefaultWeight:            cfg.DefaultWeight,
		HalfLifeDays:             cfg.HalfLifeDays,
		WritePolicy:              spaceWritePolicyToProto(cfg.WritePolicy),
		Migrating:                cfg.Migrating,
		MigrationTargetModelId:   cfg.MigrationTargetModelID,
		MigrationTargetDimension: int32(cfg.MigrationTargetDimension),
		MigrationStartedAt:       timestampPtr(cfg.MigrationStartedAt),
	}
}

func spaceWritePolicyToProto(policy omnethdb.SpaceWritePolicy) *omnethdbv1.SpaceWritePolicy {
	return &omnethdbv1.SpaceWritePolicy{
		HumanTrust:          policy.HumanTrust,
		SystemTrust:         policy.SystemTrust,
		DefaultAgentTrust:   policy.DefaultAgentTrust,
		TrustLevels:         cloneTrustLevels(policy.TrustLevels),
		EpisodicWriters:     writersPolicyToProto(policy.EpisodicWriters),
		StaticWriters:       writersPolicyToProto(policy.StaticWriters),
		DerivedWriters:      writersPolicyToProto(policy.DerivedWriters),
		PromotePolicy:       writersPolicyToProto(policy.PromotePolicy),
		MaxStaticMemories:   int32(policy.MaxStaticMemories),
		MaxEpisodicMemories: int32(policy.MaxEpisodicMemories),
		ProfileMaxStatic:    int32(policy.ProfileMaxStatic),
		ProfileMaxEpisodic:  int32(policy.ProfileMaxEpisodic),
	}
}

func writersPolicyToProto(policy omnethdb.WritersPolicy) *omnethdbv1.WritersPolicy {
	return &omnethdbv1.WritersPolicy{
		AllowHuman:      policy.AllowHuman,
		AllowSystem:     policy.AllowSystem,
		AllowAllAgents:  policy.AllowAllAgents,
		AllowedAgentIds: append([]string(nil), policy.AllowedAgentIDs...),
		MinTrustLevel:   policy.MinTrustLevel,
	}
}

func runtimeConfigToProto(cfg *omnethdb.RuntimeConfig) *omnethdbv1.RuntimeConfig {
	if cfg == nil {
		return &omnethdbv1.RuntimeConfig{}
	}
	out := &omnethdbv1.RuntimeConfig{Spaces: map[string]*omnethdbv1.RuntimeSpaceSettings{}}
	for spaceID, settings := range cfg.Spaces {
		out.Spaces[spaceID] = runtimeSpaceSettingsToProto(settings)
	}
	return out
}

func runtimeSpaceSettingsToProto(settings omnethdb.RuntimeSpaceSettings) *omnethdbv1.RuntimeSpaceSettings {
	out := &omnethdbv1.RuntimeSpaceSettings{
		Embedder: &omnethdbv1.RuntimeEmbedderConfig{
			ModelId:    settings.Embedder.ModelID,
			Dimensions: int32(settings.Embedder.Dimensions),
		},
	}
	if settings.DefaultWeight != nil {
		out.DefaultWeight = settings.DefaultWeight
	}
	if settings.HalfLifeDays != nil {
		out.HalfLifeDays = settings.HalfLifeDays
	}
	if settings.MaxStaticMemories != nil {
		v := int32(*settings.MaxStaticMemories)
		out.MaxStaticMemories = &v
	}
	if settings.MaxEpisodicMemories != nil {
		v := int32(*settings.MaxEpisodicMemories)
		out.MaxEpisodicMemories = &v
	}
	if settings.ProfileMaxStatic != nil {
		v := int32(*settings.ProfileMaxStatic)
		out.ProfileMaxStatic = &v
	}
	if settings.ProfileMaxEpisodic != nil {
		v := int32(*settings.ProfileMaxEpisodic)
		out.ProfileMaxEpisodic = &v
	}
	if settings.HumanTrust != nil {
		out.HumanTrust = settings.HumanTrust
	}
	if settings.SystemTrust != nil {
		out.SystemTrust = settings.SystemTrust
	}
	if settings.DefaultAgentTrust != nil {
		out.DefaultAgentTrust = settings.DefaultAgentTrust
	}
	return out
}

func auditEntriesToProto(entries []omnethdb.AuditEntry) []*omnethdbv1.AuditEntry {
	out := make([]*omnethdbv1.AuditEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, &omnethdbv1.AuditEntry{
			Timestamp: timestamppb.New(entry.Timestamp),
			SpaceId:   entry.SpaceID,
			Operation: entry.Operation,
			Actor:     actorToProto(entry.Actor),
			MemoryIds: append([]string(nil), entry.MemoryIDs...),
			Reason:    entry.Reason,
		})
	}
	return out
}

func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, omnethdb.ErrConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, omnethdb.ErrSpaceNotFound), errors.Is(err, omnethdb.ErrMemoryNotFound),
		errors.Is(err, omnethdb.ErrUpdateTargetNotFound), errors.Is(err, omnethdb.ErrExtendsTargetNotFound),
		errors.Is(err, omnethdb.ErrDerivedSourceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, omnethdb.ErrSpaceMigrating):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, omnethdb.ErrPolicyViolation), errors.Is(err, omnethdb.ErrCorpusLimit),
		errors.Is(err, omnethdb.ErrLineageActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.InvalidArgument, err.Error())
	}
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func timestampPtr(v *time.Time) *timestamppb.Timestamp {
	if v == nil {
		return nil
	}
	return timestamppb.New(*v)
}

func structFromMap(in map[string]any) *structpb.Struct {
	if in == nil {
		return nil
	}
	s, err := structpb.NewStruct(in)
	if err != nil {
		return nil
	}
	return s
}

func structToMap(in *structpb.Struct) map[string]any {
	if in == nil {
		return nil
	}
	return in.AsMap()
}

func cloneTrustLevels(in map[string]float32) map[string]float32 {
	if in == nil {
		return nil
	}
	out := make(map[string]float32, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
