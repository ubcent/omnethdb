package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	omnethdb "omnethdb"
	hashembedder "omnethdb/embedders/hash"
)

const (
	defaultModelID   = "builtin/hash-embedder-v1"
	defaultDimension = 256
)

type WorkspaceOpener func() (*omnethdb.Store, *omnethdb.RuntimeConfig, error)

func NewOmnethDBTools(open WorkspaceOpener) []Tool {
	return []Tool{
		spaceInitTool{open: open},
		memoryLintRememberTool{open: open},
		memoryRememberTool{open: open},
		memoryForgetTool{open: open},
		memoryRecallTool{open: open},
		memoryProfileTool{open: open},
		memoryProfileCompactTool{open: open},
		memorySynthesisCandidatesTool{open: open},
		memoryPromotionSuggestionsTool{open: open},
		memoryLineageTool{open: open},
		memoryRelatedTool{open: open},
		memoryExportSummaryTool{open: open},
	}
}

type spaceInitTool struct{ open WorkspaceOpener }
type memoryLintRememberTool struct{ open WorkspaceOpener }
type memoryRememberTool struct{ open WorkspaceOpener }
type memoryForgetTool struct{ open WorkspaceOpener }
type memoryRecallTool struct{ open WorkspaceOpener }
type memoryProfileTool struct{ open WorkspaceOpener }
type memoryProfileCompactTool struct{ open WorkspaceOpener }
type memorySynthesisCandidatesTool struct{ open WorkspaceOpener }
type memoryPromotionSuggestionsTool struct{ open WorkspaceOpener }
type memoryLineageTool struct{ open WorkspaceOpener }
type memoryRelatedTool struct{ open WorkspaceOpener }
type memoryExportSummaryTool struct{ open WorkspaceOpener }

type rememberAdvisory struct {
	Warnings    []omnethdb.RememberLintWarning    `json:"warnings,omitempty"`
	Suggestions []omnethdb.RememberLintSuggestion `json:"suggestions,omitempty"`
	Candidates  []omnethdb.ScoredMemory           `json:"candidates,omitempty"`
}

type rememberToolResponse struct {
	omnethdb.Memory
	Advisory *rememberAdvisory `json:"advisory,omitempty"`
}

type compactScoredMemory struct {
	ID             string         `json:"id"`
	SpaceID        string         `json:"space_id"`
	Kind           string         `json:"kind"`
	ContentPreview string         `json:"content_preview"`
	Score          float32        `json:"score"`
	Confidence     float32        `json:"confidence"`
	Actor          omnethdb.Actor `json:"actor"`
	Version        int            `json:"version"`
	Latest         bool           `json:"latest"`
	Forgotten      bool           `json:"forgotten"`
	Orphaned       bool           `json:"orphaned"`
}

type compactProfile struct {
	Static   []compactScoredMemory `json:"static"`
	Episodic []compactScoredMemory `json:"episodic"`
}

func (t spaceInitTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "space_init",
		Description: "Bootstrap a space so an agent can start writing and recalling governed memories.",
		InputSchema: objectSchema(
			requiredProp("space_id", "string", "Space ID like repo:company/app."),
		),
	}
}

