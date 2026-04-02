package bolt

import (
	"cmp"
	"encoding/json"
	"fmt"
	"omnethdb/internal/memory"
	"sort"
	"strings"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func (s *Store) ExportSnapshot(req memory.ExportRequest) (*memory.ExportSnapshot, error) {
	return s.exportSnapshotAt(req, time.Now().UTC())
}

func (s *Store) ExportDiff(req memory.ExportDiffRequest) (*memory.ExportDiff, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(req.SpaceID); err != nil {
		return nil, err
	}
	if req.Since.IsZero() {
		return nil, memory.ErrInvalidContent
	}

	until := req.Until.UTC()
	if until.IsZero() {
		until = time.Now().UTC()
	}
	since := req.Since.UTC()
	if until.Before(since) {
		return nil, memory.ErrInvalidContent
	}

	before, err := s.exportSnapshotAt(memory.ExportRequest{SpaceID: req.SpaceID}, since)
	if err != nil {
		return nil, err
	}
	after, err := s.exportSnapshotAt(memory.ExportRequest{SpaceID: req.SpaceID}, until)
	if err != nil {
		return nil, err
	}

	diff := &memory.ExportDiff{
		SpaceID: req.SpaceID,
	}
	computed := memory.CompareExportSnapshots(*before, *after)
	diff = &computed
	return diff, nil
}

func (s *Store) exportSnapshotAt(req memory.ExportRequest, asOf time.Time) (*memory.ExportSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, memory.ErrStoreClosed
	}
	if err := memory.ValidateSpaceID(req.SpaceID); err != nil {
		return nil, err
	}

	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOf = asOf.UTC()

	snapshot := &memory.ExportSnapshot{
		SpaceID:     req.SpaceID,
		GeneratedAt: asOf,
	}

	err := s.db.View(func(tx *bbolt.Tx) error {
		memories, err := loadAllSpaceMemories(tx, req.SpaceID)
		if err != nil {
			return err
		}
		audit, err := loadAuditEntries(tx, req.SpaceID, time.Time{})
		if err != nil {
			return err
		}

		transformed := projectMemoriesAsOf(memories, audit, asOf)
		snapshot.LiveMemories = collectLiveMemories(transformed, asOf)
		snapshot.Lineages = buildExportLineages(transformed)
		snapshot.Relations = collectExportEdges(transformed)
		snapshot.AuditEntries = filterAuditEntriesAsOf(audit, asOf)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

func (s *Store) RenderExportSummaryMarkdown(req memory.ExportRequest) (string, error) {
	snapshot, err := s.ExportSnapshot(req)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Space: %s\n\n", snapshot.SpaceID)

	b.WriteString("## Live Corpus\n")
	if len(snapshot.LiveMemories) == 0 {
		b.WriteString("- none\n\n")
	} else {
		for _, mem := range snapshot.LiveMemories {
			fmt.Fprintf(&b, "- [%s] %s\n", renderMemoryKind(mem.Kind), mem.Content)
			fmt.Fprintf(&b, "  id: %s\n", mem.ID)
			fmt.Fprintf(&b, "  version: %d\n", mem.Version)
			fmt.Fprintf(&b, "  latest: %t\n", mem.IsLatest)
			fmt.Fprintf(&b, "  forgotten: %t\n", mem.IsForgotten)
			fmt.Fprintf(&b, "  orphaned: %t\n", mem.HasOrphanedSources)
			fmt.Fprintf(&b, "  actor: %s\n", mem.Actor.ID)
			fmt.Fprintf(&b, "  confidence: %.2f\n", mem.Confidence)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Lineages\n")
	if len(snapshot.Lineages) == 0 {
		b.WriteString("- none\n\n")
	} else {
		for _, lineage := range snapshot.Lineages {
			fmt.Fprintf(&b, "### Root %s\n", lineage.RootID)
			for _, mem := range lineage.Memories {
				fmt.Fprintf(
					&b,
					"- v%d %s latest=%t forgotten=%t orphaned=%t\n  %s\n",
					mem.Version,
					mem.ID,
					mem.IsLatest,
					mem.IsForgotten,
					mem.HasOrphanedSources,
					mem.Content,
				)
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("## Relations\n")
	if len(snapshot.Relations) == 0 {
		b.WriteString("- none\n\n")
	} else {
		for _, edge := range snapshot.Relations {
			fmt.Fprintf(&b, "- %s %s %s\n", edge.FromID, edge.Relation, edge.ToID)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Audit Timeline\n")
	if len(snapshot.AuditEntries) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, entry := range snapshot.AuditEntries {
			fmt.Fprintf(&b, "- %s %s %s", entry.Timestamp.Format(time.RFC3339), entry.Operation, strings.Join(entry.MemoryIDs, " -> "))
			if strings.TrimSpace(entry.Reason) != "" {
				fmt.Fprintf(&b, " reason=%q", entry.Reason)
			}
			b.WriteString("\n")
		}
	}

	return b.String(), nil
}

func (s *Store) RenderExportGraphMermaid(req memory.ExportRequest) (string, error) {
	snapshot, err := s.ExportSnapshot(req)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("graph TD\n")

	nodes := make(map[string]memory.Memory)
	for _, lineage := range snapshot.Lineages {
		for _, mem := range lineage.Memories {
			nodes[mem.ID] = mem
		}
	}

	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		mem := nodes[id]
		fmt.Fprintf(&b, "  %s[%q]\n", mermaidNodeID(id), renderMermaidLabel(mem))
	}
	for _, edge := range snapshot.Relations {
		fmt.Fprintf(&b, "  %s -->|%s| %s\n", mermaidNodeID(edge.FromID), edge.Relation, mermaidNodeID(edge.ToID))
	}

	latestIDs := make([]string, 0)
	for _, mem := range nodes {
		if mem.IsLatest && !mem.IsForgotten {
			latestIDs = append(latestIDs, mermaidNodeID(mem.ID))
		}
	}
	sort.Strings(latestIDs)
	if len(latestIDs) > 0 {
		fmt.Fprintf(&b, "  class %s latest;\n", strings.Join(latestIDs, ","))
	}

	forgottenIDs := make([]string, 0)
	orphanedIDs := make([]string, 0)
	for _, mem := range nodes {
		if mem.IsForgotten {
			forgottenIDs = append(forgottenIDs, mermaidNodeID(mem.ID))
		}
		if mem.HasOrphanedSources {
			orphanedIDs = append(orphanedIDs, mermaidNodeID(mem.ID))
		}
	}
	sort.Strings(forgottenIDs)
	sort.Strings(orphanedIDs)
	if len(forgottenIDs) > 0 {
		fmt.Fprintf(&b, "  class %s forgotten;\n", strings.Join(forgottenIDs, ","))
	}
	if len(orphanedIDs) > 0 {
		fmt.Fprintf(&b, "  class %s orphaned;\n", strings.Join(orphanedIDs, ","))
	}
	b.WriteString("  classDef latest fill:#d9f99d,stroke:#3f6212;\n")
	b.WriteString("  classDef forgotten fill:#fecaca,stroke:#991b1b;\n")
	b.WriteString("  classDef orphaned fill:#fed7aa,stroke:#9a3412;\n")

	return b.String(), nil
}

func loadAllSpaceMemories(tx *bbolt.Tx, spaceID string) ([]memory.Memory, error) {
	memories := make([]memory.Memory, 0)
	b := tx.Bucket(bucketMemories)
	c := b.Cursor()
	for _, raw := c.First(); raw != nil; _, raw = c.Next() {
		var mem memory.Memory
		if err := json.Unmarshal(raw, &mem); err != nil {
			return nil, err
		}
		if mem.SpaceID != spaceID {
			continue
		}
		memories = append(memories, mem)
	}

	sort.SliceStable(memories, func(i, j int) bool {
		if !memories[i].CreatedAt.Equal(memories[j].CreatedAt) {
			return memories[i].CreatedAt.Before(memories[j].CreatedAt)
		}
		return cmp.Less(memories[i].ID, memories[j].ID)
	})

	return memories, nil
}

func collectLiveMemories(memories []memory.Memory, asOf time.Time) []memory.Memory {
	live := make([]memory.Memory, 0)
	for _, mem := range memories {
		mem := mem
		if isRecallEligible(&mem, asOf, nil, false) {
			live = append(live, mem)
		}
	}
	return live
}

func buildExportLineages(memories []memory.Memory) []memory.ExportLineage {
	byRoot := make(map[string][]memory.Memory)
	for _, mem := range memories {
		rootID := mem.ID
		if mem.RootID != nil {
			rootID = *mem.RootID
		}
		byRoot[rootID] = append(byRoot[rootID], mem)
	}

	rootIDs := make([]string, 0, len(byRoot))
	for rootID := range byRoot {
		rootIDs = append(rootIDs, rootID)
	}
	sort.Strings(rootIDs)

	lineages := make([]memory.ExportLineage, 0, len(rootIDs))
	for _, rootID := range rootIDs {
		history := byRoot[rootID]
		sort.SliceStable(history, func(i, j int) bool {
			if history[i].Version != history[j].Version {
				return history[i].Version < history[j].Version
			}
			if !history[i].CreatedAt.Equal(history[j].CreatedAt) {
				return history[i].CreatedAt.Before(history[j].CreatedAt)
			}
			return cmp.Less(history[i].ID, history[j].ID)
		})
		hasLatest := false
		for _, mem := range history {
			if mem.IsLatest {
				hasLatest = true
				break
			}
		}
		lineages = append(lineages, memory.ExportLineage{
			RootID:    rootID,
			Memories:  history,
			HasLatest: hasLatest,
		})
	}

	return lineages
}

func collectExportEdges(memories []memory.Memory) []memory.ExportEdge {
	edges := make([]memory.ExportEdge, 0)
	for _, mem := range memories {
		for _, target := range mem.Relations.Updates {
			edges = append(edges, memory.ExportEdge{FromID: mem.ID, ToID: target, Relation: memory.RelationUpdates})
		}
		for _, target := range mem.Relations.Extends {
			edges = append(edges, memory.ExportEdge{FromID: mem.ID, ToID: target, Relation: memory.RelationExtends})
		}
		for _, target := range mem.Relations.Derives {
			edges = append(edges, memory.ExportEdge{FromID: mem.ID, ToID: target, Relation: memory.RelationDerives})
		}
	}

	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].FromID != edges[j].FromID {
			return edges[i].FromID < edges[j].FromID
		}
		if edges[i].Relation != edges[j].Relation {
			return edges[i].Relation < edges[j].Relation
		}
		return edges[i].ToID < edges[j].ToID
	})

	return edges
}

func renderMemoryKind(kind memory.MemoryKind) string {
	switch kind {
	case memory.KindEpisodic:
		return "episodic"
	case memory.KindStatic:
		return "static"
	case memory.KindDerived:
		return "derived"
	default:
		return "unknown"
	}
}

func mermaidNodeID(id string) string {
	return "mem_" + strings.NewReplacer(":", "_", "-", "_").Replace(id)
}

func renderMermaidLabel(mem memory.Memory) string {
	content := strings.TrimSpace(mem.Content)
	if len(content) > 48 {
		content = content[:45] + "..."
	}
	return fmt.Sprintf("%s v%d %s", mem.ID, mem.Version, content)
}

func projectMemoriesAsOf(memories []memory.Memory, audit []memory.AuditEntry, asOf time.Time) []memory.Memory {
	forgetAt := make(map[string]time.Time)
	for _, entry := range audit {
		if entry.Operation != "forget" || entry.Timestamp.After(asOf) {
			continue
		}
		for _, id := range entry.MemoryIDs {
			if previous, ok := forgetAt[id]; !ok || entry.Timestamp.Before(previous) {
				forgetAt[id] = entry.Timestamp
			}
		}
	}

	projected := make([]memory.Memory, 0, len(memories))
	highestVersionByRoot := make(map[string]int)
	highestIDByRoot := make(map[string]string)
	highestCreatedAtByRoot := make(map[string]time.Time)
	for _, mem := range memories {
		if mem.CreatedAt.After(asOf) {
			continue
		}
		rootID := mem.ID
		if mem.RootID != nil {
			rootID = *mem.RootID
		}
		version, ok := highestVersionByRoot[rootID]
		if !ok || mem.Version > version || (mem.Version == version && mem.CreatedAt.After(highestCreatedAtByRoot[rootID])) {
			highestVersionByRoot[rootID] = mem.Version
			highestIDByRoot[rootID] = mem.ID
			highestCreatedAtByRoot[rootID] = mem.CreatedAt
		}
		projected = append(projected, mem)
	}

	for i := range projected {
		mem := &projected[i]
		_, forgotten := forgetAt[mem.ID]
		mem.IsForgotten = forgotten
		mem.HasOrphanedSources = false
		for _, sourceID := range mem.SourceIDs {
			if _, ok := forgetAt[sourceID]; ok {
				mem.HasOrphanedSources = true
				break
			}
		}
		rootID := mem.ID
		if mem.RootID != nil {
			rootID = *mem.RootID
		}
		mem.IsLatest = mem.ID == highestIDByRoot[rootID] && !mem.IsForgotten
	}

	return projected
}

func filterAuditEntriesAsOf(entries []memory.AuditEntry, asOf time.Time) []memory.AuditEntry {
	filtered := make([]memory.AuditEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Timestamp.After(asOf) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
