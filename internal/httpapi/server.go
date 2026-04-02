package httpapi

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	omnethdb "omnethdb"
	hashembedder "omnethdb/embedders/hash"
)

const (
	defaultModelID   = "builtin/hash-embedder-v1"
	defaultDimension = 256
)

type Server struct {
	store *omnethdb.Store
	cfg   *omnethdb.RuntimeConfig
}

func NewHandler(store *omnethdb.Store, cfg *omnethdb.RuntimeConfig) http.Handler {
	s := &Server{store: store, cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("/inspect", s.handleInspector)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/config", s.handleConfig)
	mux.HandleFunc("/v1/recall", s.handleRecall)
	mux.HandleFunc("/v1/profile", s.handleProfile)
	mux.HandleFunc("/v1/candidates", s.handleCandidates)
	mux.HandleFunc("/v1/memories/list", s.handleListMemories)
	mux.HandleFunc("/v1/memories/remember-batch", s.handleRememberBatch)
	mux.HandleFunc("/v1/memories/remember", s.handleRemember)
	mux.HandleFunc("/v1/memories/", s.handleMemoryAction)
	mux.HandleFunc("/v1/lineages/", s.handleLineageAction)
	mux.HandleFunc("/v1/spaces/init", s.handleSpaceInit)
	mux.HandleFunc("/v1/spaces/config", s.handleSpaceConfig)
	mux.HandleFunc("/v1/spaces/migrate", s.handleSpaceMigrate)
	mux.HandleFunc("/v1/audit", s.handleAudit)
	mux.HandleFunc("/v1/export/compare", s.handleExportCompare)
	mux.HandleFunc("/v1/export/diff", s.handleExportDiff)
	mux.HandleFunc("/v1/export", s.handleExport)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleInspector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	spaceID := strings.TrimSpace(r.URL.Query().Get("space_id"))
	if spaceID == "" {
		spaceID = "repo:company/app"
	}
	data := struct {
		SpaceID    string
		Since      string
		BeforeFile string
		AfterFile  string
	}{
		SpaceID:    spaceID,
		Since:      strings.TrimSpace(r.URL.Query().Get("since")),
		BeforeFile: strings.TrimSpace(r.URL.Query().Get("before_file")),
		AfterFile:  strings.TrimSpace(r.URL.Query().Get("after_file")),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = inspectorTemplate.Execute(w, data)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, s.cfg)
}

func (s *Server) handleRemember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input omnethdb.MemoryInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.ensureEmbedderForSpace(input.SpaceID)
	mem, err := s.store.Remember(input)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mem)
}

