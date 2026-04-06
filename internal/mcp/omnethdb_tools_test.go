package mcp

import (
	"context"
	"path/filepath"
	"testing"

	omnethdb "omnethdb"
	hashembedder "omnethdb/embedders/hash"
)

type mcpMapEmbedder struct {
	modelID    string
	dimensions int
	vectors    map[string][]float32
}

func (e mcpMapEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if vec, ok := e.vectors[text]; ok {
		out := make([]float32, len(vec))
		copy(out, vec)
		return out, nil
	}
	return make([]float32, e.dimensions), nil
}

func (e mcpMapEmbedder) Dimensions() int { return e.dimensions }
func (e mcpMapEmbedder) ModelID() string { return e.modelID }

func TestStructuredContentForWrapsArraysIntoRecords(t *testing.T) {
	t.Parallel()

	value, err := structuredContentFor([]map[string]any{
		{"id": "mem-1"},
	})
	if err != nil {
		t.Fatalf("structuredContentFor returned unexpected error: %v", err)
	}

	raw, ok := value["result"]
	if !ok {
		t.Fatalf("expected wrapped result field, got %#v", value)
	}
	items, ok := raw.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected wrapped array result, got %#v", value)
	}
}

func TestMemoryLintRememberToolReturnsWarningsAndCandidates(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, layout, err := omnethdb.OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace returned unexpected error: %v", err)
	}
	embedder := hashembedder.New("builtin/hash-embedder-v1", 256)
	store.RegisterEmbedder(embedder)
	if _, err := store.EnsureSpace("repo:company/app", embedder, omnethdb.SpaceInit{
		DefaultWeight: 1,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	}); err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}
	if _, err := store.Remember(omnethdb.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "existing fact",
		Kind:       omnethdb.KindStatic,
		Actor:      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	opener := func() (*omnethdb.Store, *omnethdb.RuntimeConfig, error) {
		store, _, err := omnethdb.OpenWorkspace(root)
		if err != nil {
			return nil, nil, err
		}
		cfg, err := omnethdb.LoadRuntimeConfig(filepath.Join(layout.RootDir, "config.toml"))
		if err != nil {
			_ = store.Close()
			return nil, nil, err
		}
		store.RegisterEmbedder(embedder)
		return store, cfg, nil
	}

	var lintTool memoryLintRememberTool
	found := false
	for _, tool := range NewOmnethDBTools(opener) {
		if candidate, ok := tool.(memoryLintRememberTool); ok {
			lintTool = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected memory_lint_remember tool to be registered")
	}

	result, err := lintTool.Call(context.Background(), map[string]any{
		"space_id": "repo:company/app",
		"content":  "existing fact",
		"kind":     "static",
	})
	if err != nil {
		t.Fatalf("Call returned unexpected error: %v", err)
	}
	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %#v", result.StructuredContent)
	}
	warnings, ok := sc["warnings"].([]any)
	if !ok || len(warnings) == 0 {
		t.Fatalf("expected lint warnings in structured content, got %#v", sc)
	}
	candidates, ok := sc["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		t.Fatalf("expected lint candidates in structured content, got %#v", sc)
	}
}

func TestMemoryRememberToolIncludesAdvisoryWarnings(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, layout, err := omnethdb.OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace returned unexpected error: %v", err)
	}
	embedder := hashembedder.New("builtin/hash-embedder-v1", 256)
	store.RegisterEmbedder(embedder)
	if _, err := store.EnsureSpace("repo:company/app", embedder, omnethdb.SpaceInit{
		DefaultWeight: 1,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	}); err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}
	if _, err := store.Remember(omnethdb.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "existing fact",
		Kind:       omnethdb.KindStatic,
		Actor:      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	opener := func() (*omnethdb.Store, *omnethdb.RuntimeConfig, error) {
		store, _, err := omnethdb.OpenWorkspace(root)
		if err != nil {
			return nil, nil, err
		}
		cfg, err := omnethdb.LoadRuntimeConfig(filepath.Join(layout.RootDir, "config.toml"))
		if err != nil {
			_ = store.Close()
			return nil, nil, err
		}
		store.RegisterEmbedder(embedder)
		return store, cfg, nil
	}

	var rememberTool memoryRememberTool
	found := false
	for _, tool := range NewOmnethDBTools(opener) {
		if candidate, ok := tool.(memoryRememberTool); ok {
			rememberTool = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected memory_remember tool to be registered")
	}

	result, err := rememberTool.Call(context.Background(), map[string]any{
		"space_id":   "repo:company/app",
		"content":    "existing fact",
		"kind":       "static",
		"actor_id":   "user:bob",
		"actor_kind": "human",
	})
	if err != nil {
		t.Fatalf("Call returned unexpected error: %v", err)
	}
	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %#v", result.StructuredContent)
	}
	if _, ok := sc["ID"].(string); !ok {
		t.Fatalf("expected written memory fields to remain top-level, got %#v", sc)
	}
	advisory, ok := sc["advisory"].(map[string]any)
	if !ok {
		t.Fatalf("expected advisory section, got %#v", sc)
	}
	warnings, ok := advisory["warnings"].([]any)
	if !ok || len(warnings) == 0 {
		t.Fatalf("expected advisory warnings, got %#v", advisory)
	}
	suggestions, ok := advisory["suggestions"].([]any)
	if !ok || len(suggestions) == 0 {
		t.Fatalf("expected advisory suggestions, got %#v", advisory)
	}
}