func (t spaceInitTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceID string `json:"space_id"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, cfg, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()

	embedder := embedderForBootstrap(cfg, req.SpaceID)
	init := omnethdb.DefaultSpaceWritePolicy()
	spaceInit := omnethdb.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   init,
	}
	if cfg != nil {
		spaceInit = cfg.SpaceInit(req.SpaceID, spaceInit)
	}
	spaceCfg, err := store.EnsureSpace(req.SpaceID, embedder, spaceInit)
	if err != nil {
		return ToolResult{}, err
	}
	return jsonResult(spaceCfg)
}

func (t memoryLintRememberTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_lint_remember",
		Description: "Advisory lint for a candidate memory write. Returns possible duplicate, update-target, or blob warnings without writing anything.",
		InputSchema: objectSchema(
			requiredProp("space_id", "string", "Target space ID."),
			requiredProp("content", "string", "Candidate memory content."),
			requiredProp("kind", "string", "episodic, static, or derived."),
			optionalProp("actor_id", "string", "Writer actor ID."),
			optionalProp("actor_kind", "string", "human, agent, or system."),
			optionalProp("confidence", "number", "Confidence 0..1."),
			optionalProp("update_id", "string", "Optional memory ID this write intends to update."),
			arrayProp("extends", "string", "Optional explicit extends targets."),
			arrayProp("source_ids", "string", "Optional source ids for derived writes."),
			optionalProp("rationale", "string", "Optional rationale for derived writes."),
			optionalProp("top_k", "integer", "Top similar live memories to inspect."),
		),
	}
}

func (t memoryLintRememberTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceID    string   `json:"space_id"`
		Content    string   `json:"content"`
		Kind       string   `json:"kind"`
		ActorID    string   `json:"actor_id"`
		ActorKind  string   `json:"actor_kind"`
		Confidence *float32 `json:"confidence"`
		UpdateID   string   `json:"update_id"`
		Extends    []string `json:"extends"`
		SourceIDs  []string `json:"source_ids"`
		Rationale  string   `json:"rationale"`
		TopK       int      `json:"top_k"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, cfg, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, req.SpaceID)

	confidence := float32(1.0)
	if req.Confidence != nil {
		confidence = *req.Confidence
	}
	actorID := req.ActorID
	if strings.TrimSpace(actorID) == "" {
		actorID = "agent:mcp"
	}
	actorKind := req.ActorKind
	if strings.TrimSpace(actorKind) == "" {
		actorKind = "agent"
	}

	input := omnethdb.MemoryInput{
		SpaceID:    req.SpaceID,
		Content:    req.Content,
		Kind:       parseMemoryKind(req.Kind),
		Actor:      parseActor(actorID, actorKind),
		Confidence: confidence,
		SourceIDs:  req.SourceIDs,
		Rationale:  req.Rationale,
		Relations: omnethdb.MemoryRelations{
			Extends: req.Extends,
		},
	}
	if strings.TrimSpace(req.UpdateID) != "" {
		input.Relations.Updates = []string{req.UpdateID}
	}

	result, err := store.LintRemember(omnethdb.RememberLintRequest{
		Input: input,
		TopK:  req.TopK,
	})
	if err != nil {
		return ToolResult{}, err
	}
	return jsonResult(result)
}

func (t memoryRememberTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_remember",
		Description: "Write a memory into OmnethDB with explicit kind, actor, and optional lineage/derivation relations.",
		InputSchema: objectSchema(
			requiredProp("space_id", "string", "Target space ID."),
			requiredProp("content", "string", "Memory content."),
			requiredProp("kind", "string", "episodic, static, or derived."),
			optionalProp("actor_id", "string", "Writer actor ID."),
			optionalProp("actor_kind", "string", "human, agent, or system."),
			optionalProp("confidence", "number", "Confidence 0..1."),
			optionalProp("update_id", "string", "Optional memory ID to supersede."),
			arrayProp("extends", "string", "Optional explicit extends targets."),
			arrayProp("source_ids", "string", "Required for derived memories."),
			optionalProp("rationale", "string", "Required for derived memories."),
			optionalProp("if_latest_id", "string", "Optional optimistic-lock latest ID."),
			optionalProp("forget_after", "string", "Optional RFC3339 TTL timestamp."),
		),
	}
}

