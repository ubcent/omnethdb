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

func TestResolveLayoutUsesStableWorkspacePaths(t *testing.T) {
	t.Parallel()

	layout, err := ResolveLayout("/tmp/omnethdb-workspace")
	if err != nil {
		t.Fatalf("ResolveLayout returned unexpected error: %v", err)
	}
	if layout.ConfigPath != filepath.Join(layout.RootDir, "config.toml") {
		t.Fatalf("unexpected config path: %q", layout.ConfigPath)
	}
	if layout.DataDir != filepath.Join(layout.RootDir, "data") {
		t.Fatalf("unexpected data dir: %q", layout.DataDir)
	}
	if layout.DataPath != filepath.Join(layout.RootDir, "data", "memory.db") {
		t.Fatalf("unexpected data path: %q", layout.DataPath)
	}
}

func TestLoadConfigAppliesPerSpaceOverridesToSpaceInit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	layout, err := ResolveLayout(root)
	if err != nil {
		t.Fatalf("ResolveLayout returned unexpected error: %v", err)
	}

	configText := `
[spaces."repo:company/app"]
default_weight = 0.25
half_life_days = 7
max_static_memories = 1
max_episodic_memories = 9
profile_max_static = 3
profile_max_episodic = 2
human_trust = 0.95
system_trust = 1.0
default_agent_trust = 0.4

[spaces."repo:company/app".static_writers]
allow_human = true
allow_system = true
allow_all_agents = false
allowed_agent_ids = ["agent:claude"]
min_trust_level = 0.3

[spaces."repo:company/app".embedder]
model_id = "openai/text-embedding-3-small"
dimensions = 8
`
	if err := os.WriteFile(layout.ConfigPath, []byte(configText), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	cfg, err := LoadConfig(layout.ConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig returned unexpected error: %v", err)
	}
	settings, ok := cfg.SpaceSettings("repo:company/app")
	if !ok {
		t.Fatal("expected per-space settings to be loaded")
	}
	if settings.Embedder.ModelID != "openai/text-embedding-3-small" || settings.Embedder.Dimensions != 8 {
		t.Fatalf("unexpected embedder settings: %+v", settings.Embedder)
	}

	base := storebolt.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   policy.DefaultSpaceWritePolicy(),
	}
	init := cfg.SpaceInit("repo:company/app", base)
	if init.DefaultWeight != 0.25 || init.HalfLifeDays != 7 {
		t.Fatalf("expected overrides to apply, got %+v", init)
	}
	if init.WritePolicy.MaxStaticMemories != 1 || init.WritePolicy.ProfileMaxStatic != 3 {
		t.Fatalf("expected policy overrides to apply, got %+v", init.WritePolicy)
	}
	if init.WritePolicy.HumanTrust != 0.95 || init.WritePolicy.DefaultAgentTrust != 0.4 {
		t.Fatalf("expected trust overrides to apply, got %+v", init.WritePolicy)
	}
	if init.WritePolicy.StaticWriters.AllowAllAgents {
		t.Fatalf("expected explicit static writer override, got %+v", init.WritePolicy.StaticWriters)
	}
	if len(init.WritePolicy.StaticWriters.AllowedAgentIDs) != 1 || init.WritePolicy.StaticWriters.AllowedAgentIDs[0] != "agent:claude" {
		t.Fatalf("expected allowed static writer override, got %+v", init.WritePolicy.StaticWriters)
	}
	if init.WritePolicy.StaticWriters.MinTrustLevel != 0.3 {
		t.Fatalf("expected static writer trust override, got %+v", init.WritePolicy.StaticWriters)
	}
}

func TestOpenWorkspaceCreatesStableDataLayout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, layout, err := OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	if _, err := os.Stat(layout.DataDir); err != nil {
		t.Fatalf("expected data dir to exist: %v", err)
	}
	if layout.DataPath != filepath.Join(layout.RootDir, "data", "memory.db") {
		t.Fatalf("unexpected data path: %q", layout.DataPath)
	}
}

