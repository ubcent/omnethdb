package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	omnethdb "omnethdb"
)

type testEmbedder struct {
	modelID    string
	dimensions int
}

type mappedEmbedder struct {
	modelID    string
	dimensions int
	vectors    map[string][]float32
}

func (e testEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	out := make([]float32, e.dimensions)
	for i := range out {
		out[i] = float32(len(text)+i+1) / 100
	}
	return out, nil
}

func (e testEmbedder) Dimensions() int { return e.dimensions }
func (e testEmbedder) ModelID() string { return e.modelID }

func (e mappedEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if vector, ok := e.vectors[text]; ok {
		return append([]float32(nil), vector...), nil
	}
	out := make([]float32, e.dimensions)
	return out, nil
}

func (e mappedEmbedder) Dimensions() int { return e.dimensions }
func (e mappedEmbedder) ModelID() string { return e.modelID }

func TestHTTPAPIEndToEndMemoryLifecycle(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "builtin/hash-embedder-v1",
					Dimensions: 256,
				},
			},
		},
	}
	store.RegisterEmbedder(testEmbedder{modelID: "builtin/hash-embedder-v1", dimensions: 256})

	server := httptest.NewServer(NewHandler(store, cfg))
	defer server.Close()

	doJSON(t, http.MethodPost, server.URL+"/v1/spaces/init", map[string]any{
		"space_id": "repo:company/app",
	}, http.StatusOK, nil)

	var root omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "payments use cursor pagination",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
		"Relations":  omnethdb.MemoryRelations{},
	}, http.StatusOK, &root)

	var updated omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "payments use signed cursor pagination",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
		"IfLatestID": root.ID,
		"Relations": map[string]any{
			"Updates": []string{root.ID},
		},
	}, http.StatusOK, &updated)

	var recall []omnethdb.ScoredMemory
	doJSON(t, http.MethodPost, server.URL+"/v1/recall", map[string]any{
		"SpaceIDs": []string{"repo:company/app"},
		"Query":    "signed cursor pagination",
		"TopK":     5,
	}, http.StatusOK, &recall)
	if len(recall) != 1 || recall[0].ID != updated.ID {
		t.Fatalf("unexpected recall results: %#v", recall)
	}

	var lineage []omnethdb.Memory
	doJSON(t, http.MethodGet, server.URL+"/v1/lineages/"+root.ID, nil, http.StatusOK, &lineage)
	if len(lineage) != 2 {
		t.Fatalf("expected two lineage versions, got %#v", lineage)
	}

	doJSON(t, http.MethodPost, server.URL+"/v1/memories/"+updated.ID+"/forget", map[string]any{
		"actor":  omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"reason": "obsolete",
	}, http.StatusOK, nil)

	var revived omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/lineages/"+root.ID+"/revive", map[string]any{
		"Content":    "payments use signed cursor pagination again",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
	}, http.StatusOK, &revived)
	if !revived.IsLatest {
		t.Fatalf("expected revived memory to be latest, got %#v", revived)
	}

	var audit []map[string]any
	doJSON(t, http.MethodGet, server.URL+"/v1/audit?space_id=repo:company/app", nil, http.StatusOK, &audit)
	if len(audit) == 0 {
		t.Fatal("expected audit history to be present")
	}
}

func TestHTTPAPIExposesRelatedAndCandidates(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "builtin/hash-embedder-v1",
					Dimensions: 256,
				},
			},
		},
	}
	store.RegisterEmbedder(testEmbedder{modelID: "builtin/hash-embedder-v1", dimensions: 256})
	server := httptest.NewServer(NewHandler(store, cfg))
	defer server.Close()

	doJSON(t, http.MethodPost, server.URL+"/v1/spaces/init", map[string]any{
		"space_id": "repo:company/app",
	}, http.StatusOK, nil)

	var base omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "payments schema uses ledger entries",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
	}, http.StatusOK, &base)

	var ext omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "ledger entries are append-only",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
		"Relations": map[string]any{
			"Extends": []string{base.ID},
		},
	}, http.StatusOK, &ext)

	var candidates []omnethdb.ScoredMemory
	doJSON(t, http.MethodPost, server.URL+"/v1/candidates", map[string]any{
		"SpaceID": "repo:company/app",
		"Content": "ledger entries",
		"TopK":    5,
	}, http.StatusOK, &candidates)
	if len(candidates) == 0 {
		t.Fatal("expected candidate results")
	}

	var related []omnethdb.Memory
	doJSON(t, http.MethodGet, server.URL+"/v1/memories/"+ext.ID+"/related/extends?depth=1", nil, http.StatusOK, &related)
	if len(related) != 1 || related[0].ID != base.ID {
		t.Fatalf("unexpected related response: %#v", related)
	}
}