func TestMemoryProfileCompactToolReturnsPreviews(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, layout, err := omnethdb.OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace returned unexpected error: %v", err)
	}
	embedder := hashembedder.New("builtin/hash-embedder-v1", 256)
	store.RegisterEmbedder(embedder)
	if _, err := store.EnsureSpace("repo:company/app", embedder, omnethdb.SpaceInit{
		DefaultWeight: 1,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	}); err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}
	longText := "Platform scale snapshot: onboarding has many tabs including bio, achievements, fee, rider, and media. DeepSeek prefill should not overwrite verified user-entered values."
	if _, err := store.Remember(omnethdb.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    longText,
		Kind:       omnethdb.KindStatic,
		Actor:      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	opener := func() (*omnethdb.Store, *omnethdb.RuntimeConfig, error) {
		store, _, err := omnethdb.OpenWorkspace(root)
		if err != nil {
			return nil, nil, err
		}
		cfg, err := omnethdb.LoadRuntimeConfig(filepath.Join(layout.RootDir, "config.toml"))
		if err != nil {
			_ = store.Close()
			return nil, nil, err
		}
		store.RegisterEmbedder(embedder)
		return store, cfg, nil
	}

	var compactTool memoryProfileCompactTool
	found := false
	for _, tool := range NewOmnethDBTools(opener) {
		if candidate, ok := tool.(memoryProfileCompactTool); ok {
			compactTool = candidate
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected memory_profile_compact tool to be registered")
	}

	result, err := compactTool.Call(context.Background(), map[string]any{
		"space_ids":      []string{"repo:company/app"},
		"query":          "onboarding profile tabs",
		"preview_chars":  60,
		"static_top_k":   5,
		"episodic_top_k": 5,
	})
	if err != nil {
		t.Fatalf("Call returned unexpected error: %v", err)
	}
	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %#v", result.StructuredContent)
	}
	staticItems, ok := sc["static"].([]any)
	if !ok || len(staticItems) == 0 {
		t.Fatalf("expected compact static memories, got %#v", sc)
	}
	first, ok := staticItems[0].(map[string]any)
	if !ok {
		t.Fatalf("expected compact memory object, got %#v", staticItems[0])
	}
	preview, ok := first["content_preview"].(string)
	if !ok || preview == "" {
		t.Fatalf("expected content_preview, got %#v", first)
	}
	if preview == longText {
		t.Fatalf("expected compact preview instead of full content, got %q", preview)
	}
	if _, exists := first["Content"]; exists {
		t.Fatalf("expected no full Content field in compact response, got %#v", first)
	}
}

