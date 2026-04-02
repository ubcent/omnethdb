package bolt

import (
	"strings"
	"testing"
	"time"

	"omnethdb/internal/memory"
)

func TestExportSnapshotAndRenderersExplainCurrentAndHistoricalState(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	v1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use cursor pagination",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("v1 Remember returned unexpected error: %v", err)
	}

	v2, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use signed cursor pagination",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
		Relations:  memory.MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("v2 Remember returned unexpected error: %v", err)
	}

	e1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy failed because migration skipped smoke test",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:scout", Kind: memory.ActorAgent},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("episodic Remember returned unexpected error: %v", err)
	}

	d1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments migrations require a smoke test before deploy",
		Kind:       memory.KindDerived,
		Actor:      memory.Actor{ID: "agent:scout", Kind: memory.ActorAgent},
		Confidence: 0.85,
		SourceIDs:  []string{v2.ID, e1.ID},
		Rationale:  "static pagination change and deploy incident both point to missing validation before rollout",
	})
	if err != nil {
		t.Fatalf("derived Remember returned unexpected error: %v", err)
	}

	if err := store.Forget(e1.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "incident folded into postmortem"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	snapshot, err := store.ExportSnapshot(memory.ExportRequest{SpaceID: "repo:company/app"})
	if err != nil {
		t.Fatalf("ExportSnapshot returned unexpected error: %v", err)
	}

	if snapshot.SpaceID != "repo:company/app" {
		t.Fatalf("unexpected snapshot space: %#v", snapshot)
	}
	if len(snapshot.LiveMemories) != 2 {
		t.Fatalf("expected v2 and d1 to remain live, got %#v", snapshot.LiveMemories)
	}
	if snapshot.LiveMemories[0].ID != v2.ID || snapshot.LiveMemories[1].ID != d1.ID {
		t.Fatalf("unexpected live ordering: %#v", snapshot.LiveMemories)
	}
	if !snapshot.LiveMemories[1].HasOrphanedSources {
		t.Fatalf("expected derived memory to expose orphaned-source state, got %#v", snapshot.LiveMemories[1])
	}
	if len(snapshot.Lineages) != 3 {
		t.Fatalf("expected static, episodic, and derived lineages, got %#v", snapshot.Lineages)
	}

	var staticLineage memory.ExportLineage
	foundStatic := false
	for _, lineage := range snapshot.Lineages {
		if lineage.RootID == v1.ID {
			staticLineage = lineage
			foundStatic = true
			break
		}
	}
	if !foundStatic {
		t.Fatalf("expected static lineage rooted at %s, got %#v", v1.ID, snapshot.Lineages)
	}
	if len(staticLineage.Memories) != 2 || staticLineage.Memories[0].ID != v1.ID || staticLineage.Memories[1].ID != v2.ID {
		t.Fatalf("unexpected static lineage history: %#v", staticLineage)
	}

	if !containsEdge(snapshot.Relations, d1.ID, v2.ID, memory.RelationDerives) || !containsEdge(snapshot.Relations, d1.ID, e1.ID, memory.RelationDerives) {
		t.Fatalf("expected derive edges in snapshot, got %#v", snapshot.Relations)
	}
	if !containsEdge(snapshot.Relations, v2.ID, v1.ID, memory.RelationUpdates) {
		t.Fatalf("expected update edge in snapshot, got %#v", snapshot.Relations)
	}
	if len(snapshot.AuditEntries) != 5 {
		t.Fatalf("expected remember/update/remember/remember/forget audit entries, got %#v", snapshot.AuditEntries)
	}

	md, err := store.RenderExportSummaryMarkdown(memory.ExportRequest{SpaceID: "repo:company/app"})
	if err != nil {
		t.Fatalf("RenderExportSummaryMarkdown returned unexpected error: %v", err)
	}
	if !strings.Contains(md, "## Live Corpus") || !strings.Contains(md, "## Lineages") || !strings.Contains(md, "## Audit Timeline") {
		t.Fatalf("expected markdown sections, got:\n%s", md)
	}
	if !strings.Contains(md, "orphaned: true") || !strings.Contains(md, "Root "+v1.ID) || !strings.Contains(md, d1.ID+" derives "+e1.ID) {
		t.Fatalf("expected markdown to explain orphaned derived state, got:\n%s", md)
	}

	mermaid, err := store.RenderExportGraphMermaid(memory.ExportRequest{SpaceID: "repo:company/app"})
	if err != nil {
		t.Fatalf("RenderExportGraphMermaid returned unexpected error: %v", err)
	}
	if !strings.Contains(mermaid, "graph TD") || !strings.Contains(mermaid, "|updates|") || !strings.Contains(mermaid, "|derives|") {
		t.Fatalf("expected mermaid edges, got:\n%s", mermaid)
	}
	if !strings.Contains(mermaid, "classDef latest") || !strings.Contains(mermaid, "classDef orphaned") {
		t.Fatalf("expected mermaid classes, got:\n%s", mermaid)
	}
}

