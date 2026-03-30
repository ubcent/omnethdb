package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
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
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/config", s.handleConfig)
	mux.HandleFunc("/v1/recall", s.handleRecall)
	mux.HandleFunc("/v1/profile", s.handleProfile)
	mux.HandleFunc("/v1/candidates", s.handleCandidates)
	mux.HandleFunc("/v1/memories/remember", s.handleRemember)
	mux.HandleFunc("/v1/memories/", s.handleMemoryAction)
	mux.HandleFunc("/v1/lineages/", s.handleLineageAction)
	mux.HandleFunc("/v1/spaces/init", s.handleSpaceInit)
	mux.HandleFunc("/v1/spaces/config", s.handleSpaceConfig)
	mux.HandleFunc("/v1/spaces/migrate", s.handleSpaceMigrate)
	mux.HandleFunc("/v1/audit", s.handleAudit)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
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
