package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	storebolt "omnethdb/internal/store/bolt"
)

type restoreEmbedder struct {
	modelID    string
	dimensions int
}

func (e restoreEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	out := make([]float32, e.dimensions)
	for i := range out {
		out[i] = float32((len(text)%13)+i+1) / 50
	}
	return out, nil
}

func (e restoreEmbedder) Dimensions() int { return e.dimensions }
func (e restoreEmbedder) ModelID() string { return e.modelID }

func TestWorkspaceBackupRestoreKeepsDataAndConfigUsable(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	sourceLayout, err := ResolveLayout(sourceRoot)
	if err != nil {
		t.Fatalf("ResolveLayout returned unexpected error: %v", err)
	}

	configText := `
[spaces."repo:company/app"]
default_weight = 0.75
half_life_days = 10

[spaces."repo:company/app".embedder]
model_id = "openai/text-embedding-3-small"
dimensions = 8
`
	if err := os.WriteFile(sourceLayout.ConfigPath, []byte(configText), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	cfg, err := LoadConfig(sourceLayout.ConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig returned unexpected error: %v", err)
	}
	store, _, err := OpenWorkspace(sourceRoot)
	if err != nil {
		t.Fatalf("OpenWorkspace returned unexpected error: %v", err)
	}

	embedder := restoreEmbedder{modelID: "openai/text-embedding-3-small", dimensions: 8}
	init := cfg.SpaceInit("repo:company/app", storebolt.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   policy.DefaultSpaceWritePolicy(),
	})
	if _, err := store.EnsureSpace("repo:company/app", embedder, init); err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}
	mem, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "restorable fact",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("Remember returned unexpected error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	restoreRoot := t.TempDir()
	restoreLayout, err := ResolveLayout(restoreRoot)
	if err != nil {
		t.Fatalf("ResolveLayout returned unexpected error: %v", err)
	}
	if err := os.MkdirAll(restoreLayout.DataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	copyFile(t, sourceLayout.ConfigPath, restoreLayout.ConfigPath)
	copyFile(t, sourceLayout.DataPath, restoreLayout.DataPath)

	restoredCfg, err := LoadConfig(restoreLayout.ConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig on restored workspace returned unexpected error: %v", err)
	}
	restoredStore, _, err := OpenWorkspace(restoreRoot)
	if err != nil {
		t.Fatalf("OpenWorkspace on restored workspace returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = restoredStore.Close() })

	restoredInit := restoredCfg.SpaceInit("repo:company/app", storebolt.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   policy.DefaultSpaceWritePolicy(),
	})
	if _, err := restoredStore.EnsureSpace("repo:company/app", embedder, restoredInit); err != nil {
		t.Fatalf("EnsureSpace on restored store returned unexpected error: %v", err)
	}

	results, err := restoredStore.Recall(memory.RecallRequest{
		SpaceIDs: []string{"repo:company/app"},
		Query:    "restorable",
		TopK:     5,
	})
	if err != nil {
		t.Fatalf("Recall returned unexpected error after restore: %v", err)
	}
	if len(results) != 1 || results[0].ID != mem.ID {
		t.Fatalf("expected restored memory to be recallable, got %#v", results)
	}

	persisted, err := restoredStore.GetSpaceConfig("repo:company/app")
	if err != nil {
		t.Fatalf("GetSpaceConfig returned unexpected error after restore: %v", err)
	}
	if persisted.DefaultWeight != 0.75 || persisted.HalfLifeDays != 10 {
		t.Fatalf("expected restored persisted config to remain coherent, got %+v", *persisted)
	}
}

func copyFile(t *testing.T, from string, to string) {
	t.Helper()

	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned unexpected error: %v", from, err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) returned unexpected error: %v", filepath.Dir(to), err)
	}
	if err := os.WriteFile(to, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) returned unexpected error: %v", to, err)
	}
}