func TestHTTPAPIExposesQualityDiagnostics(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "test/http-quality",
					Dimensions: 2,
				},
			},
		},
	}
	store.RegisterEmbedder(mappedEmbedder{
		modelID:    "test/http-quality",
		dimensions: 2,
		vectors: map[string][]float32{
			"payments use signed cursor pagination":       {1, 0},
			"payments use signed cursor pagination again": {1, 0},
			"payments use HMAC signed cursor pagination":  {0.88, 0.3},
			"ledger entries are append-only":              {0, 1},
		},
	})
	server := httptest.NewServer(NewHandler(store, cfg))
	defer server.Close()

	doJSON(t, http.MethodPost, server.URL+"/v1/spaces/init", map[string]any{
		"space_id": "repo:company/app",
	}, http.StatusOK, nil)

	for _, content := range []string{
		"payments use signed cursor pagination",
		"payments use signed cursor pagination again",
		"payments use HMAC signed cursor pagination",
		"ledger entries are append-only",
	} {
		doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
			"SpaceID":    "repo:company/app",
			"Content":    content,
			"Kind":       omnethdb.KindStatic,
			"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
			"Confidence": 1.0,
		}, http.StatusOK, nil)
	}

	var diagnostics omnethdb.QualityDiagnosticsResult
	doJSON(t, http.MethodGet, server.URL+"/v1/diagnostics/quality?space_id=repo:company/app", nil, http.StatusOK, &diagnostics)
	if diagnostics.LiveStaticCount != 4 {
		t.Fatalf("unexpected live static count: %#v", diagnostics)
	}
	if len(diagnostics.DuplicateGroups) == 0 {
		t.Fatalf("expected duplicate groups, got %#v", diagnostics)
	}
	if len(diagnostics.PossibleUpdates) == 0 {
		t.Fatalf("expected possible updates, got %#v", diagnostics)
	}

	var cleanupPlan omnethdb.QualityCleanupPlanResult
	doJSON(t, http.MethodGet, server.URL+"/v1/diagnostics/quality/cleanup-plan?space_id=repo:company/app", nil, http.StatusOK, &cleanupPlan)
	if len(cleanupPlan.DuplicateSuggestions) == 0 {
		t.Fatalf("expected duplicate cleanup suggestions, got %#v", cleanupPlan)
	}
	if cleanupPlan.SuggestedForgetBatchCommand == "" {
		t.Fatalf("expected suggested forget-batch command, got %#v", cleanupPlan)
	}
}