func TestMemorySynthesisCandidatesToolReturnsCandidates(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, layout, err := omnethdb.OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace returned unexpected error: %v", err)
	}
	embedder := mcpMapEmbedder{
		modelID:    "test/mcp-synthesis",
		dimensions: 2,
		vectors: map[string][]float32{
			"cache timeout affected job run alpha": {1, 0},
			"cache timeout affected job run beta":  {0.98, 0.02},
			"cache timeout affected job run gamma": {0.97, 0.03},
		},
	}
	store.RegisterEmbedder(embedder)
	if _, err := store.EnsureSpace("repo:company/app", embedder, omnethdb.SpaceInit{
		DefaultWeight: 1,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	}); err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}
	for _, item := range []omnethdb.MemoryInput{
		{SpaceID: "repo:company/app", Content: "cache timeout affected job run alpha", Kind: omnethdb.KindEpisodic, Actor: omnethdb.Actor{ID: "agent:one", Kind: omnethdb.ActorAgent}, Confidence: 0.8},
		{SpaceID: "repo:company/app", Content: "cache timeout affected job run beta", Kind: omnethdb.KindEpisodic, Actor: omnethdb.Actor{ID: "agent:two", Kind: omnethdb.ActorAgent}, Confidence: 0.85},
		{SpaceID: "repo:company/app", Content: "cache timeout affected job run gamma", Kind: omnethdb.KindEpisodic, Actor: omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman}, Confidence: 0.9},
	} {
		if _, err := store.Remember(item); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	opener := func() (*omnethdb.Store, *omnethdb.RuntimeConfig, error) {
		store, _, err := omnethdb.OpenWorkspace(root)
		if err != nil {
			return nil, nil, err
		}
		cfg, err := omnethdb.LoadRuntimeConfig(filepath.Join(layout.RootDir, "config.toml"))
		if err != nil {
			_ = store.Close()
			return nil, nil, err
		}
		store.RegisterEmbedder(embedder)
		return store, cfg, nil
	}

	var tool memorySynthesisCandidatesTool
	found := false
	for _, candidate := range NewOmnethDBTools(opener) {
		if current, ok := candidate.(memorySynthesisCandidatesTool); ok {
			tool = current
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected memory_synthesis_candidates tool to be registered")
	}

	result, err := tool.Call(context.Background(), map[string]any{
		"space_id":         "repo:company/app",
		"min_cluster_size": 2,
	})
	if err != nil {
		t.Fatalf("Call returned unexpected error: %v", err)
	}
	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %#v", result.StructuredContent)
	}
	candidates, ok := sc["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		t.Fatalf("expected synthesis candidates in structured content, got %#v", sc)
	}
}

func TestMemoryPromotionSuggestionsToolReturnsSuggestions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, layout, err := omnethdb.OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace returned unexpected error: %v", err)
	}
	embedder := hashembedder.New("builtin/hash-embedder-v1", 256)
	store.RegisterEmbedder(embedder)
	if _, err := store.EnsureSpace("repo:company/app", embedder, omnethdb.SpaceInit{
		DefaultWeight: 1,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	}); err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}
	for _, item := range []omnethdb.MemoryInput{
		{SpaceID: "repo:company/app", Content: "repository requires reviewed migrations", Kind: omnethdb.KindEpisodic, Actor: omnethdb.Actor{ID: "agent:one", Kind: omnethdb.ActorAgent}, Confidence: 0.8},
		{SpaceID: "repo:company/app", Content: "repository requires reviewed migrations", Kind: omnethdb.KindEpisodic, Actor: omnethdb.Actor{ID: "agent:two", Kind: omnethdb.ActorAgent}, Confidence: 0.85},
		{SpaceID: "repo:company/app", Content: "repository requires reviewed migrations", Kind: omnethdb.KindEpisodic, Actor: omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman}, Confidence: 0.9},
	} {
		if _, err := store.Remember(item); err != nil {
			t.Fatalf("Remember returned unexpected error: %v", err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	opener := func() (*omnethdb.Store, *omnethdb.RuntimeConfig, error) {
		store, _, err := omnethdb.OpenWorkspace(root)
		if err != nil {
			return nil, nil, err
		}
		cfg, err := omnethdb.LoadRuntimeConfig(filepath.Join(layout.RootDir, "config.toml"))
		if err != nil {
			_ = store.Close()
			return nil, nil, err
		}
		store.RegisterEmbedder(embedder)
		return store, cfg, nil
	}

	var tool memoryPromotionSuggestionsTool
	found := false
	for _, candidate := range NewOmnethDBTools(opener) {
		if current, ok := candidate.(memoryPromotionSuggestionsTool); ok {
			tool = current
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected memory_promotion_suggestions tool to be registered")
	}

	result, err := tool.Call(context.Background(), map[string]any{
		"space_id":              "repo:company/app",
		"min_observation_count": 2,
		"min_distinct_actors":   2,
		"min_cumulative_score":  2.5,
	})
	if err != nil {
		t.Fatalf("Call returned unexpected error: %v", err)
	}
	sc, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %#v", result.StructuredContent)
	}
	suggestions, ok := sc["suggestions"].([]any)
	if !ok || len(suggestions) == 0 {
		t.Fatalf("expected promotion suggestions in structured content, got %#v", sc)
	}
}
