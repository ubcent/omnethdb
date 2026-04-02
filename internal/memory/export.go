package memory

import (
	"sort"
	"strings"
	"time"
)

type ExportFormat string

const (
	ExportFormatSnapshotJSON ExportFormat = "snapshot-json"
	ExportFormatSummaryMD    ExportFormat = "summary-md"
	ExportFormatGraphMermaid ExportFormat = "graph-mermaid"
)

type ExportRequest struct {
	SpaceID string
}

type ExportDiffRequest struct {
	SpaceID string
	Since   time.Time
	Until   time.Time
}

type ExportSnapshot struct {
	SpaceID      string          `json:"space_id"`
	GeneratedAt  time.Time       `json:"generated_at"`
	LiveMemories []Memory        `json:"live_memories"`
	Lineages     []ExportLineage `json:"lineages"`
	Relations    []ExportEdge    `json:"relations"`
	AuditEntries []AuditEntry    `json:"audit_entries"`
}

type ExportLineage struct {
	RootID    string   `json:"root_id"`
	Memories  []Memory `json:"memories"`
	HasLatest bool     `json:"has_latest"`
}

type ExportEdge struct {
	FromID   string       `json:"from_id"`
	ToID     string       `json:"to_id"`
	Relation RelationType `json:"relation"`
}

type ExportDiff struct {
	SpaceID             string              `json:"space_id"`
	Since               time.Time           `json:"since"`
	Until               time.Time           `json:"until"`
	Before              ExportSnapshot      `json:"before"`
	After               ExportSnapshot      `json:"after"`
	AddedLiveMemories   []Memory            `json:"added_live_memories"`
	RemovedLiveMemories []Memory            `json:"removed_live_memories"`
	NewOrphanedDerives  []Memory            `json:"new_orphaned_derives"`
	AddedAuditEntries   []AuditEntry        `json:"added_audit_entries"`
	ChangedLineages     []ExportLineageDiff `json:"changed_lineages"`
}

type ExportLineageDiff struct {
	RootID         string  `json:"root_id"`
	BeforeLatestID *string `json:"before_latest_id"`
	AfterLatestID  *string `json:"after_latest_id"`
	BeforeVersions int     `json:"before_versions"`
	AfterVersions  int     `json:"after_versions"`
}

func CompareExportSnapshots(before ExportSnapshot, after ExportSnapshot) ExportDiff {
	diff := ExportDiff{
		SpaceID: before.SpaceID,
		Since:   before.GeneratedAt,
		Until:   after.GeneratedAt,
		Before:  before,
		After:   after,
	}
	if diff.SpaceID == "" {
		diff.SpaceID = after.SpaceID
	}
	diff.AddedLiveMemories, diff.RemovedLiveMemories = compareLiveMemories(before.LiveMemories, after.LiveMemories)
	diff.NewOrphanedDerives = compareNewOrphanedDerives(before.LiveMemories, after.LiveMemories)
	diff.AddedAuditEntries = compareAddedAuditEntries(before.AuditEntries, after.AuditEntries)
	diff.ChangedLineages = compareLineages(before.Lineages, after.Lineages)
	return diff
}

func compareLiveMemories(before []Memory, after []Memory) ([]Memory, []Memory) {
	beforeMap := make(map[string]Memory, len(before))
	afterMap := make(map[string]Memory, len(after))
	for _, mem := range before {
		beforeMap[mem.ID] = mem
	}
	for _, mem := range after {
		afterMap[mem.ID] = mem
	}

	added := make([]Memory, 0)
	removed := make([]Memory, 0)
	for _, mem := range after {
		if _, ok := beforeMap[mem.ID]; !ok {
			added = append(added, mem)
		}
	}
	for _, mem := range before {
		if _, ok := afterMap[mem.ID]; !ok {
			removed = append(removed, mem)
		}
	}
	return added, removed
}

func compareNewOrphanedDerives(before []Memory, after []Memory) []Memory {
	beforeMap := make(map[string]Memory, len(before))
	for _, mem := range before {
		beforeMap[mem.ID] = mem
	}

	out := make([]Memory, 0)
	for _, mem := range after {
		if mem.Kind != KindDerived || !mem.HasOrphanedSources {
			continue
		}
		previous, ok := beforeMap[mem.ID]
		if !ok || !previous.HasOrphanedSources {
			out = append(out, mem)
		}
	}
	return out
}

func compareAddedAuditEntries(before []AuditEntry, after []AuditEntry) []AuditEntry {
	seen := make(map[string]struct{}, len(before))
	for _, entry := range before {
		seen[auditEntryKey(entry)] = struct{}{}
	}
	out := make([]AuditEntry, 0)
	for _, entry := range after {
		if _, ok := seen[auditEntryKey(entry)]; ok {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func compareLineages(before []ExportLineage, after []ExportLineage) []ExportLineageDiff {
	beforeMap := make(map[string]ExportLineage, len(before))
	afterMap := make(map[string]ExportLineage, len(after))
	for _, lineage := range before {
		beforeMap[lineage.RootID] = lineage
	}
	for _, lineage := range after {
		afterMap[lineage.RootID] = lineage
	}

	rootSet := make(map[string]struct{}, len(beforeMap)+len(afterMap))
	for rootID := range beforeMap {
		rootSet[rootID] = struct{}{}
	}
	for rootID := range afterMap {
		rootSet[rootID] = struct{}{}
	}
	rootIDs := make([]string, 0, len(rootSet))
	for rootID := range rootSet {
		rootIDs = append(rootIDs, rootID)
	}
	sort.Strings(rootIDs)

	out := make([]ExportLineageDiff, 0)
	for _, rootID := range rootIDs {
		beforeLineage := beforeMap[rootID]
		afterLineage := afterMap[rootID]
		beforeLatest := latestExportMemoryID(beforeLineage)
		afterLatest := latestExportMemoryID(afterLineage)
		if optionalStringEqual(beforeLatest, afterLatest) && len(beforeLineage.Memories) == len(afterLineage.Memories) {
			continue
		}
		out = append(out, ExportLineageDiff{
			RootID:         rootID,
			BeforeLatestID: beforeLatest,
			AfterLatestID:  afterLatest,
			BeforeVersions: len(beforeLineage.Memories),
			AfterVersions:  len(afterLineage.Memories),
		})
	}
	return out
}

func latestExportMemoryID(lineage ExportLineage) *string {
	for _, mem := range lineage.Memories {
		if mem.IsLatest {
			id := mem.ID
			return &id
		}
	}
	return nil
}

func optionalStringEqual(a *string, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func auditEntryKey(entry AuditEntry) string {
	return entry.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + entry.Operation + "|" + entry.Actor.ID + "|" + strings.Join(entry.MemoryIDs, ",") + "|" + entry.Reason
}