func TestHTTPAPIExposesAdvisoryCurationDiagnostics(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "test/http-curation",
					Dimensions: 2,
				},
			},
		},
	}
	store.RegisterEmbedder(mappedEmbedder{
		modelID:    "test/http-curation",
		dimensions: 2,
		vectors: map[string][]float32{
			"job run failed with cache timeout":         {1, 0},
			"another job run failed with cache timeout": {0.98, 0.02},
			"cache timeout affected a later job run":    {0.97, 0.03},
			"team requires reviewed migrations":         {0, 1},
			"repository requires reviewed migrations":   {0.02, 0.98},
			"migration policy requires review approval": {0.03, 0.97},
		},
	})
	server := httptest.NewServer(NewHandler(store, cfg))
	defer server.Close()

	doJSON(t, http.MethodPost, server.URL+"/v1/spaces/init", map[string]any{
		"space_id": "repo:company/app",
	}, http.StatusOK, nil)

	for _, item := range []struct {
		content string
		actor   omnethdb.Actor
	}{
		{"job run failed with cache timeout", omnethdb.Actor{ID: "agent:one", Kind: omnethdb.ActorAgent}},
		{"another job run failed with cache timeout", omnethdb.Actor{ID: "agent:two", Kind: omnethdb.ActorAgent}},
		{"cache timeout affected a later job run", omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman}},
		{"team requires reviewed migrations", omnethdb.Actor{ID: "agent:one", Kind: omnethdb.ActorAgent}},
		{"repository requires reviewed migrations", omnethdb.Actor{ID: "agent:two", Kind: omnethdb.ActorAgent}},
		{"migration policy requires review approval", omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman}},
	} {
		doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
			"SpaceID":    "repo:company/app",
			"Content":    item.content,
			"Kind":       omnethdb.KindEpisodic,
			"Actor":      item.actor,
			"Confidence": 0.9,
		}, http.StatusOK, nil)
	}

	var synthesis omnethdb.SynthesisCandidatesResult
	doJSON(t, http.MethodGet, server.URL+"/v1/diagnostics/quality/synthesis-candidates?space_id=repo:company/app", nil, http.StatusOK, &synthesis)
	if synthesis.LiveEpisodicCount != 6 {
		t.Fatalf("unexpected live episodic count: %#v", synthesis)
	}
	if len(synthesis.Candidates) == 0 {
		t.Fatalf("expected synthesis candidates, got %#v", synthesis)
	}

	var promotion omnethdb.PromotionSuggestionsResult
	doJSON(t, http.MethodGet, server.URL+"/v1/diagnostics/quality/promotion-suggestions?space_id=repo:company/app&min_observation_count=2&min_distinct_actors=2&min_cumulative_score=2.5", nil, http.StatusOK, &promotion)
	if promotion.LiveEpisodicCount != 6 {
		t.Fatalf("unexpected live episodic count: %#v", promotion)
	}
	if len(promotion.Suggestions) == 0 {
		t.Fatalf("expected promotion suggestions, got %#v", promotion)
	}
}

func TestHTTPAPIExposesBatchForget(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "builtin/hash-embedder-v1",
					Dimensions: 256,
				},
			},
		},
	}
	store.RegisterEmbedder(testEmbedder{modelID: "builtin/hash-embedder-v1", dimensions: 256})
	server := httptest.NewServer(NewHandler(store, cfg))
	defer server.Close()

	doJSON(t, http.MethodPost, server.URL+"/v1/spaces/init", map[string]any{
		"space_id": "repo:company/app",
	}, http.StatusOK, nil)

	var first omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "duplicate fact one",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
	}, http.StatusOK, &first)

	var second omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "duplicate fact two",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
	}, http.StatusOK, &second)

	doJSON(t, http.MethodPost, server.URL+"/v1/memories/forget-batch", map[string]any{
		"memory_ids": []string{first.ID, second.ID},
		"actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"reason":     "duplicate cleanup",
	}, http.StatusOK, nil)

	var listed []omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/list", map[string]any{
		"SpaceIDs": []string{"repo:company/app"},
		"Kinds":    []omnethdb.MemoryKind{omnethdb.KindStatic},
	}, http.StatusOK, &listed)
	if len(listed) != 0 {
		t.Fatalf("expected no live memories after batch forget, got %#v", listed)
	}
}

func TestHTTPAPIExposesStructuredListAndBatchRemember(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "builtin/hash-embedder-v1",
					Dimensions: 256,
				},
			},
		},
	}
	store.RegisterEmbedder(testEmbedder{modelID: "builtin/hash-embedder-v1", dimensions: 256})
	server := httptest.NewServer(NewHandler(store, cfg))
	defer server.Close()

	doJSON(t, http.MethodPost, server.URL+"/v1/spaces/init", map[string]any{
		"space_id": "repo:company/app",
	}, http.StatusOK, nil)

	var written []omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember-batch", map[string]any{
		"inputs": []map[string]any{
			{
				"SpaceID":    "repo:company/app",
				"Content":    "adr one",
				"Kind":       omnethdb.KindStatic,
				"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
				"Confidence": 1.0,
			},
			{
				"SpaceID":    "repo:company/app",
				"Content":    "adr two",
				"Kind":       omnethdb.KindStatic,
				"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
				"Confidence": 1.0,
			},
		},
	}, http.StatusOK, &written)
	if len(written) != 2 {
		t.Fatalf("unexpected batch remember response: %#v", written)
	}

	var listed []omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/list", map[string]any{
		"SpaceIDs": []string{"repo:company/app"},
		"Kinds":    []omnethdb.MemoryKind{omnethdb.KindStatic},
	}, http.StatusOK, &listed)
	if len(listed) != 2 {
		t.Fatalf("unexpected structured list response: %#v", listed)
	}
}

