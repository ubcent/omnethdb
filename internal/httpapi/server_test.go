package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	omnethdb "omnethdb"
)

type testEmbedder struct {
	modelID    string
	dimensions int
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