func (t memoryRememberTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceID     string   `json:"space_id"`
		Content     string   `json:"content"`
		Kind        string   `json:"kind"`
		ActorID     string   `json:"actor_id"`
		ActorKind   string   `json:"actor_kind"`
		Confidence  *float32 `json:"confidence"`
		UpdateID    string   `json:"update_id"`
		Extends     []string `json:"extends"`
		SourceIDs   []string `json:"source_ids"`
		Rationale   string   `json:"rationale"`
		IfLatestID  string   `json:"if_latest_id"`
		ForgetAfter string   `json:"forget_after"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, cfg, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, req.SpaceID)

	confidence := float32(1.0)
	if req.Confidence != nil {
		confidence = *req.Confidence
	}
	actorID := req.ActorID
	if strings.TrimSpace(actorID) == "" {
		actorID = "agent:mcp"
	}
	actorKind := req.ActorKind
	if strings.TrimSpace(actorKind) == "" {
		actorKind = "agent"
	}

	var ttl *time.Time
	if strings.TrimSpace(req.ForgetAfter) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ForgetAfter)
		if err != nil {
			return ToolResult{}, err
		}
		ttl = &parsed
	}

	input := omnethdb.MemoryInput{
		SpaceID:     req.SpaceID,
		Content:     req.Content,
		Kind:        parseMemoryKind(req.Kind),
		Actor:       parseActor(actorID, actorKind),
		Confidence:  confidence,
		ForgetAfter: ttl,
		SourceIDs:   req.SourceIDs,
		Rationale:   req.Rationale,
		Relations: omnethdb.MemoryRelations{
			Extends: req.Extends,
		},
	}
	if strings.TrimSpace(req.UpdateID) != "" {
		input.Relations.Updates = []string{req.UpdateID}
	}
	if strings.TrimSpace(req.IfLatestID) != "" {
		input.IfLatestID = &req.IfLatestID
	}

	lint, err := store.LintRemember(omnethdb.RememberLintRequest{
		Input: input,
		TopK:  3,
	})
	if err != nil {
		return ToolResult{}, err
	}

	mem, err := store.Remember(input)
	if err != nil {
		return ToolResult{}, err
	}

	resp := rememberToolResponse{Memory: *mem}
	if len(lint.Warnings) > 0 || len(lint.Suggestions) > 0 || len(lint.Candidates) > 0 {
		resp.Advisory = &rememberAdvisory{
			Warnings:    lint.Warnings,
			Suggestions: lint.Suggestions,
			Candidates:  lint.Candidates,
		}
	}
	return jsonResult(resp)
}

func (t memoryForgetTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_forget",
		Description: "Forget a memory so it leaves the live corpus while staying inspectable in history and audit.",
		InputSchema: objectSchema(
			requiredProp("memory_id", "string", "Memory ID to forget."),
			requiredProp("actor_id", "string", "Actor performing the forget."),
			requiredProp("actor_kind", "string", "human, agent, or system."),
			requiredProp("reason", "string", "Why this memory should leave the live corpus."),
		),
	}
}

func (t memoryForgetTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		MemoryID  string `json:"memory_id"`
		ActorID   string `json:"actor_id"`
		ActorKind string `json:"actor_kind"`
		Reason    string `json:"reason"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, _, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()

	if err := store.Forget(req.MemoryID, parseActor(req.ActorID, req.ActorKind), req.Reason); err != nil {
		return ToolResult{}, err
	}
	return jsonResult(map[string]any{
		"status":    "ok",
		"memory_id": req.MemoryID,
		"reason":    req.Reason,
	})
}

func (t memoryRecallTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_recall",
		Description: "Recall live current knowledge from one or more spaces without traversing graph history.",
		InputSchema: objectSchema(
			requiredArrayProp("space_ids", "string", "Requested spaces."),
			requiredProp("query", "string", "Recall query."),
			optionalProp("top_k", "integer", "Maximum number of results."),
			arrayProp("kinds", "string", "Optional memory kind filters."),
			optionalProp("exclude_orphaned", "boolean", "Exclude orphaned derived memories."),
		),
	}
}

