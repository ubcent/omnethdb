package memory

import "testing"

func TestValidateSpaceID(t *testing.T) {
	t.Parallel()

	valid := []string{
		"repo:company/app",
		"repo:company/app:agent:scout",
		"repo_company/app-1",
	}
	for _, tc := range valid {
		if err := ValidateSpaceID(tc); err != nil {
			t.Fatalf("ValidateSpaceID(%q) returned unexpected error: %v", tc, err)
		}
	}

	invalid := []string{
		"",
		" ",
		"repo with space",
		"repo.company.app",
	}
	for _, tc := range invalid {
		if err := ValidateSpaceID(tc); err == nil {
			t.Fatalf("ValidateSpaceID(%q) expected error, got nil", tc)
		}
	}
}

func TestValidateActor(t *testing.T) {
	t.Parallel()

	if err := ValidateActor(Actor{ID: "agent:scout-1", Kind: ActorAgent}); err != nil {
		t.Fatalf("ValidateActor returned unexpected error: %v", err)
	}

	if err := ValidateActor(Actor{ID: "", Kind: ActorAgent}); err == nil {
		t.Fatal("ValidateActor expected invalid actor id error")
	}

	if err := ValidateActor(Actor{ID: "agent:scout-1", Kind: ActorKind(99)}); err == nil {
		t.Fatal("ValidateActor expected invalid actor kind error")
	}
}

func TestValidateConfidence(t *testing.T) {
	t.Parallel()

	for _, tc := range []float32{0, 0.5, 1} {
		if err := ValidateConfidence(tc); err != nil {
			t.Fatalf("ValidateConfidence(%v) returned unexpected error: %v", tc, err)
		}
	}

	for _, tc := range []float32{-0.1, 1.1} {
		if err := ValidateConfidence(tc); err == nil {
			t.Fatalf("ValidateConfidence(%v) expected error, got nil", tc)
		}
	}
}

func TestValidateSpaceWritePolicy(t *testing.T) {
	t.Parallel()

	valid := SpaceWritePolicy{
		HumanTrust:        1.0,
		SystemTrust:       1.0,
		DefaultAgentTrust: 0.7,
		TrustLevels: map[string]float32{
			"agent:trusted": 0.9,
		},
		EpisodicWriters:     WritersPolicy{AllowHuman: true, AllowSystem: true, AllowAllAgents: true},
		StaticWriters:       WritersPolicy{AllowHuman: true, AllowSystem: true},
		DerivedWriters:      WritersPolicy{AllowHuman: true, AllowAllAgents: true},
		PromotePolicy:       WritersPolicy{AllowHuman: true},
		MaxStaticMemories:   500,
		MaxEpisodicMemories: 10000,
		ProfileMaxStatic:    50,
		ProfileMaxEpisodic:  10,
	}
	if err := ValidateSpaceWritePolicy(valid); err != nil {
		t.Fatalf("ValidateSpaceWritePolicy returned unexpected error: %v", err)
	}

	invalidTrust := valid
	invalidTrust.DefaultAgentTrust = 1.1
	if err := ValidateSpaceWritePolicy(invalidTrust); err == nil {
		t.Fatal("ValidateSpaceWritePolicy expected trust-level validation error")
	}

	invalidWriter := valid
	invalidWriter.StaticWriters.MinTrustLevel = 1.1
	if err := ValidateSpaceWritePolicy(invalidWriter); err == nil {
		t.Fatal("ValidateSpaceWritePolicy expected writers policy validation error")
	}
}

func TestValidateSpaceConfig(t *testing.T) {
	t.Parallel()

	cfg := SpaceConfig{
		EmbeddingModelID: "openai/text-embedding-3-small",
		Dimension:        1536,
		DefaultWeight:    1.0,
		HalfLifeDays:     30,
		WritePolicy: SpaceWritePolicy{
			HumanTrust:        1.0,
			SystemTrust:       1.0,
			DefaultAgentTrust: 0.7,
		},
	}
	if err := ValidateSpaceConfig(cfg); err != nil {
		t.Fatalf("ValidateSpaceConfig returned unexpected error: %v", err)
	}

	cfg.Dimension = 0
	if err := ValidateSpaceConfig(cfg); err == nil {
		t.Fatal("ValidateSpaceConfig expected invalid dimension error")
	}
}