func TestHTTPAPIExposesExportFormats(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "builtin/hash-embedder-v1",
					Dimensions: 256,
				},
			},
		},
	}
	store.RegisterEmbedder(testEmbedder{modelID: "builtin/hash-embedder-v1", dimensions: 256})
	server := httptest.NewServer(NewHandler(store, cfg))
	defer server.Close()

	doJSON(t, http.MethodPost, server.URL+"/v1/spaces/init", map[string]any{
		"space_id": "repo:company/app",
	}, http.StatusOK, nil)

	var root omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "payments use cursor pagination",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
	}, http.StatusOK, &root)

	var updated omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "payments use signed cursor pagination",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
		"IfLatestID": root.ID,
		"Relations": map[string]any{
			"Updates": []string{root.ID},
		},
	}, http.StatusOK, &updated)

	var snapshot omnethdb.ExportSnapshot
	doJSON(t, http.MethodGet, server.URL+"/v1/export?space_id=repo:company/app", nil, http.StatusOK, &snapshot)
	if snapshot.SpaceID != "repo:company/app" || len(snapshot.LiveMemories) != 1 || snapshot.LiveMemories[0].ID != updated.ID {
		t.Fatalf("unexpected export snapshot: %#v", snapshot)
	}

	md := doText(t, http.MethodGet, server.URL+"/v1/export?space_id=repo:company/app&format=summary-md", http.StatusOK)
	if !strings.Contains(md, "## Live Corpus") || !strings.Contains(md, "Root "+root.ID) {
		t.Fatalf("unexpected markdown export:\n%s", md)
	}

	mermaid := doText(t, http.MethodGet, server.URL+"/v1/export?space_id=repo:company/app&format=graph-mermaid", http.StatusOK)
	if !strings.Contains(mermaid, "graph TD") || !strings.Contains(mermaid, "|updates|") {
		t.Fatalf("unexpected mermaid export:\n%s", mermaid)
	}
}

func TestHTTPAPIExposesInspectorPage(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := httptest.NewServer(NewHandler(store, &omnethdb.RuntimeConfig{}))
	defer server.Close()

	page := doText(t, http.MethodGet, server.URL+"/inspect?space_id=repo:company/app", http.StatusOK)
	if !strings.Contains(page, "OmnethDB Inspector") || !strings.Contains(page, "/v1/export?space_id=") || !strings.Contains(page, "repo:company/app") || !strings.Contains(page, "Signals") || !strings.Contains(page, "Quality") || !strings.Contains(page, "Cleanup Plan") || !strings.Contains(page, "Curation Review") || !strings.Contains(page, "Live View") || !strings.Contains(page, "before_file") || !strings.Contains(page, "after_file") {
		t.Fatalf("unexpected inspector page:\n%s", page)
	}
}

func TestHTTPAPIExposesExportDiff(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := &omnethdb.RuntimeConfig{
		Spaces: map[string]omnethdb.RuntimeSpaceSettings{
			"repo:company/app": {
				Embedder: omnethdb.RuntimeEmbedderConfig{
					ModelID:    "builtin/hash-embedder-v1",
					Dimensions: 256,
				},
			},
		},
	}
	store.RegisterEmbedder(testEmbedder{modelID: "builtin/hash-embedder-v1", dimensions: 256})
	server := httptest.NewServer(NewHandler(store, cfg))
	defer server.Close()

	doJSON(t, http.MethodPost, server.URL+"/v1/spaces/init", map[string]any{
		"space_id": "repo:company/app",
	}, http.StatusOK, nil)

	var root omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "payments use cursor pagination",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
	}, http.StatusOK, &root)
	since := time.Now().UTC()
	time.Sleep(5 * time.Millisecond)

	var updated omnethdb.Memory
	doJSON(t, http.MethodPost, server.URL+"/v1/memories/remember", map[string]any{
		"SpaceID":    "repo:company/app",
		"Content":    "payments use signed cursor pagination",
		"Kind":       omnethdb.KindStatic,
		"Actor":      omnethdb.Actor{ID: "user:alice", Kind: omnethdb.ActorHuman},
		"Confidence": 1.0,
		"IfLatestID": root.ID,
		"Relations": map[string]any{
			"Updates": []string{root.ID},
		},
	}, http.StatusOK, &updated)

	var diff omnethdb.ExportDiff
	doJSON(t, http.MethodGet, server.URL+"/v1/export/diff?space_id=repo:company/app&since="+since.Format(time.RFC3339Nano), nil, http.StatusOK, &diff)
	if diff.SpaceID != "repo:company/app" {
		t.Fatalf("unexpected export diff: %#v", diff)
	}
	if len(diff.AddedLiveMemories) != 1 || diff.AddedLiveMemories[0].ID != updated.ID {
		t.Fatalf("unexpected added live memories: %#v", diff.AddedLiveMemories)
	}
	if len(diff.RemovedLiveMemories) != 1 || diff.RemovedLiveMemories[0].ID != root.ID {
		t.Fatalf("unexpected removed live memories: %#v", diff.RemovedLiveMemories)
	}
	if len(diff.AddedAuditEntries) == 0 {
		t.Fatalf("expected diff audit entries, got %#v", diff)
	}
}