func (t memoryRecallTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceIDs        []string `json:"space_ids"`
		Query           string   `json:"query"`
		TopK            int      `json:"top_k"`
		Kinds           []string `json:"kinds"`
		ExcludeOrphaned bool     `json:"exclude_orphaned"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, cfg, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	ensureEmbeddersForSpaces(store, cfg, req.SpaceIDs)

	topK := req.TopK
	if topK == 0 {
		topK = 10
	}
	results, err := store.Recall(omnethdb.RecallRequest{
		SpaceIDs:               req.SpaceIDs,
		Query:                  req.Query,
		TopK:                   topK,
		Kinds:                  parseMemoryKinds(req.Kinds),
		ExcludeOrphanedDerives: req.ExcludeOrphaned,
	})
	if err != nil {
		return ToolResult{}, err
	}
	return jsonResult(results)
}

func (t memoryProfileTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_profile",
		Description: "Build a layered profile for agent initialization with static and episodic slices kept separate.",
		InputSchema: objectSchema(
			requiredArrayProp("space_ids", "string", "Requested spaces."),
			requiredProp("query", "string", "Profile query."),
			optionalProp("static_top_k", "integer", "Static layer size."),
			optionalProp("episodic_top_k", "integer", "Episodic layer size."),
			optionalProp("exclude_orphaned", "boolean", "Exclude orphaned derived memories."),
		),
	}
}

func (t memoryProfileTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceIDs        []string `json:"space_ids"`
		Query           string   `json:"query"`
		StaticTopK      int      `json:"static_top_k"`
		EpisodicTopK    int      `json:"episodic_top_k"`
		ExcludeOrphaned bool     `json:"exclude_orphaned"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, cfg, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	ensureEmbeddersForSpaces(store, cfg, req.SpaceIDs)

	profile, err := store.GetProfile(omnethdb.ProfileRequest{
		SpaceIDs:               req.SpaceIDs,
		Query:                  req.Query,
		StaticTopK:             req.StaticTopK,
		EpisodicTopK:           req.EpisodicTopK,
		ExcludeOrphanedDerives: req.ExcludeOrphaned,
	})
	if err != nil {
		return ToolResult{}, err
	}
	return jsonResult(profile)
}

func (t memoryProfileCompactTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_profile_compact",
		Description: "Build a compact layered profile for agent initialization using short previews instead of full memory content.",
		InputSchema: objectSchema(
			requiredArrayProp("space_ids", "string", "Requested spaces."),
			requiredProp("query", "string", "Profile query."),
			optionalProp("static_top_k", "integer", "Static layer size."),
			optionalProp("episodic_top_k", "integer", "Episodic layer size."),
			optionalProp("preview_chars", "integer", "Maximum preview length for each memory."),
			optionalProp("exclude_orphaned", "boolean", "Exclude orphaned derived memories."),
		),
	}
}