func TestRuntimeConfigCanDrivePersistedSpaceBehavior(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	layout, err := ResolveLayout(root)
	if err != nil {
		t.Fatalf("ResolveLayout returned unexpected error: %v", err)
	}

	configText := `
[spaces."repo:company/app"]
default_weight = 0.5
half_life_days = 5
max_static_memories = 1
`
	if err := os.WriteFile(layout.ConfigPath, []byte(configText), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	cfg, err := LoadConfig(layout.ConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig returned unexpected error: %v", err)
	}
	store, _, err := OpenWorkspace(root)
	if err != nil {
		t.Fatalf("OpenWorkspace returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	embedder := testEmbedder{modelID: "openai/text-embedding-3-small", dimensions: 8}
	init := cfg.SpaceInit("repo:company/app", storebolt.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   policy.DefaultSpaceWritePolicy(),
	})
	if _, err := store.EnsureSpace("repo:company/app", embedder, init); err != nil {
		t.Fatalf("EnsureSpace returned unexpected error: %v", err)
	}

	persisted, err := store.GetSpaceConfig("repo:company/app")
	if err != nil {
		t.Fatalf("GetSpaceConfig returned unexpected error: %v", err)
	}
	if persisted.DefaultWeight != 0.5 || persisted.HalfLifeDays != 5 {
		t.Fatalf("expected persisted config overrides, got %+v", *persisted)
	}

	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "first fact",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	}); err != nil {
		t.Fatalf("first Remember returned unexpected error: %v", err)
	}

	if _, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "second fact",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	}); err == nil {
		t.Fatal("expected config-driven static corpus limit to be enforced")
	}
}

func TestReconcileSpaceConfigReportsApplyableWriterPolicyChange(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Spaces: map[string]SpaceSettings{
			"repo:company/app": {
				MaxStaticMemories: intPtr(5),
				StaticWriters: RuntimeWritersPolicy{
					AllowAllAgents:  boolPtr(false),
					AllowedAgentIDs: []string{"claude-sonnet-4-6"},
				},
			},
		},
	}

	persisted := memory.SpaceConfig{
		EmbeddingModelID: "builtin/hash-embedder-v1",
		Dimension:        256,
		DefaultWeight:    1,
		HalfLifeDays:     30,
		WritePolicy:      policy.DefaultSpaceWritePolicy(),
	}

	diff := cfg.ReconcileSpaceConfig("repo:company/app", persisted)
	if !diff.Applyable {
		t.Fatalf("expected diff to be applyable, got %+v", diff)
	}
	if diff.Desired.WritePolicy.MaxStaticMemories != 5 {
		t.Fatalf("expected desired max static override, got %+v", diff.Desired.WritePolicy)
	}
	if len(diff.Desired.WritePolicy.StaticWriters.AllowedAgentIDs) != 1 || diff.Desired.WritePolicy.StaticWriters.AllowedAgentIDs[0] != "claude-sonnet-4-6" {
		t.Fatalf("expected static writer override, got %+v", diff.Desired.WritePolicy.StaticWriters)
	}
	if len(diff.Changes) == 0 {
		t.Fatal("expected reconcile to report changes")
	}
}

func TestReconcileSpaceConfigRejectsEmbedderDriftForApply(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Spaces: map[string]SpaceSettings{
			"repo:company/app": {
				Embedder: RuntimeEmbedderConfig{
					ModelID:    "openai/text-embedding-3-small",
					Dimensions: 1536,
				},
			},
		},
	}

	persisted := memory.SpaceConfig{
		EmbeddingModelID: "builtin/hash-embedder-v1",
		Dimension:        256,
		DefaultWeight:    1,
		HalfLifeDays:     30,
		WritePolicy:      policy.DefaultSpaceWritePolicy(),
	}

	diff := cfg.ReconcileSpaceConfig("repo:company/app", persisted)
	if diff.Applyable {
		t.Fatalf("expected embedder drift to block apply, got %+v", diff)
	}
	if len(diff.Errors) == 0 {
		t.Fatal("expected reconcile errors for embedder drift")
	}
	foundBlocked := false
	for _, change := range diff.Changes {
		if change.Field == "embedding_model_id" && !change.Applyable {
			foundBlocked = true
			break
		}
	}
	if !foundBlocked {
		t.Fatalf("expected blocked embedder change, got %+v", diff.Changes)
	}
}

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }
