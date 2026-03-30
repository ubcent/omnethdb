package policy

import "omnethdb/internal/memory"

func DefaultSpaceWritePolicy() memory.SpaceWritePolicy {
	return memory.SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		TrustLevels:       map[string]float32{},
		EpisodicWriters: memory.WritersPolicy{
			AllowHuman:     true,
			AllowSystem:    true,
			AllowAllAgents: true,
		},
		StaticWriters: memory.WritersPolicy{
			AllowHuman:  true,
			AllowSystem: true,
		},
		DerivedWriters: memory.WritersPolicy{
			AllowHuman:     true,
			AllowAllAgents: true,
		},
		PromotePolicy: memory.WritersPolicy{
			AllowHuman: true,
		},
		MaxStaticMemories:   500,
		MaxEpisodicMemories: 10000,
		ProfileMaxStatic:    50,
		ProfileMaxEpisodic:  10,
	}
}

func NormalizeSpaceWritePolicy(policy memory.SpaceWritePolicy) memory.SpaceWritePolicy {
	defaults := DefaultSpaceWritePolicy()

	if policy.HumanTrust == 0 {
		policy.HumanTrust = defaults.HumanTrust
	}
	if policy.SystemTrust == 0 {
		policy.SystemTrust = defaults.SystemTrust
	}
	if policy.DefaultAgentTrust == 0 {
		policy.DefaultAgentTrust = defaults.DefaultAgentTrust
	}
	if policy.TrustLevels == nil {
		policy.TrustLevels = map[string]float32{}
	}

	if isZeroWritersPolicy(policy.EpisodicWriters) {
		policy.EpisodicWriters = defaults.EpisodicWriters
	}
	if isZeroWritersPolicy(policy.StaticWriters) {
		policy.StaticWriters = defaults.StaticWriters
	}
	if isZeroWritersPolicy(policy.DerivedWriters) {
		policy.DerivedWriters = defaults.DerivedWriters
	}
	if isZeroWritersPolicy(policy.PromotePolicy) {
		policy.PromotePolicy = defaults.PromotePolicy
	}

	if policy.MaxStaticMemories == 0 {
		policy.MaxStaticMemories = defaults.MaxStaticMemories
	}
	if policy.MaxEpisodicMemories == 0 {
		policy.MaxEpisodicMemories = defaults.MaxEpisodicMemories
	}
	if policy.ProfileMaxStatic == 0 {
		policy.ProfileMaxStatic = defaults.ProfileMaxStatic
	}
	if policy.ProfileMaxEpisodic == 0 {
		policy.ProfileMaxEpisodic = defaults.ProfileMaxEpisodic
	}

	return policy
}

func ResolveActorTrust(policy memory.SpaceWritePolicy, actor memory.Actor) float32 {
	policy = NormalizeSpaceWritePolicy(policy)

	if trust, ok := policy.TrustLevels[actor.ID]; ok {
		return trust
	}

	switch actor.Kind {
	case memory.ActorHuman:
		return policy.HumanTrust
	case memory.ActorSystem:
		return policy.SystemTrust
	case memory.ActorAgent:
		return policy.DefaultAgentTrust
	default:
		return 0
	}
}

func CanWriteKind(policy memory.SpaceWritePolicy, actor memory.Actor, kind memory.MemoryKind) bool {
	switch kind {
	case memory.KindEpisodic:
		return isActorAllowed(policy.EpisodicWriters, policy, actor)
	case memory.KindStatic:
		return isActorAllowed(policy.StaticWriters, policy, actor)
	case memory.KindDerived:
		return isActorAllowed(policy.DerivedWriters, policy, actor)
	default:
		return false
	}
}

func CanPromote(policy memory.SpaceWritePolicy, actor memory.Actor) bool {
	return isActorAllowed(policy.PromotePolicy, policy, actor)
}

func isActorAllowed(writers memory.WritersPolicy, policy memory.SpaceWritePolicy, actor memory.Actor) bool {
	trust := ResolveActorTrust(policy, actor)
	if trust < writers.MinTrustLevel {
		return false
	}

	switch actor.Kind {
	case memory.ActorHuman:
		if containsString(writers.AllowedAgentIDs, actor.ID) {
			return true
		}
		return writers.AllowHuman
	case memory.ActorSystem:
		if containsString(writers.AllowedAgentIDs, actor.ID) {
			return true
		}
		return writers.AllowSystem
	case memory.ActorAgent:
		if writers.AllowAllAgents {
			return true
		}
		return containsString(writers.AllowedAgentIDs, actor.ID)
	default:
		return false
	}
}

func isZeroWritersPolicy(policy memory.WritersPolicy) bool {
	return !policy.AllowHuman &&
		!policy.AllowSystem &&
		!policy.AllowAllAgents &&
		len(policy.AllowedAgentIDs) == 0 &&
		policy.MinTrustLevel == 0
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