func (t memoryProfileCompactTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceIDs        []string `json:"space_ids"`
		Query           string   `json:"query"`
		StaticTopK      int      `json:"static_top_k"`
		EpisodicTopK    int      `json:"episodic_top_k"`
		PreviewChars    int      `json:"preview_chars"`
		ExcludeOrphaned bool     `json:"exclude_orphaned"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, cfg, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	ensureEmbeddersForSpaces(store, cfg, req.SpaceIDs)

	profile, err := store.GetProfile(omnethdb.ProfileRequest{
		SpaceIDs:               req.SpaceIDs,
		Query:                  req.Query,
		StaticTopK:             req.StaticTopK,
		EpisodicTopK:           req.EpisodicTopK,
		ExcludeOrphanedDerives: req.ExcludeOrphaned,
	})
	if err != nil {
		return ToolResult{}, err
	}

	previewChars := req.PreviewChars
	if previewChars <= 0 {
		previewChars = 180
	}
	out := compactProfile{
		Static:   compactMemories(profile.Static, previewChars),
		Episodic: compactMemories(profile.Episodic, previewChars),
	}
	return jsonResult(out)
}

func (t memorySynthesisCandidatesTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_synthesis_candidates",
		Description: "Advisory curator-review clusters over live episodic memories that may justify explicit synthesis review.",
		InputSchema: objectSchema(
			requiredProp("space_id", "string", "Target space ID."),
			optionalProp("top_k_per_memory", "integer", "Top similar live episodic memories to inspect per seed memory."),
			optionalProp("max_candidates", "integer", "Maximum synthesis-review clusters to return."),
			optionalProp("min_cluster_size", "integer", "Minimum episodic memories per synthesis-review cluster."),
			optionalProp("min_pair_score", "number", "Minimum cosine similarity for cluster edges."),
		),
	}
}

func (t memorySynthesisCandidatesTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceID        string   `json:"space_id"`
		TopKPerMemory  int      `json:"top_k_per_memory"`
		MaxCandidates  int      `json:"max_candidates"`
		MinClusterSize int      `json:"min_cluster_size"`
		MinPairScore   *float32 `json:"min_pair_score"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, cfg, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, req.SpaceID)

	input := omnethdb.SynthesisCandidatesRequest{
		SpaceID:        req.SpaceID,
		TopKPerMemory:  req.TopKPerMemory,
		MaxCandidates:  req.MaxCandidates,
		MinClusterSize: req.MinClusterSize,
	}
	if req.MinPairScore != nil {
		input.MinPairScore = *req.MinPairScore
	}

	result, err := store.GetSynthesisCandidates(input)
	if err != nil {
		return ToolResult{}, err
	}
	return jsonResult(result)
}

func (t memoryPromotionSuggestionsTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_promotion_suggestions",
		Description: "Advisory curator-review suggestions for live episodic memories that may deserve explicit governed promotion review.",
		InputSchema: objectSchema(
			requiredProp("space_id", "string", "Target space ID."),
			optionalProp("top_k_per_memory", "integer", "Top similar live episodic memories to inspect per seed memory."),
			optionalProp("max_suggestions", "integer", "Maximum promotion-review suggestions to return."),
			optionalProp("min_observation_count", "integer", "Minimum similar episodic observations supporting a suggestion."),
			optionalProp("min_distinct_actors", "integer", "Minimum distinct actors supporting a suggestion."),
			optionalProp("min_distinct_windows", "integer", "Minimum distinct UTC date windows supporting a suggestion."),
			optionalProp("min_cumulative_score", "number", "Minimum cumulative advisory support score."),
			optionalProp("min_pair_score", "number", "Minimum cosine similarity for support edges."),
		),
	}
}

func (t memoryPromotionSuggestionsTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceID             string   `json:"space_id"`
		TopKPerMemory       int      `json:"top_k_per_memory"`
		MaxSuggestions      int      `json:"max_suggestions"`
		MinObservationCount int      `json:"min_observation_count"`
		MinDistinctActors   int      `json:"min_distinct_actors"`
		MinDistinctWindows  int      `json:"min_distinct_windows"`
		MinCumulativeScore  *float32 `json:"min_cumulative_score"`
		MinPairScore        *float32 `json:"min_pair_score"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}

	store, cfg, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, req.SpaceID)

	input := omnethdb.PromotionSuggestionsRequest{
		SpaceID:             req.SpaceID,
		TopKPerMemory:       req.TopKPerMemory,
		MaxSuggestions:      req.MaxSuggestions,
		MinObservationCount: req.MinObservationCount,
		MinDistinctActors:   req.MinDistinctActors,
		MinDistinctWindows:  req.MinDistinctWindows,
	}
	if req.MinCumulativeScore != nil {
		input.MinCumulativeScore = *req.MinCumulativeScore
	}
	if req.MinPairScore != nil {
		input.MinPairScore = *req.MinPairScore
	}

	result, err := store.GetPromotionSuggestions(input)
	if err != nil {
		return ToolResult{}, err
	}
	return jsonResult(result)
}

func (t memoryLineageTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_lineage",
		Description: "Inspect the full lineage history for a root memory, including superseded and forgotten versions.",
		InputSchema: objectSchema(
			requiredProp("root_id", "string", "Root memory ID."),
		),
	}
}

func (t memoryLineageTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		RootID string `json:"root_id"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}
	store, _, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()

	lineage, err := store.GetLineage(req.RootID)
	if err != nil {
		return ToolResult{}, err
	}
	return jsonResult(lineage)
}