func containsEdge(edges []memory.ExportEdge, fromID string, toID string, relation memory.RelationType) bool {
	for _, edge := range edges {
		if edge.FromID == fromID && edge.ToID == toID && edge.Relation == relation {
			return true
		}
	}
	return false
}

func TestExportDiffReconstructsSnapshotChangesAcrossTimeWindow(t *testing.T) {
	t.Parallel()

	store := newRememberTestStore(t)

	v1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use cursor pagination",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
	})
	if err != nil {
		t.Fatalf("v1 Remember returned unexpected error: %v", err)
	}
	before := time.Now().UTC()
	time.Sleep(5 * time.Millisecond)

	v2, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments use signed cursor pagination",
		Kind:       memory.KindStatic,
		Actor:      memory.Actor{ID: "user:alice", Kind: memory.ActorHuman},
		Confidence: 1.0,
		Relations:  memory.MemoryRelations{Updates: []string{v1.ID}},
	})
	if err != nil {
		t.Fatalf("v2 Remember returned unexpected error: %v", err)
	}
	e1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "deploy incident showed missing smoke test",
		Kind:       memory.KindEpisodic,
		Actor:      memory.Actor{ID: "agent:scout", Kind: memory.ActorAgent},
		Confidence: 0.7,
	})
	if err != nil {
		t.Fatalf("e1 Remember returned unexpected error: %v", err)
	}
	d1, err := store.Remember(memory.MemoryInput{
		SpaceID:    "repo:company/app",
		Content:    "payments migrations require smoke test",
		Kind:       memory.KindDerived,
		Actor:      memory.Actor{ID: "agent:scout", Kind: memory.ActorAgent},
		Confidence: 0.8,
		SourceIDs:  []string{v2.ID, e1.ID},
		Rationale:  "update plus incident together imply mandatory validation",
	})
	if err != nil {
		t.Fatalf("d1 Remember returned unexpected error: %v", err)
	}
	if err := store.Forget(e1.ID, memory.Actor{ID: "user:alice", Kind: memory.ActorHuman}, "rolled into postmortem"); err != nil {
		t.Fatalf("Forget returned unexpected error: %v", err)
	}

	diff, err := store.ExportDiff(memory.ExportDiffRequest{
		SpaceID: "repo:company/app",
		Since:   before,
	})
	if err != nil {
		t.Fatalf("ExportDiff returned unexpected error: %v", err)
	}

	if len(diff.Before.LiveMemories) != 1 || diff.Before.LiveMemories[0].ID != v1.ID {
		t.Fatalf("expected diff before-snapshot to expose only v1, got %#v", diff.Before.LiveMemories)
	}
	if len(diff.After.LiveMemories) != 2 || diff.After.LiveMemories[0].ID != v2.ID || diff.After.LiveMemories[1].ID != d1.ID {
		t.Fatalf("expected diff after-snapshot to expose v2 and d1, got %#v", diff.After.LiveMemories)
	}
	if len(diff.AddedLiveMemories) != 2 {
		t.Fatalf("expected two added live memories, got %#v", diff.AddedLiveMemories)
	}
	if len(diff.RemovedLiveMemories) != 1 || diff.RemovedLiveMemories[0].ID != v1.ID {
		t.Fatalf("expected v1 to leave live corpus, got %#v", diff.RemovedLiveMemories)
	}
	if len(diff.NewOrphanedDerives) != 1 || diff.NewOrphanedDerives[0].ID != d1.ID {
		t.Fatalf("expected orphaned derived diff, got %#v", diff.NewOrphanedDerives)
	}
	if len(diff.AddedAuditEntries) != 4 {
		t.Fatalf("expected update, episodic, derived, and forget audit entries, got %#v", diff.AddedAuditEntries)
	}
	if len(diff.ChangedLineages) == 0 {
		t.Fatalf("expected changed lineage entries, got %#v", diff.ChangedLineages)
	}
}
