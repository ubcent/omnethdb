package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	omnethdb "omnethdb"
	hashembedder "omnethdb/embedders/hash"
	omnethdbv1 "omnethdb/gen/omnethdb/v1"
	"omnethdb/internal/grpcapi"
	"omnethdb/internal/httpapi"

	"google.golang.org/grpc"
)

const (
	defaultWorkspace = "."
	defaultModelID   = "builtin/hash-embedder-v1"
	defaultDimension = 256
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "init":
		err = runInit(os.Args[2:])
	case "remember":
		err = runRemember(os.Args[2:])
	case "lint-remember":
		err = runLintRemember(os.Args[2:])
	case "recall":
		err = runRecall(os.Args[2:])
	case "profile":
		err = runProfile(os.Args[2:])
	case "forget":
		err = runForget(os.Args[2:])
	case "revive":
		err = runRevive(os.Args[2:])
	case "lineage":
		err = runLineage(os.Args[2:])
	case "related":
		err = runRelated(os.Args[2:])
	case "candidates":
		err = runCandidates(os.Args[2:])
	case "quality":
		err = runQuality(os.Args[2:])
	case "quality-plan":
		err = runQualityPlan(os.Args[2:])
	case "forget-batch":
		err = runForgetBatch(os.Args[2:])
	case "audit":
		err = runAudit(os.Args[2:])
	case "export":
		err = runExport(os.Args[2:])
	case "migrate":
		err = runMigrate(os.Args[2:])
	case "space":
		err = runSpace(os.Args[2:])
	case "config":
		err = runConfig(os.Args[2:])
	case "serve":
		err = runServe(os.Args[2:])
	case "serve-grpc":
		err = runServeGRPC(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("init requires --space")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	embedder := embedderForBootstrap(cfg, *spaceID)
	init := cfg.SpaceInit(*spaceID, omnethdb.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	})
	spaceCfg, err := store.EnsureSpace(*spaceID, embedder, init)
	if err != nil {
		return err
	}
	return printJSON(spaceCfg)
}

func runRemember(args []string) error {
	fs := flag.NewFlagSet("remember", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	content := fs.String("content", "", "memory content")
	kind := fs.String("kind", "episodic", "memory kind: episodic|static|derived")
	actorID := fs.String("actor-id", "user:cli", "actor id")
	actorKind := fs.String("actor-kind", "human", "actor kind: human|agent|system")
	confidence := fs.Float64("confidence", 1.0, "confidence 0..1")
	updateID := fs.String("update", "", "memory id to update")
	extendsIDs := fs.String("extends", "", "comma-separated extends target ids")
	sourceIDs := fs.String("sources", "", "comma-separated source ids for derived writes")
	rationale := fs.String("rationale", "", "derived rationale")
	ifLatestID := fs.String("if-latest-id", "", "optimistic lock latest id")
	forgetAfter := fs.String("forget-after", "", "RFC3339 timestamp")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" || strings.TrimSpace(*content) == "" {
		return errors.New("remember requires --space and --content")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, *spaceID)

	var ttl *time.Time
	if strings.TrimSpace(*forgetAfter) != "" {
		parsed, err := time.Parse(time.RFC3339, *forgetAfter)
		if err != nil {
			return fmt.Errorf("invalid --forget-after: %w", err)
		}
		ttl = &parsed
	}

	input := omnethdb.MemoryInput{
		SpaceID:     *spaceID,
		Content:     *content,
		Kind:        parseMemoryKind(*kind),
		Actor:       parseActor(*actorID, *actorKind),
		Confidence:  float32(*confidence),
		ForgetAfter: ttl,
		Relations: omnethdb.MemoryRelations{
			Extends: splitCSV(*extendsIDs),
		},
		SourceIDs: splitCSV(*sourceIDs),
		Rationale: *rationale,
	}
	if strings.TrimSpace(*updateID) != "" {
		input.Relations.Updates = []string{*updateID}
	}
	if strings.TrimSpace(*ifLatestID) != "" {
		input.IfLatestID = ifLatestID
	}

	mem, err := store.Remember(input)
	if err != nil {
		return err
	}
	return printJSON(mem)
}