func (t memoryRelatedTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_related",
		Description: "Traverse explicit stored relations for inspection without changing retrieval semantics.",
		InputSchema: objectSchema(
			requiredProp("memory_id", "string", "Memory ID to inspect."),
			requiredProp("relation", "string", "updates, extends, or derives."),
			optionalProp("depth", "integer", "Traversal depth."),
		),
	}
}

func (t memoryRelatedTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		MemoryID string `json:"memory_id"`
		Relation string `json:"relation"`
		Depth    int    `json:"depth"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}
	if req.Depth == 0 {
		req.Depth = 1
	}
	store, _, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()

	related, err := store.GetRelated(req.MemoryID, parseRelationType(req.Relation), req.Depth)
	if err != nil {
		return ToolResult{}, err
	}
	return jsonResult(related)
}

func (t memoryExportSummaryTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "memory_export_summary",
		Description: "Render a human-readable Markdown summary of the current cold-path memory state for a space.",
		InputSchema: objectSchema(
			requiredProp("space_id", "string", "Target space ID."),
		),
	}
}

func (t memoryExportSummaryTool) Call(_ context.Context, args map[string]any) (ToolResult, error) {
	var req struct {
		SpaceID string `json:"space_id"`
	}
	if err := decodeArgs(args, &req); err != nil {
		return ToolResult{}, err
	}
	store, _, err := t.open()
	if err != nil {
		return ToolResult{}, err
	}
	defer store.Close()

	summary, err := store.RenderExportSummaryMarkdown(omnethdb.ExportRequest{SpaceID: req.SpaceID})
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ToolContent{{Type: "text", Text: summary}},
		StructuredContent: map[string]any{
			"space_id": req.SpaceID,
			"summary":  summary,
		},
	}, nil
}

func decodeArgs(args map[string]any, dest any) error {
	raw, err := json.Marshal(args)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func jsonResult(v any) (ToolResult, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ToolResult{}, err
	}
	structured, err := structuredContentFor(v)
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content:           []ToolContent{{Type: "text", Text: string(raw)}},
		StructuredContent: structured,
	}, nil
}

func structuredContentFor(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	if record, ok := decoded.(map[string]any); ok {
		return record, nil
	}
	return map[string]any{
		"result": decoded,
		"type":   fmt.Sprintf("%T", v),
	}, nil
}

func objectSchema(properties ...schemaProp) map[string]any {
	props := make(map[string]any, len(properties))
	required := make([]string, 0)
	for _, property := range properties {
		props[property.Name] = property.Schema
		if property.Required {
			required = append(required, property.Name)
		}
	}
	out := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func compactMemories(memories []omnethdb.ScoredMemory, previewChars int) []compactScoredMemory {
	if len(memories) == 0 {
		return nil
	}
	out := make([]compactScoredMemory, 0, len(memories))
	for _, mem := range memories {
		out = append(out, compactScoredMemory{
			ID:             mem.ID,
			SpaceID:        mem.SpaceID,
			Kind:           memoryKindLabel(mem.Kind),
			ContentPreview: previewText(mem.Content, previewChars),
			Score:          mem.Score,
			Confidence:     mem.Confidence,
			Actor:          mem.Actor,
			Version:        mem.Version,
			Latest:         mem.IsLatest,
			Forgotten:      mem.IsForgotten,
			Orphaned:       mem.HasOrphanedSources,
		})
	}
	return out
}

func previewText(v string, maxChars int) string {
	v = strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
	if maxChars <= 0 || len([]rune(v)) <= maxChars {
		return v
	}
	runes := []rune(v)
	if maxChars <= 1 {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-1]) + "…"
}

func memoryKindLabel(kind omnethdb.MemoryKind) string {
	switch kind {
	case omnethdb.KindEpisodic:
		return "episodic"
	case omnethdb.KindStatic:
		return "static"
	case omnethdb.KindDerived:
		return "derived"
	default:
		return "unknown"
	}
}

type schemaProp struct {
	Name     string
	Schema   map[string]any
	Required bool
}

func requiredProp(name string, typ string, description string) schemaProp {
	return schemaProp{
		Name:     name,
		Required: true,
		Schema: map[string]any{
			"type":        typ,
			"description": description,
		},
	}
}

func optionalProp(name string, typ string, description string) schemaProp {
	return schemaProp{
		Name: name,
		Schema: map[string]any{
			"type":        typ,
			"description": description,
		},
	}
}

func arrayProp(name string, itemType string, description string) schemaProp {
	return schemaProp{
		Name: name,
		Schema: map[string]any{
			"type":        "array",
			"description": description,
			"items": map[string]any{
				"type": itemType,
			},
		},
	}
}

func requiredArrayProp(name string, itemType string, description string) schemaProp {
	return schemaProp{
		Name:     name,
		Required: true,
		Schema: map[string]any{
			"type":        "array",
			"description": description,
			"items": map[string]any{
				"type": itemType,
			},
		},
	}
}

func embedderForBootstrap(cfg *omnethdb.RuntimeConfig, spaceID string) omnethdb.Embedder {
	if cfg != nil {
		if settings, ok := cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				return hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions)
			}
		}
	}
	return hashembedder.New(defaultModelID, defaultDimension)
}

func ensureEmbeddersForSpaces(store *omnethdb.Store, cfg *omnethdb.RuntimeConfig, spaceIDs []string) {
	for _, spaceID := range spaceIDs {
		ensureEmbedderForSpace(store, cfg, spaceID)
	}
}

func ensureEmbedderForSpace(store *omnethdb.Store, cfg *omnethdb.RuntimeConfig, spaceID string) {
	if store == nil || strings.TrimSpace(spaceID) == "" {
		return
	}
	if cfg != nil {
		if settings, ok := cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				store.RegisterEmbedder(hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions))
				return
			}
		}
	}
	if persisted, err := store.GetSpaceConfig(spaceID); err == nil {
		store.RegisterEmbedder(hashembedder.New(persisted.EmbeddingModelID, persisted.Dimension))
	}
}

func parseMemoryKind(raw string) omnethdb.MemoryKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "episodic":
		return omnethdb.KindEpisodic
	case "static":
		return omnethdb.KindStatic
	case "derived":
		return omnethdb.KindDerived
	default:
		return omnethdb.KindUnknown
	}
}

func parseMemoryKinds(raw []string) []omnethdb.MemoryKind {
	out := make([]omnethdb.MemoryKind, 0, len(raw))
	for _, item := range raw {
		out = append(out, parseMemoryKind(item))
	}
	return out
}

func parseActor(id string, kind string) omnethdb.Actor {
	return omnethdb.Actor{ID: id, Kind: parseActorKind(kind)}
}

func parseActorKind(raw string) omnethdb.ActorKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "human":
		return omnethdb.ActorHuman
	case "agent":
		return omnethdb.ActorAgent
	case "system":
		return omnethdb.ActorSystem
	default:
		return omnethdb.ActorSystem + 99
	}
}

func parseRelationType(raw string) omnethdb.RelationType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "updates":
		return omnethdb.RelationUpdates
	case "extends":
		return omnethdb.RelationExtends
	case "derives":
		return omnethdb.RelationDerives
	default:
		return omnethdb.RelationType(raw)
	}
}