func TestHTTPAPIExposesExportCompare(t *testing.T) {
	t.Parallel()

	before := omnethdb.ExportSnapshot{
		SpaceID:     "repo:company/app",
		GeneratedAt: time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
		LiveMemories: []omnethdb.Memory{
			{ID: "mem-before", SpaceID: "repo:company/app", Content: "old fact", Kind: omnethdb.KindStatic, Version: 1, IsLatest: true},
		},
	}
	after := omnethdb.ExportSnapshot{
		SpaceID:     "repo:company/app",
		GeneratedAt: time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC),
		LiveMemories: []omnethdb.Memory{
			{ID: "mem-after", SpaceID: "repo:company/app", Content: "new fact", Kind: omnethdb.KindStatic, Version: 2, IsLatest: true},
		},
		AuditEntries: []omnethdb.AuditEntry{
			{Timestamp: time.Date(2026, 3, 31, 10, 30, 0, 0, time.UTC), SpaceID: "repo:company/app", Operation: "remember", MemoryIDs: []string{"mem-before", "mem-after"}},
		},
	}

	dir := t.TempDir()
	beforePath := filepath.Join(dir, "before.json")
	afterPath := filepath.Join(dir, "after.json")
	writeSnapshotFile(t, beforePath, before)
	writeSnapshotFile(t, afterPath, after)

	path := filepath.Join(t.TempDir(), "memory.db")
	store, err := omnethdb.Open(path)
	if err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := httptest.NewServer(NewHandler(store, &omnethdb.RuntimeConfig{}))
	defer server.Close()

	var diff omnethdb.ExportDiff
	doJSON(t, http.MethodGet, server.URL+"/v1/export/compare?before_file="+beforePath+"&after_file="+afterPath, nil, http.StatusOK, &diff)
	if len(diff.AddedLiveMemories) != 1 || diff.AddedLiveMemories[0].ID != "mem-after" {
		t.Fatalf("unexpected compare diff added memories: %#v", diff)
	}
	if len(diff.RemovedLiveMemories) != 1 || diff.RemovedLiveMemories[0].ID != "mem-before" {
		t.Fatalf("unexpected compare diff removed memories: %#v", diff)
	}
}

func doJSON(t *testing.T, method string, url string, body any, wantStatus int, out any) {
	t.Helper()

	var reqBody *bytes.Reader
	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Marshal returned unexpected error: %v", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("NewRequest returned unexpected error: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do returned unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		t.Fatalf("unexpected status %d want %d payload=%#v", resp.StatusCode, wantStatus, payload)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("Decode returned unexpected error: %v", err)
		}
	}
}

func doText(t *testing.T, method string, url string, wantStatus int) string {
	t.Helper()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("NewRequest returned unexpected error: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do returned unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d want %d body=%s", resp.StatusCode, wantStatus, string(raw))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll returned unexpected error: %v", err)
	}
	return string(raw)
}

func writeSnapshotFile(t *testing.T, path string, snapshot omnethdb.ExportSnapshot) {
	t.Helper()
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal returned unexpected error: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}
}