func runLintRemember(args []string) error {
	fs := flag.NewFlagSet("lint-remember", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	content := fs.String("content", "", "candidate content")
	kind := fs.String("kind", "static", "memory kind: episodic|static|derived")
	actorID := fs.String("actor-id", "user:cli", "actor id")
	actorKind := fs.String("actor-kind", "human", "actor kind: human|agent|system")
	confidence := fs.Float64("confidence", 1.0, "confidence 0..1")
	updateID := fs.String("update", "", "optional update target id")
	extendsIDs := fs.String("extends", "", "comma-separated extends target ids")
	sourceIDs := fs.String("sources", "", "comma-separated source ids for derived writes")
	rationale := fs.String("rationale", "", "derived rationale")
	topK := fs.Int("top-k", 3, "top-k similar live memories")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" || strings.TrimSpace(*content) == "" {
		return errors.New("lint-remember requires --space and --content")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, *spaceID)

	input := omnethdb.MemoryInput{
		SpaceID:    *spaceID,
		Content:    *content,
		Kind:       parseMemoryKind(*kind),
		Actor:      parseActor(*actorID, *actorKind),
		Confidence: float32(*confidence),
		SourceIDs:  splitCSV(*sourceIDs),
		Rationale:  *rationale,
		Relations: omnethdb.MemoryRelations{
			Extends: splitCSV(*extendsIDs),
		},
	}
	if strings.TrimSpace(*updateID) != "" {
		input.Relations.Updates = []string{*updateID}
	}

	result, err := store.LintRemember(omnethdb.RememberLintRequest{
		Input: input,
		TopK:  *topK,
	})
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runRecall(args []string) error {
	fs := flag.NewFlagSet("recall", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaces := fs.String("spaces", "", "comma-separated space ids")
	query := fs.String("query", "", "query string")
	topK := fs.Int("top-k", 10, "top-k results")
	kinds := fs.String("kinds", "", "comma-separated kinds")
	excludeOrphaned := fs.Bool("exclude-orphaned", false, "exclude orphaned derived memories")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaces) == "" {
		return errors.New("recall requires --spaces")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()
	ensureEmbeddersForSpaces(store, cfg, splitCSV(*spaces))

	results, err := store.Recall(omnethdb.RecallRequest{
		SpaceIDs:               splitCSV(*spaces),
		Query:                  *query,
		TopK:                   *topK,
		Kinds:                  parseMemoryKinds(*kinds),
		ExcludeOrphanedDerives: *excludeOrphaned,
	})
	if err != nil {
		return err
	}
	return printJSON(results)
}

func runProfile(args []string) error {
	fs := flag.NewFlagSet("profile", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaces := fs.String("spaces", "", "comma-separated space ids")
	query := fs.String("query", "", "query string")
	staticTopK := fs.Int("static-top-k", 0, "static layer top-k")
	episodicTopK := fs.Int("episodic-top-k", 0, "episodic layer top-k")
	excludeOrphaned := fs.Bool("exclude-orphaned", false, "exclude orphaned derived memories")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaces) == "" {
		return errors.New("profile requires --spaces")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()
	ensureEmbeddersForSpaces(store, cfg, splitCSV(*spaces))

	profile, err := store.GetProfile(omnethdb.ProfileRequest{
		SpaceIDs:               splitCSV(*spaces),
		Query:                  *query,
		StaticTopK:             *staticTopK,
		EpisodicTopK:           *episodicTopK,
		ExcludeOrphanedDerives: *excludeOrphaned,
	})
	if err != nil {
		return err
	}
	return printJSON(profile)
}

func runForget(args []string) error {
	fs := flag.NewFlagSet("forget", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	id := fs.String("id", "", "memory id")
	actorID := fs.String("actor-id", "user:cli", "actor id")
	actorKind := fs.String("actor-kind", "human", "actor kind")
	reason := fs.String("reason", "", "forget reason")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" || strings.TrimSpace(*reason) == "" {
		return errors.New("forget requires --id and --reason")
	}

	store, _, _, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.Forget(*id, parseActor(*actorID, *actorKind), *reason); err != nil {
		return err
	}
	fmt.Println("ok")
	return nil
}

func runRevive(args []string) error {
	fs := flag.NewFlagSet("revive", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	rootID := fs.String("root", "", "root memory id")
	content := fs.String("content", "", "revived content")
	kind := fs.String("kind", "static", "memory kind: episodic|static")
	actorID := fs.String("actor-id", "user:cli", "actor id")
	actorKind := fs.String("actor-kind", "human", "actor kind")
	confidence := fs.Float64("confidence", 1.0, "confidence 0..1")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*rootID) == "" || strings.TrimSpace(*content) == "" {
		return errors.New("revive requires --root and --content")
	}

	store, _, _, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	mem, err := store.Revive(*rootID, omnethdb.ReviveInput{
		Content:    *content,
		Kind:       parseMemoryKind(*kind),
		Actor:      parseActor(*actorID, *actorKind),
		Confidence: float32(*confidence),
	})
	if err != nil {
		return err
	}
	return printJSON(mem)
}

func runLineage(args []string) error {
	fs := flag.NewFlagSet("lineage", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	rootID := fs.String("root", "", "root memory id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*rootID) == "" {
		return errors.New("lineage requires --root")
	}

	store, _, _, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	lineage, err := store.GetLineage(*rootID)
	if err != nil {
		return err
	}
	return printJSON(lineage)
}

func runRelated(args []string) error {
	fs := flag.NewFlagSet("related", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	id := fs.String("id", "", "memory id")
	relation := fs.String("relation", "extends", "relation type: updates|extends|derives")
	depth := fs.Int("depth", 1, "traversal depth")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("related requires --id")
	}

	store, _, _, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	related, err := store.GetRelated(*id, parseRelationType(*relation), *depth)
	if err != nil {
		return err
	}
	return printJSON(related)
}

func runCandidates(args []string) error {
	fs := flag.NewFlagSet("candidates", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	content := fs.String("content", "", "candidate query content")
	topK := fs.Int("top-k", 10, "top-k results")
	includeSuperseded := fs.Bool("include-superseded", false, "include superseded memories")
	includeForgotten := fs.Bool("include-forgotten", false, "include forgotten memories")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" || strings.TrimSpace(*content) == "" {
		return errors.New("candidates requires --space and --content")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, *spaceID)

	results, err := store.FindCandidates(omnethdb.FindCandidatesRequest{
		SpaceID:           *spaceID,
		Content:           *content,
		TopK:              *topK,
		IncludeSuperseded: *includeSuperseded,
		IncludeForgotten:  *includeForgotten,
	})
	if err != nil {
		return err
	}
	return printJSON(results)
}

func runQuality(args []string) error {
	fs := flag.NewFlagSet("quality", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	topK := fs.Int("top-k-per-memory", 5, "top similar live candidates to inspect per live static memory")
	maxDuplicateGroups := fs.Int("max-duplicate-groups", 8, "maximum duplicate groups to return")
	maxUpdatePairs := fs.Int("max-update-pairs", 12, "maximum possible update pairs to return")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("quality requires --space")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, *spaceID)

	result, err := store.GetQualityDiagnostics(omnethdb.QualityDiagnosticsRequest{
		SpaceID:            *spaceID,
		TopKPerMemory:      *topK,
		MaxDuplicateGroups: *maxDuplicateGroups,
		MaxUpdatePairs:     *maxUpdatePairs,
	})
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runQualityPlan(args []string) error {
	fs := flag.NewFlagSet("quality-plan", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	maxDuplicateActions := fs.Int("max-duplicate-actions", 8, "maximum duplicate cleanup suggestions to return")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("quality-plan requires --space")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()
	ensureEmbedderForSpace(store, cfg, *spaceID)

	result, err := store.BuildQualityCleanupPlan(omnethdb.QualityCleanupPlanRequest{
		SpaceID:             *spaceID,
		MaxDuplicateActions: *maxDuplicateActions,
	})
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runForgetBatch(args []string) error {
	fs := flag.NewFlagSet("forget-batch", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	ids := fs.String("ids", "", "comma-separated memory ids")
	actorID := fs.String("actor-id", "user:cli", "actor id")
	actorKind := fs.String("actor-kind", "human", "actor kind")
	reason := fs.String("reason", "", "forget reason")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*ids) == "" || strings.TrimSpace(*reason) == "" {
		return errors.New("forget-batch requires --ids and --reason")
	}

	store, _, _, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	items := splitCSV(*ids)
	if err := store.ForgetBatch(items, parseActor(*actorID, *actorKind), *reason); err != nil {
		return err
	}
	return printJSON(map[string]any{
		"status":     "ok",
		"memory_ids": items,
		"reason":     *reason,
	})
}

func runAudit(args []string) error {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	since := fs.String("since", "", "RFC3339 timestamp")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("audit requires --space")
	}

	store, _, _, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	var sinceTime time.Time
	if strings.TrimSpace(*since) != "" {
		parsed, err := time.Parse(time.RFC3339, *since)
		if err != nil {
			return fmt.Errorf("invalid --since: %w", err)
		}
		sinceTime = parsed
	}

	audit, err := store.GetAuditLog(*spaceID, sinceTime)
	if err != nil {
		return err
	}
	return printJSON(audit)
}

func runMigrate(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	modelID := fs.String("model-id", "", "target embedder model id")
	dimensions := fs.Int("dimensions", 0, "target embedder dimensions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("migrate requires --space")
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	embedder := embedderForMigration(store, cfg, *spaceID, *modelID, *dimensions)
	if err := store.MigrateEmbeddings(*spaceID, embedder); err != nil {
		return err
	}
	spaceCfg, err := store.GetSpaceConfig(*spaceID)
	if err != nil {
		return err
	}
	return printJSON(spaceCfg)
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	format := fs.String("format", string(omnethdb.ExportFormatSnapshotJSON), "export format: snapshot-json|summary-md|graph-mermaid")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("export requires --space")
	}

	store, _, _, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	req := omnethdb.ExportRequest{SpaceID: *spaceID}
	switch omnethdb.ExportFormat(strings.TrimSpace(*format)) {
	case omnethdb.ExportFormatSnapshotJSON:
		snapshot, err := store.ExportSnapshot(req)
		if err != nil {
			return err
		}
		return printJSON(snapshot)
	case omnethdb.ExportFormatSummaryMD:
		out, err := store.RenderExportSummaryMarkdown(req)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, out)
		return err
	case omnethdb.ExportFormatGraphMermaid:
		out, err := store.RenderExportGraphMermaid(req)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, out)
		return err
	default:
		return fmt.Errorf("unsupported --format %q", *format)
	}
}

func runSpace(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "validate-config":
			return runSpaceValidateConfig(args[1:])
		case "diff-config":
			return runSpaceDiffConfig(args[1:])
		case "apply-config":
			return runSpaceApplyConfig(args[1:])
		}
	}

	fs := flag.NewFlagSet("space", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("space requires --space")
	}

	store, _, _, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	cfg, err := store.GetSpaceConfig(*spaceID)
	if err != nil {
		return err
	}
	return printJSON(cfg)
}

func runSpaceValidateConfig(args []string) error {
	fs := flag.NewFlagSet("space validate-config", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("space validate-config requires --space")
	}

	reconcile, _, err := resolveSpaceConfigReconcile(*workspace, *spaceID)
	if err != nil {
		return err
	}
	return printJSON(struct {
		Valid bool                          `json:"valid"`
		Diff  omnethdb.SpaceConfigReconcile `json:"diff"`
	}{
		Valid: reconcile.Applyable,
		Diff:  reconcile,
	})
}

func runSpaceDiffConfig(args []string) error {
	fs := flag.NewFlagSet("space diff-config", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("space diff-config requires --space")
	}

	reconcile, _, err := resolveSpaceConfigReconcile(*workspace, *spaceID)
	if err != nil {
		return err
	}
	return printJSON(reconcile)
}

func runSpaceApplyConfig(args []string) error {
	fs := flag.NewFlagSet("space apply-config", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	spaceID := fs.String("space", "", "space id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("space apply-config requires --space")
	}

	reconcile, store, err := resolveSpaceConfigReconcile(*workspace, *spaceID)
	if err != nil {
		return err
	}
	defer store.Close()

	if !reconcile.Applyable {
		return fmt.Errorf("config cannot be applied: %s", strings.Join(reconcile.Errors, "; "))
	}
	updated, err := store.UpdateSpaceConfig(*spaceID, reconcile.Desired)
	if err != nil {
		return err
	}
	return printJSON(struct {
		Applied bool                          `json:"applied"`
		Config  *omnethdb.SpaceConfig         `json:"config"`
		Diff    omnethdb.SpaceConfigReconcile `json:"diff"`
	}{
		Applied: true,
		Config:  updated,
		Diff:    reconcile,
	})
}

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, layout, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()
	return printJSON(struct {
		Layout omnethdb.WorkspaceLayout `json:"layout"`
		Config *omnethdb.RuntimeConfig  `json:"config"`
	}{
		Layout: layout,
		Config: cfg,
	})
}

func resolveSpaceConfigReconcile(workspace string, spaceID string) (omnethdb.SpaceConfigReconcile, *omnethdb.Store, error) {
	store, _, cfg, err := openCLIStore(workspace)
	if err != nil {
		return omnethdb.SpaceConfigReconcile{}, nil, err
	}
	persisted, err := store.GetSpaceConfig(spaceID)
	if err != nil {
		_ = store.Close()
		return omnethdb.SpaceConfigReconcile{}, nil, err
	}
	reconcile := cfg.ReconcileSpaceConfig(spaceID, *persisted)
	return reconcile, store, nil
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	addr := fs.String("addr", ":8080", "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	handler := httpapi.NewHandler(store, cfg)
	fmt.Fprintf(os.Stdout, "omnethdb http api listening on %s\n", *addr)
	return http.ListenAndServe(*addr, handler)
}

func runServeGRPC(args []string) error {
	fs := flag.NewFlagSet("serve-grpc", flag.ContinueOnError)
	workspace := fs.String("workspace", defaultWorkspace, "workspace root")
	addr := fs.String("addr", ":9090", "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, _, cfg, err := openCLIStore(*workspace)
	if err != nil {
		return err
	}
	defer store.Close()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	defer lis.Close()

	server := grpc.NewServer()
	omnethdbv1.RegisterOmnethDBServer(server, grpcapi.NewServer(store, cfg))
	fmt.Fprintf(os.Stdout, "omnethdb grpc api listening on %s\n", *addr)
	return server.Serve(lis)
}

func openCLIStore(workspace string) (*omnethdb.Store, omnethdb.WorkspaceLayout, *omnethdb.RuntimeConfig, error) {
	store, layout, err := omnethdb.OpenWorkspace(workspace)
	if err != nil {
		return nil, omnethdb.WorkspaceLayout{}, nil, err
	}
	cfg, err := omnethdb.LoadRuntimeConfig(layout.ConfigPath)
	if err != nil {
		_ = store.Close()
		return nil, omnethdb.WorkspaceLayout{}, nil, err
	}
	for spaceID, settings := range cfg.Spaces {
		if settings.Embedder.ModelID == "" || settings.Embedder.Dimensions <= 0 {
			continue
		}
		_ = spaceID
		store.RegisterEmbedder(hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions))
	}
	return store, layout, cfg, nil
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

func embedderForMigration(store *omnethdb.Store, cfg *omnethdb.RuntimeConfig, spaceID string, modelID string, dimensions int) omnethdb.Embedder {
	if strings.TrimSpace(modelID) != "" && dimensions > 0 {
		return hashembedder.New(modelID, dimensions)
	}
	if cfg != nil {
		if settings, ok := cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				return hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions)
			}
		}
	}
	if persisted, err := store.GetSpaceConfig(spaceID); err == nil {
		return hashembedder.New(persisted.EmbeddingModelID, persisted.Dimension)
	}
	return hashembedder.New(defaultModelID, defaultDimension)
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

func parseMemoryKinds(raw string) []omnethdb.MemoryKind {
	parts := splitCSV(raw)
	if len(parts) == 0 {
		return nil
	}
	out := make([]omnethdb.MemoryKind, 0, len(parts))
	for _, part := range parts {
		out = append(out, parseMemoryKind(part))
	}
	return out
}

func parseActor(id string, kind string) omnethdb.Actor {
	return omnethdb.Actor{
		ID:   id,
		Kind: parseActorKind(kind),
	}
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

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func usage() {
	fmt.Fprintf(os.Stderr, `omnethdb - embedded versioned memory database

Usage:
  omnethdb <command> [flags]

Commands:
  init      bootstrap a space
  remember  write a memory
  lint-remember preview duplicate/update/blob warnings for a candidate memory
  recall    query live memories
  profile   build a layered memory profile
  forget    forget a memory
  revive    revive an inactive lineage
  lineage   inspect a lineage
  related   traverse explicit relations
  candidates raw cosine candidate search
  quality   inspect duplicate and update diagnostics for a space
  quality-plan build an advisory duplicate cleanup plan for a space
  audit     inspect audit history
  forget-batch forget multiple memories in one explicit operator action
  export    render inspection exports
  migrate   migrate a space to a new embedder
  space     print persisted space config
  config    print workspace layout and loaded config
  serve     run the HTTP API server
  serve-grpc run the gRPC API server

Space subcommands:
  space validate-config --workspace . --space repo:company/app
  space diff-config --workspace . --space repo:company/app
  space apply-config --workspace . --space repo:company/app

Examples:
  omnethdb init --workspace . --space repo:company/app
  omnethdb remember --workspace . --space repo:company/app --kind static --content "payments use cursor pagination"
  omnethdb lint-remember --workspace . --space repo:company/app --kind static --content "payments use signed cursor pagination"
  omnethdb recall --workspace . --spaces repo:company/app --query pagination
  omnethdb candidates --workspace . --space repo:company/app --content "pagination"
  omnethdb quality --workspace . --space repo:company/app
  omnethdb quality-plan --workspace . --space repo:company/app
  omnethdb forget-batch --workspace . --ids mem-1,mem-2 --reason "duplicate cleanup"
  omnethdb export --workspace . --space repo:company/app --format summary-md
  omnethdb space diff-config --workspace . --space repo:company/app
  omnethdb serve --workspace . --addr :8080
  omnethdb serve-grpc --workspace . --addr :9090
`)
}