func (s *Server) handleRememberBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Inputs []omnethdb.MemoryInput `json:"inputs"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	spaceIDs := make([]string, 0, len(body.Inputs))
	seen := make(map[string]struct{}, len(body.Inputs))
	for _, input := range body.Inputs {
		if _, ok := seen[input.SpaceID]; ok {
			continue
		}
		seen[input.SpaceID] = struct{}{}
		spaceIDs = append(spaceIDs, input.SpaceID)
	}
	s.ensureEmbeddersForSpaces(spaceIDs)
	memories, err := s.store.RememberBatch(body.Inputs)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, memories)
}

func (s *Server) handleRecall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req omnethdb.RecallRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.ensureEmbeddersForSpaces(req.SpaceIDs)
	results, err := s.store.Recall(req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleListMemories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req omnethdb.ListMemoriesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	memories, err := s.store.ListMemories(req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, memories)
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req omnethdb.ProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.ensureEmbeddersForSpaces(req.SpaceIDs)
	profile, err := s.store.GetProfile(req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req omnethdb.FindCandidatesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.ensureEmbedderForSpace(req.SpaceID)
	results, err := s.store.FindCandidates(req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	spaceID := r.URL.Query().Get("space_id")
	if strings.TrimSpace(spaceID) == "" {
		writeError(w, http.StatusBadRequest, "space_id is required")
		return
	}
	sinceTime, err := parseOptionalTime(r.URL.Query().Get("since"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	audit, err := s.store.GetAuditLog(spaceID, sinceTime)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, audit)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	spaceID := r.URL.Query().Get("space_id")
	if strings.TrimSpace(spaceID) == "" {
		writeError(w, http.StatusBadRequest, "space_id is required")
		return
	}

	req := omnethdb.ExportRequest{SpaceID: spaceID}
	switch omnethdb.ExportFormat(strings.TrimSpace(r.URL.Query().Get("format"))) {
	case "", omnethdb.ExportFormatSnapshotJSON:
		snapshot, err := s.store.ExportSnapshot(req)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
	case omnethdb.ExportFormatSummaryMD:
		out, err := s.store.RenderExportSummaryMarkdown(req)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeText(w, http.StatusOK, out)
	case omnethdb.ExportFormatGraphMermaid:
		out, err := s.store.RenderExportGraphMermaid(req)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeText(w, http.StatusOK, out)
	default:
		writeError(w, http.StatusBadRequest, "invalid format")
	}
}

func (s *Server) handleExportDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	spaceID := r.URL.Query().Get("space_id")
	if strings.TrimSpace(spaceID) == "" {
		writeError(w, http.StatusBadRequest, "space_id is required")
		return
	}
	since, err := parseOptionalTime(r.URL.Query().Get("since"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if since.IsZero() {
		writeError(w, http.StatusBadRequest, "since is required")
		return
	}
	until, err := parseOptionalTime(r.URL.Query().Get("until"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	diff, err := s.store.ExportDiff(omnethdb.ExportDiffRequest{
		SpaceID: spaceID,
		Since:   since,
		Until:   until,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, diff)
}

func (s *Server) handleExportCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	beforeFile := strings.TrimSpace(r.URL.Query().Get("before_file"))
	afterFile := strings.TrimSpace(r.URL.Query().Get("after_file"))
	if beforeFile == "" || afterFile == "" {
		writeError(w, http.StatusBadRequest, "before_file and after_file are required")
		return
	}

	before, err := readSnapshotFile(beforeFile)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	after, err := readSnapshotFile(afterFile)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, omnethdb.CompareExportSnapshots(before, after))
}

func (s *Server) handleMemoryAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/memories/")
	parts := splitPath(path)
	if len(parts) == 2 && parts[1] == "forget" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var body struct {
			Actor  omnethdb.Actor `json:"actor"`
			Reason string         `json:"reason"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.store.Forget(parts[0], body.Actor, body.Reason); err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
		return
	}
	if len(parts) == 3 && parts[1] == "related" && r.Method == http.MethodGet {
		depth := 1
		if raw := r.URL.Query().Get("depth"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid depth")
				return
			}
			depth = parsed
		}
		related, err := s.store.GetRelated(parts[0], parseRelationType(parts[2]), depth)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, related)
		return
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) handleLineageAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/lineages/")
	parts := splitPath(path)
	if len(parts) == 1 && r.Method == http.MethodGet {
		lineage, err := s.store.GetLineage(parts[0])
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, lineage)
		return
	}
	if len(parts) == 2 && parts[1] == "revive" && r.Method == http.MethodPost {
		var input omnethdb.ReviveInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		mem, err := s.store.Revive(parts[0], input)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, mem)
		return
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) handleSpaceInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		SpaceID string `json:"space_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfgSource := s.configOrEmpty()
	embedder := s.embedderForBootstrap(body.SpaceID)
	init := cfgSource.SpaceInit(body.SpaceID, omnethdb.SpaceInit{
		DefaultWeight: 1.0,
		HalfLifeDays:  30,
		WritePolicy:   omnethdb.DefaultSpaceWritePolicy(),
	})
	cfg, err := s.store.EnsureSpace(body.SpaceID, embedder, init)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleSpaceConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	spaceID := r.URL.Query().Get("space_id")
	if strings.TrimSpace(spaceID) == "" {
		writeError(w, http.StatusBadRequest, "space_id is required")
		return
	}
	cfg, err := s.store.GetSpaceConfig(spaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleSpaceMigrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		SpaceID    string `json:"space_id"`
		ModelID    string `json:"model_id"`
		Dimensions int    `json:"dimensions"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	embedder := s.embedderForMigration(body.SpaceID, body.ModelID, body.Dimensions)
	if err := s.store.MigrateEmbeddings(body.SpaceID, embedder); err != nil {
		writeStoreError(w, err)
		return
	}
	cfg, err := s.store.GetSpaceConfig(body.SpaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) ensureEmbeddersForSpaces(spaceIDs []string) {
	for _, spaceID := range spaceIDs {
		s.ensureEmbedderForSpace(spaceID)
	}
}

func (s *Server) ensureEmbedderForSpace(spaceID string) {
	if strings.TrimSpace(spaceID) == "" || s.store == nil {
		return
	}
	if s.cfg != nil {
		if settings, ok := s.cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				s.store.RegisterEmbedder(hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions))
				return
			}
		}
	}
	if persisted, err := s.store.GetSpaceConfig(spaceID); err == nil {
		s.store.RegisterEmbedder(hashembedder.New(persisted.EmbeddingModelID, persisted.Dimension))
	}
}

func (s *Server) embedderForBootstrap(spaceID string) omnethdb.Embedder {
	if cfg := s.configOrEmpty(); cfg != nil {
		if settings, ok := cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				return hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions)
			}
		}
	}
	return hashembedder.New(defaultModelID, defaultDimension)
}

func (s *Server) embedderForMigration(spaceID string, modelID string, dimensions int) omnethdb.Embedder {
	if strings.TrimSpace(modelID) != "" && dimensions > 0 {
		return hashembedder.New(modelID, dimensions)
	}
	if cfg := s.configOrEmpty(); cfg != nil {
		if settings, ok := cfg.SpaceSettings(spaceID); ok {
			if settings.Embedder.ModelID != "" && settings.Embedder.Dimensions > 0 {
				return hashembedder.New(settings.Embedder.ModelID, settings.Embedder.Dimensions)
			}
		}
	}
	if persisted, err := s.store.GetSpaceConfig(spaceID); err == nil {
		return hashembedder.New(persisted.EmbeddingModelID, persisted.Dimension)
	}
	return hashembedder.New(defaultModelID, defaultDimension)
}

