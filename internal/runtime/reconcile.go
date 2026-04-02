package runtime

import (
	"fmt"
	"strings"

	"omnethdb/internal/memory"
	storebolt "omnethdb/internal/store/bolt"
)

type SpaceConfigChange struct {
	Field     string `json:"field"`
	Persisted any    `json:"persisted"`
	Desired   any    `json:"desired"`
	Applyable bool   `json:"applyable"`
	Reason    string `json:"reason,omitempty"`
}

type SpaceConfigReconcile struct {
	SpaceID            string              `json:"space_id"`
	HasRuntimeSettings bool                `json:"has_runtime_settings"`
	Persisted          memory.SpaceConfig  `json:"persisted"`
	Desired            memory.SpaceConfig  `json:"desired"`
	Changes            []SpaceConfigChange `json:"changes"`
	Warnings           []string            `json:"warnings,omitempty"`
	Errors             []string            `json:"errors,omitempty"`
	Applyable          bool                `json:"applyable"`
}

func (c Config) ReconcileSpaceConfig(spaceID string, persisted memory.SpaceConfig) SpaceConfigReconcile {
	out := SpaceConfigReconcile{
		SpaceID:   spaceID,
		Persisted: persisted,
		Desired:   persisted,
		Applyable: true,
	}

	settings, ok := c.SpaceSettings(spaceID)
	out.HasRuntimeSettings = ok
	if !ok {
		out.Warnings = append(out.Warnings, "no runtime config overrides found for this space")
		return out
	}

	init := c.SpaceInit(spaceID, storebolt.SpaceInit{
		DefaultWeight: persisted.DefaultWeight,
		HalfLifeDays:  persisted.HalfLifeDays,
		WritePolicy:   persisted.WritePolicy,
	})
	out.Desired.DefaultWeight = init.DefaultWeight
	out.Desired.HalfLifeDays = init.HalfLifeDays
	out.Desired.WritePolicy = init.WritePolicy

	hasEmbedderOverride := strings.TrimSpace(settings.Embedder.ModelID) != "" || settings.Embedder.Dimensions > 0
	if hasEmbedderOverride {
		if strings.TrimSpace(settings.Embedder.ModelID) == "" || settings.Embedder.Dimensions <= 0 {
			out.Errors = append(out.Errors, "runtime embedder override must set both model_id and dimensions")
			out.Applyable = false
		} else {
			out.Desired.EmbeddingModelID = settings.Embedder.ModelID
			out.Desired.Dimension = settings.Embedder.Dimensions
		}
	}

	out.addChange("embedding_model_id", persisted.EmbeddingModelID, out.Desired.EmbeddingModelID, true, "")
	out.addChange("dimension", persisted.Dimension, out.Desired.Dimension, true, "")
	out.addChange("default_weight", persisted.DefaultWeight, out.Desired.DefaultWeight, true, "")
	out.addChange("half_life_days", persisted.HalfLifeDays, out.Desired.HalfLifeDays, true, "")
	out.addChange("write_policy", persisted.WritePolicy, out.Desired.WritePolicy, true, "")

	if persisted.EmbeddingModelID != out.Desired.EmbeddingModelID || persisted.Dimension != out.Desired.Dimension {
		out.Applyable = false
		reason := "embedder config drift requires explicit embedding migration"
		out.Errors = append(out.Errors, reason)
		out.blockChange("embedding_model_id", reason)
		out.blockChange("dimension", reason)
	}

	if err := memory.ValidateSpaceConfig(out.Desired); err != nil {
		out.Applyable = false
		out.Errors = append(out.Errors, fmt.Sprintf("desired config invalid: %v", err))
	}

	return out
}

func (r *SpaceConfigReconcile) addChange(field string, persisted any, desired any, applyable bool, reason string) {
	if equalConfigValue(persisted, desired) {
		return
	}
	r.Changes = append(r.Changes, SpaceConfigChange{
		Field:     field,
		Persisted: persisted,
		Desired:   desired,
		Applyable: applyable,
		Reason:    reason,
	})
}

func (r *SpaceConfigReconcile) blockChange(field string, reason string) {
	for i := range r.Changes {
		if r.Changes[i].Field != field {
			continue
		}
		r.Changes[i].Applyable = false
		r.Changes[i].Reason = reason
	}
}

func equalConfigValue(a any, b any) bool {
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case float32:
		bv, ok := b.(float32)
		return ok && av == bv
	case memory.SpaceWritePolicy:
		bv, ok := b.(memory.SpaceWritePolicy)
		return ok &&
			av.HumanTrust == bv.HumanTrust &&
			av.SystemTrust == bv.SystemTrust &&
			av.DefaultAgentTrust == bv.DefaultAgentTrust &&
			equalTrustLevels(av.TrustLevels, bv.TrustLevels) &&
			equalWritersPolicy(av.EpisodicWriters, bv.EpisodicWriters) &&
			equalWritersPolicy(av.StaticWriters, bv.StaticWriters) &&
			equalWritersPolicy(av.DerivedWriters, bv.DerivedWriters) &&
			equalWritersPolicy(av.PromotePolicy, bv.PromotePolicy) &&
			av.MaxStaticMemories == bv.MaxStaticMemories &&
			av.MaxEpisodicMemories == bv.MaxEpisodicMemories &&
			av.ProfileMaxStatic == bv.ProfileMaxStatic &&
			av.ProfileMaxEpisodic == bv.ProfileMaxEpisodic
	default:
		return false
	}
}

func equalTrustLevels(a map[string]float32, b map[string]float32) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		if bv, ok := b[k]; !ok || av != bv {
			return false
		}
	}
	return true
}

func equalWritersPolicy(a memory.WritersPolicy, b memory.WritersPolicy) bool {
	if a.AllowHuman != b.AllowHuman || a.AllowSystem != b.AllowSystem || a.AllowAllAgents != b.AllowAllAgents || a.MinTrustLevel != b.MinTrustLevel {
		return false
	}
	if len(a.AllowedAgentIDs) != len(b.AllowedAgentIDs) {
		return false
	}
	for i := range a.AllowedAgentIDs {
		if a.AllowedAgentIDs[i] != b.AllowedAgentIDs[i] {
			return false
		}
	}
	return true
}
