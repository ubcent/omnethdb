package memory

import (
	"errors"
	"regexp"
	"strings"
)

var spaceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_:/-]{1,256}$`)

func ValidateSpaceID(spaceID string) error {
	if !spaceIDPattern.MatchString(spaceID) {
		return ErrInvalidSpaceID
	}
	return nil
}

func ValidateMemoryID(memoryID string) error {
	if strings.TrimSpace(memoryID) == "" {
		return ErrInvalidMemoryID
	}
	return nil
}

func ValidateContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return ErrInvalidContent
	}
	return nil
}

func ValidateMemoryKind(kind MemoryKind) error {
	switch kind {
	case KindEpisodic, KindStatic, KindDerived:
		return nil
	default:
		return ErrInvalidMemoryKind
	}
}

func ValidateActorKind(kind ActorKind) error {
	switch kind {
	case ActorHuman, ActorAgent, ActorSystem:
		return nil
	default:
		return ErrInvalidActorKind
	}
}

func ValidateActor(actor Actor) error {
	if strings.TrimSpace(actor.ID) == "" {
		return ErrInvalidActorID
	}
	if err := ValidateActorKind(actor.Kind); err != nil {
		return err
	}
	return nil
}

func ValidateConfidence(confidence float32) error {
	if confidence < 0 || confidence > 1 {
		return ErrInvalidConfidence
	}
	return nil
}

func ValidateWritersPolicy(policy WritersPolicy) error {
	if policy.MinTrustLevel < 0 || policy.MinTrustLevel > 1 {
		return ErrInvalidWritersPolicy
	}
	return nil
}

func ValidateSpaceWritePolicy(policy SpaceWritePolicy) error {
	if err := validateTrustLevel(policy.HumanTrust); err != nil {
		return errors.Join(ErrInvalidSpaceWritePolicy, err)
	}
	if err := validateTrustLevel(policy.SystemTrust); err != nil {
		return errors.Join(ErrInvalidSpaceWritePolicy, err)
	}
	if err := validateTrustLevel(policy.DefaultAgentTrust); err != nil {
		return errors.Join(ErrInvalidSpaceWritePolicy, err)
	}

	for _, trust := range policy.TrustLevels {
		if err := validateTrustLevel(trust); err != nil {
			return errors.Join(ErrInvalidSpaceWritePolicy, err)
		}
	}

	for _, writers := range []WritersPolicy{
		policy.EpisodicWriters,
		policy.StaticWriters,
		policy.DerivedWriters,
		policy.PromotePolicy,
	} {
		if err := ValidateWritersPolicy(writers); err != nil {
			return errors.Join(ErrInvalidSpaceWritePolicy, err)
		}
	}

	if policy.MaxStaticMemories < 0 || policy.MaxEpisodicMemories < 0 {
		return ErrInvalidSpaceWritePolicy
	}
	if policy.ProfileMaxStatic < 0 || policy.ProfileMaxEpisodic < 0 {
		return ErrInvalidSpaceWritePolicy
	}

	return nil
}

func ValidateSpaceConfig(config SpaceConfig) error {
	if strings.TrimSpace(config.EmbeddingModelID) == "" {
		return ErrInvalidSpaceConfig
	}
	if config.Dimension <= 0 {
		return errors.Join(ErrInvalidSpaceConfig, ErrInvalidDimension)
	}
	if config.DefaultWeight <= 0 {
		return errors.Join(ErrInvalidSpaceConfig, ErrInvalidDefaultWeight)
	}
	if config.HalfLifeDays <= 0 {
		return errors.Join(ErrInvalidSpaceConfig, ErrInvalidHalfLife)
	}
	if err := ValidateSpaceWritePolicy(config.WritePolicy); err != nil {
		return errors.Join(ErrInvalidSpaceConfig, err)
	}
	if config.Migrating {
		if strings.TrimSpace(config.MigrationTargetModelID) == "" {
			return ErrInvalidSpaceConfig
		}
		if config.MigrationTargetDimension <= 0 {
			return errors.Join(ErrInvalidSpaceConfig, ErrInvalidDimension)
		}
	}

	return nil
}

func validateTrustLevel(v float32) error {
	if v < 0 || v > 1 {
		return ErrInvalidTrustLevel
	}
	return nil
}