func (s *Server) configOrEmpty() *omnethdb.RuntimeConfig {
	if s.cfg != nil {
		return s.cfg
	}
	return &omnethdb.RuntimeConfig{}
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

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

var inspectorTemplate = template.Must(template.New("inspector").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>OmnethDB Inspector</title>
  <style>
    :root {
      --bg: #f4efe6;
      --panel: #fffaf2;
      --panel-strong: #f7f0e4;
      --ink: #1f2937;
      --muted: #5b6472;
      --line: #d6c7b0;
      --accent: #9a3412;
      --accent-soft: #fed7aa;
      --ok: #3f6212;
      --ok-soft: #d9f99d;
      --font-sans: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Georgia, serif;
      --font-mono: "SFMono-Regular", "SF Mono", "Cascadia Code", Menlo, monospace;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: var(--font-sans);
      color: var(--ink);
      background:
        radial-gradient(circle at top left, #fff7ed 0, transparent 28rem),
        radial-gradient(circle at bottom right, #fde68a 0, transparent 24rem),
        linear-gradient(180deg, #f8f4ec 0%, var(--bg) 100%);
      min-height: 100vh;
    }
    .shell {
      max-width: 1500px;
      margin: 0 auto;
      padding: 24px;
    }
    .hero {
      background: linear-gradient(135deg, rgba(154, 52, 18, 0.08), rgba(63, 98, 18, 0.08));
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 24px;
      box-shadow: 0 20px 60px rgba(84, 54, 22, 0.08);
    }
    h1, h2 {
      margin: 0 0 8px;
      font-weight: 700;
    }
    p {
      margin: 0;
      color: var(--muted);
      line-height: 1.5;
    }
    .controls {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto auto;
      gap: 12px;
      margin-top: 20px;
    }
    input, button {
      border-radius: 14px;
      border: 1px solid var(--line);
      padding: 12px 14px;
      font: inherit;
    }
    input {
      background: rgba(255,255,255,0.75);
      min-width: 0;
    }
    button {
      cursor: pointer;
      background: var(--panel);
    }
    button.primary {
      background: linear-gradient(135deg, var(--accent) 0%, #c2410c 100%);
      color: white;
      border-color: #7c2d12;
    }
    .meta {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 14px;
      color: var(--muted);
      font-size: 14px;
    }
    .pill {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: 6px 10px;
    }
    .grid {
      display: grid;
      grid-template-columns: 1.1fr 1fr;
      gap: 18px;
      margin-top: 20px;
    }
    .panel {
      background: rgba(255, 250, 242, 0.88);
      border: 1px solid var(--line);
      border-radius: 20px;
      overflow: hidden;
      box-shadow: 0 12px 30px rgba(84, 54, 22, 0.06);
    }
    .panel-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
      padding: 14px 16px;
      background: var(--panel-strong);
      border-bottom: 1px solid var(--line);
    }
    .panel-header span {
      color: var(--muted);
      font-size: 14px;
    }
    .body {
      padding: 16px;
    }
    pre {
      margin: 0;
      white-space: pre-wrap;
      word-break: break-word;
      font-family: var(--font-mono);
      font-size: 13px;
      line-height: 1.5;
    }
    .stack {
      display: grid;
      gap: 18px;
    }
    .toolbar {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
    .toggle {
      background: #fff;
      color: var(--ink);
    }
    .toggle.active {
      background: var(--ok-soft);
      border-color: var(--ok);
      color: #21360a;
    }
    .cards {
      display: grid;
      gap: 12px;
    }
    .memory-card, .signal {
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 14px;
      background: rgba(255,255,255,0.7);
    }
    .memory-card header, .signal header {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: baseline;
      margin-bottom: 8px;
    }
    .memory-card h3, .signal h3 {
      margin: 0;
      font-size: 16px;
    }
    .memory-meta {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin: 10px 0;
      color: var(--muted);
      font-size: 13px;
    }
    .chip {
      border-radius: 999px;
      padding: 4px 8px;
      border: 1px solid var(--line);
      background: var(--panel);
    }
    .signal.good {
      border-color: #84cc16;
      background: rgba(236, 252, 203, 0.7);
    }
    .signal.warn {
      border-color: #f59e0b;
      background: rgba(254, 240, 138, 0.45);
    }
    .signal.bad {
      border-color: #dc2626;
      background: rgba(254, 202, 202, 0.45);
    }
    .status {
      color: var(--muted);
      font-size: 14px;
    }
    .status.error {
      color: #991b1b;
    }
    #graph {
      min-height: 240px;
    }
    .hidden {
      display: none;
    }
    @media (max-width: 980px) {
      .grid {
        grid-template-columns: 1fr;
      }
      .controls {
        grid-template-columns: 1fr;
      }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <h1>OmnethDB Inspector</h1>
      <p>Cold-path view over exported memory state. Use this to inspect live corpus, lineage history, relations, and audit without touching retrieval semantics.</p>
      <form class="controls" id="controls">
        <input id="space" name="space_id" value="{{.SpaceID}}" spellcheck="false" aria-label="Space ID">
        <input id="since" name="since" value="{{.Since}}" spellcheck="false" placeholder="RFC3339 since for diff mode" aria-label="Diff since">
        <input id="before-file" name="before_file" value="{{.BeforeFile}}" spellcheck="false" placeholder="Absolute path to before snapshot" aria-label="Before snapshot file">
        <input id="after-file" name="after_file" value="{{.AfterFile}}" spellcheck="false" placeholder="Absolute path to after snapshot" aria-label="After snapshot file">
        <button class="primary" type="submit">Refresh</button>
        <button type="button" id="copy-link">Copy Link</button>
      </form>
      <div class="meta">
        <div class="pill">Space: <strong id="space-label">{{.SpaceID}}</strong></div>
        <div class="pill">Live memories: <strong id="live-count">-</strong></div>
        <div class="pill">Lineages: <strong id="lineage-count">-</strong></div>
        <div class="pill">Audit entries: <strong id="audit-count">-</strong></div>
      </div>
    </section>

    <section class="grid">
      <div class="stack">
        <article class="panel">
          <div class="panel-header">
            <div>
              <h2>Diff</h2>
              <span>Compare by time window or by saved snapshot files</span>
            </div>
            <span class="status" id="diff-status">Idle</span>
          </div>
          <div class="body">
            <div class="cards" id="diff-list">
              <article class="memory-card"><p>Set a <code>since</code> timestamp or two snapshot file paths to compare memory state.</p></article>
            </div>
          </div>
        </article>

        <article class="panel">
          <div class="panel-header">
            <div>
              <h2>Signals</h2>
              <span>Fast triage for suspicious memory state</span>
            </div>
            <span class="status" id="signals-status">Loading…</span>
          </div>
          <div class="body">
            <div class="cards" id="signals"></div>
          </div>
        </article>

        <article class="panel">
          <div class="panel-header">
            <div>
              <h2>Live View</h2>
              <span>Client-side filtered live corpus</span>
            </div>
            <div class="toolbar" id="live-filters">
              <button class="toggle active" type="button" data-kind="all">All</button>
              <button class="toggle" type="button" data-kind="static">Static</button>
              <button class="toggle" type="button" data-kind="episodic">Episodic</button>
              <button class="toggle" type="button" data-kind="derived">Derived</button>
            </div>
          </div>
          <div class="body">
            <div class="cards" id="live-list"></div>
          </div>
        </article>

        <article class="panel">
          <div class="panel-header">
            <div>
              <h2>Summary</h2>
              <span>Markdown export for fast human review</span>
            </div>
            <span class="status" id="summary-status">Loading…</span>
          </div>
          <div class="body"><pre id="summary"></pre></div>
        </article>

        <article class="panel">
          <div class="panel-header">
            <div>
              <h2>Snapshot</h2>
              <span>Canonical JSON for audit and diffing</span>
            </div>
            <span class="status" id="snapshot-status">Loading…</span>
          </div>
          <div class="body"><pre id="snapshot"></pre></div>
        </article>
      </div>

      <div class="stack">
        <article class="panel">
          <div class="panel-header">
            <div>
              <h2>Graph</h2>
              <span>Mermaid render with raw fallback</span>
            </div>
            <span class="status" id="graph-status">Loading…</span>
          </div>
          <div class="body">
            <div id="graph"></div>
            <pre id="graph-raw" class="hidden"></pre>
          </div>
        </article>
      </div>
    </section>
  </div>

  <script type="module">
    const controls = document.getElementById("controls");
    const spaceInput = document.getElementById("space");
    const sinceInput = document.getElementById("since");
    const beforeFileInput = document.getElementById("before-file");
    const afterFileInput = document.getElementById("after-file");
    const spaceLabel = document.getElementById("space-label");
    const liveCount = document.getElementById("live-count");
    const lineageCount = document.getElementById("lineage-count");
    const auditCount = document.getElementById("audit-count");
    const summaryEl = document.getElementById("summary");
    const snapshotEl = document.getElementById("snapshot");
    const signalsEl = document.getElementById("signals");
    const signalsStatus = document.getElementById("signals-status");
    const diffListEl = document.getElementById("diff-list");
    const diffStatus = document.getElementById("diff-status");
    const liveListEl = document.getElementById("live-list");
    const liveFilters = document.getElementById("live-filters");
    const graphEl = document.getElementById("graph");
    const graphRawEl = document.getElementById("graph-raw");
    const summaryStatus = document.getElementById("summary-status");
    const snapshotStatus = document.getElementById("snapshot-status");
    const graphStatus = document.getElementById("graph-status");
    const copyLinkButton = document.getElementById("copy-link");
    let currentSnapshot = null;
    let selectedKind = "all";

    controls.addEventListener("submit", (event) => {
      event.preventDefault();
      const next = new URL(window.location.href);
      next.searchParams.set("space_id", spaceInput.value.trim());
      if (sinceInput.value.trim()) {
        next.searchParams.set("since", sinceInput.value.trim());
      } else {
        next.searchParams.delete("since");
      }
      if (beforeFileInput.value.trim()) {
        next.searchParams.set("before_file", beforeFileInput.value.trim());
      } else {
        next.searchParams.delete("before_file");
      }
      if (afterFileInput.value.trim()) {
        next.searchParams.set("after_file", afterFileInput.value.trim());
      } else {
        next.searchParams.delete("after_file");
      }
      window.history.replaceState({}, "", next);
      load();
    });

    copyLinkButton.addEventListener("click", async () => {
      const url = new URL(window.location.href);
      url.searchParams.set("space_id", spaceInput.value.trim());
      if (sinceInput.value.trim()) {
        url.searchParams.set("since", sinceInput.value.trim());
      } else {
        url.searchParams.delete("since");
      }
      if (beforeFileInput.value.trim()) {
        url.searchParams.set("before_file", beforeFileInput.value.trim());
      } else {
        url.searchParams.delete("before_file");
      }
      if (afterFileInput.value.trim()) {
        url.searchParams.set("after_file", afterFileInput.value.trim());
      } else {
        url.searchParams.delete("after_file");
      }
      await navigator.clipboard.writeText(url.toString());
      copyLinkButton.textContent = "Copied";
      window.setTimeout(() => { copyLinkButton.textContent = "Copy Link"; }, 1200);
    });

    liveFilters.addEventListener("click", (event) => {
      const button = event.target.closest("[data-kind]");
      if (!button) {
        return;
      }
      selectedKind = button.dataset.kind || "all";
      for (const item of liveFilters.querySelectorAll("[data-kind]")) {
        item.classList.toggle("active", item === button);
      }
      renderLiveList(currentSnapshot);
    });

    async function fetchText(url) {
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(await response.text());
      }
      return response.text();
    }

    async function fetchJSON(url) {
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(await response.text());
      }
      return response.json();
    }

    async function renderGraph(graphText) {
      graphEl.innerHTML = "";
      graphRawEl.textContent = graphText;
      graphRawEl.classList.add("hidden");
      try {
        const mermaid = await import("https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs");
        mermaid.default.initialize({ startOnLoad: false, theme: "neutral" });
        const id = "graph-" + Date.now();
        const rendered = await mermaid.default.render(id, graphText);
        graphEl.innerHTML = rendered.svg;
        graphStatus.textContent = "Rendered";
      } catch (error) {
        graphRawEl.classList.remove("hidden");
        graphStatus.textContent = "Raw Mermaid fallback";
      }
    }

    function kindLabel(kind) {
      switch (kind) {
        case 0: return "episodic";
        case 1: return "static";
        case 2: return "derived";
        default: return "unknown";
      }
    }

    function escapeHTML(value) {
      return String(value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
    }

    function renderSignals(snapshot) {
      const live = snapshot?.live_memories ?? [];
      const lineages = snapshot?.lineages ?? [];
      const audit = snapshot?.audit_entries ?? [];
      const orphaned = live.filter(mem => mem.kind === 2 && mem.has_orphaned_sources);
      const episodic = live.filter(mem => mem.kind === 0);
      const denseLineages = lineages.filter(lineage => (lineage.memories?.length ?? 0) >= 4);
      const frequentUpdates = audit.filter(entry => entry.operation === "remember" && (entry.memory_ids?.length ?? 0) > 1);

      const signals = [];
      if (orphaned.length > 0) {
        signals.push({
          tone: "warn",
          title: "Orphaned derived memories",
          body: String(orphaned.length) + " live derived memories still depend on forgotten sources. Revalidation or retirement is likely needed.",
        });
      }
      if (episodic.length >= 8) {
        signals.push({
          tone: "warn",
          title: "Episodic pile-up",
          body: String(episodic.length) + " live episodic memories are active. This may mean session observations are not being folded into durable knowledge.",
        });
      }
      if (denseLineages.length > 0) {
        signals.push({
          tone: "warn",
          title: "Long lineages",
          body: String(denseLineages.length) + " lineages have four or more versions. That often means a fact is churning and deserves review.",
        });
      }
      if (frequentUpdates.length >= 5) {
        signals.push({
          tone: "warn",
          title: "Heavy update traffic",
          body: String(frequentUpdates.length) + " update writes were recorded. Check whether the agent is revising cleanly or repeatedly restating the same fact.",
        });
      }
      if (signals.length === 0) {
        signals.push({
          tone: "good",
          title: "No obvious memory hygiene issues",
          body: "The current snapshot does not show orphaned derives, heavy episodic buildup, or unusually churny lineages.",
        });
      }

      signalsEl.innerHTML = signals.map(signal =>
        '<article class="signal ' + escapeHTML(signal.tone) + '">' +
          '<header><h3>' + escapeHTML(signal.title) + '</h3></header>' +
          '<p>' + escapeHTML(signal.body) + '</p>' +
        '</article>'
      ).join("");
      signalsStatus.textContent = "Ready";
    }

    function renderLiveList(snapshot) {
      const live = (snapshot?.live_memories ?? []).filter(mem => selectedKind === "all" || kindLabel(mem.kind) === selectedKind);
      if (live.length === 0) {
        liveListEl.innerHTML = '<article class="memory-card"><p>No live memories match the current filter.</p></article>';
        return;
      }
      liveListEl.innerHTML = live.map(mem => {
        const orphanedChip = mem.has_orphaned_sources ? '<span class="chip">orphaned sources</span>' : '';
        return '<article class="memory-card">' +
          '<header>' +
            '<h3>' + escapeHTML(mem.content) + '</h3>' +
            '<span class="chip">' + escapeHTML(kindLabel(mem.kind)) + '</span>' +
          '</header>' +
          '<div class="memory-meta">' +
            '<span class="chip">id ' + escapeHTML(mem.id) + '</span>' +
            '<span class="chip">v' + escapeHTML(mem.version) + '</span>' +
            '<span class="chip">actor ' + escapeHTML(mem.actor?.id ?? "unknown") + '</span>' +
            '<span class="chip">confidence ' + (mem.confidence ?? 0).toFixed(2) + '</span>' +
            orphanedChip +
          '</div>' +
          '<p>' + escapeHTML(mem.rationale || "No rationale recorded.") + '</p>' +
        '</article>';
      }).join("");
    }

    function renderDiff(diff) {
      if (!diff) {
        diffListEl.innerHTML = '<article class="memory-card"><p>Set a <code>since</code> timestamp or two snapshot file paths to compare memory state.</p></article>';
        diffStatus.textContent = "Idle";
        return;
      }
      const cards = [];
      cards.push(
        '<article class="signal warn">' +
          '<header><h3>Delta summary</h3></header>' +
          '<p>Added live: ' + escapeHTML(diff.added_live_memories?.length ?? 0) +
          ' | Removed live: ' + escapeHTML(diff.removed_live_memories?.length ?? 0) +
          ' | New orphaned derives: ' + escapeHTML(diff.new_orphaned_derives?.length ?? 0) +
          ' | Added audit entries: ' + escapeHTML(diff.added_audit_entries?.length ?? 0) +
          '</p>' +
        '</article>'
      );

      const sections = [
        { title: "Added live memories", items: diff.added_live_memories ?? [] },
        { title: "Removed live memories", items: diff.removed_live_memories ?? [] },
        { title: "New orphaned derives", items: diff.new_orphaned_derives ?? [] },
      ];
      for (const section of sections) {
        if (section.items.length === 0) {
          continue;
        }
        cards.push(
          '<article class="memory-card">' +
            '<header><h3>' + escapeHTML(section.title) + '</h3></header>' +
            '<div class="memory-meta">' + section.items.map(item =>
              '<span class="chip">' + escapeHTML(item.id) + ' v' + escapeHTML(item.version) + ' ' + escapeHTML(kindLabel(item.kind)) + '</span>'
            ).join("") + '</div>' +
          '</article>'
        );
      }

      if ((diff.changed_lineages ?? []).length > 0) {
        cards.push(
          '<article class="memory-card">' +
            '<header><h3>Changed lineages</h3></header>' +
            '<div class="memory-meta">' + diff.changed_lineages.map(item =>
              '<span class="chip">' + escapeHTML(item.root_id) + ' versions ' +
              escapeHTML(item.before_versions) + '→' + escapeHTML(item.after_versions) + '</span>'
            ).join("") + '</div>' +
          '</article>'
        );
      }

      diffListEl.innerHTML = cards.join("");
      diffStatus.textContent = "Ready";
    }

    async function load() {
      const spaceID = spaceInput.value.trim();
      if (!spaceID) {
        return;
      }
      spaceLabel.textContent = spaceID;
      summaryStatus.textContent = "Loading…";
      snapshotStatus.textContent = "Loading…";
      signalsStatus.textContent = "Loading…";
      diffStatus.textContent = (sinceInput.value.trim() || (beforeFileInput.value.trim() && afterFileInput.value.trim())) ? "Loading…" : "Idle";
      graphStatus.textContent = "Loading…";
      summaryStatus.className = "status";
      snapshotStatus.className = "status";
      signalsStatus.className = "status";
      graphStatus.className = "status";
      try {
        const base = "/v1/export?space_id=" + encodeURIComponent(spaceID);
        let diffURL = null;
        if (beforeFileInput.value.trim() && afterFileInput.value.trim()) {
          diffURL = "/v1/export/compare?before_file=" + encodeURIComponent(beforeFileInput.value.trim()) + "&after_file=" + encodeURIComponent(afterFileInput.value.trim());
        } else if (sinceInput.value.trim()) {
          diffURL = "/v1/export/diff?space_id=" + encodeURIComponent(spaceID) + "&since=" + encodeURIComponent(sinceInput.value.trim());
        }
        const [summary, snapshot, graph, diff] = await Promise.all([
          fetchText(base + "&format=summary-md"),
          fetchJSON(base),
          fetchText(base + "&format=graph-mermaid"),
          diffURL ? fetchJSON(diffURL) : Promise.resolve(null),
        ]);

        summaryEl.textContent = summary;
        snapshotEl.textContent = JSON.stringify(snapshot, null, 2);
        currentSnapshot = snapshot;
        liveCount.textContent = String(snapshot.live_memories?.length ?? 0);
        lineageCount.textContent = String(snapshot.lineages?.length ?? 0);
        auditCount.textContent = String(snapshot.audit_entries?.length ?? 0);
        summaryStatus.textContent = "Ready";
        snapshotStatus.textContent = "Ready";
        renderSignals(snapshot);
        renderLiveList(snapshot);
        renderDiff(diff);
        await renderGraph(graph);
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        summaryStatus.textContent = "Load failed";
        snapshotStatus.textContent = "Load failed";
        signalsStatus.textContent = "Load failed";
        diffStatus.textContent = "Load failed";
        graphStatus.textContent = "Load failed";
        summaryStatus.className = "status error";
        snapshotStatus.className = "status error";
        signalsStatus.className = "status error";
        diffStatus.className = "status error";
        graphStatus.className = "status error";
        summaryEl.textContent = message;
        snapshotEl.textContent = message;
        signalsEl.innerHTML = '<article class="signal bad"><p>' + escapeHTML(message) + '</p></article>';
        diffListEl.innerHTML = '<article class="memory-card"><p>' + escapeHTML(message) + '</p></article>';
        liveListEl.innerHTML = '<article class="memory-card"><p>' + escapeHTML(message) + '</p></article>';
        graphEl.innerHTML = "";
        graphRawEl.textContent = message;
        graphRawEl.classList.remove("hidden");
      }
    }

    load();
  </script>
</body>
</html>`))

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeStoreError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, omnethdb.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, omnethdb.ErrSpaceNotFound), errors.Is(err, omnethdb.ErrMemoryNotFound),
		errors.Is(err, omnethdb.ErrUpdateTargetNotFound), errors.Is(err, omnethdb.ErrExtendsTargetNotFound),
		errors.Is(err, omnethdb.ErrDerivedSourceNotFound):
		status = http.StatusNotFound
	}
	writeError(w, status, err.Error())
}

func readSnapshotFile(path string) (omnethdb.ExportSnapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return omnethdb.ExportSnapshot{}, err
	}
	var snapshot omnethdb.ExportSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return omnethdb.ExportSnapshot{}, err
	}
	return snapshot, nil
}

func parseOptionalTime(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(http.TimeFormat, raw)
	if err == nil {
		return t, nil
	}
	t2, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, err
	}
	return t2, nil
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
