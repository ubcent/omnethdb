package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"omnethdb/internal/memory"
	"omnethdb/internal/policy"
	storebolt "omnethdb/internal/store/bolt"
)

type Layout struct {
	RootDir    string
	ConfigPath string
	DataDir    string
	DataPath   string
}

type Config struct {
	Spaces map[string]SpaceSettings `toml:"spaces"`
}

type SpaceSettings struct {
	DefaultWeight       *float32              `toml:"default_weight"`
	HalfLifeDays        *float32              `toml:"half_life_days"`
	MaxStaticMemories   *int                  `toml:"max_static_memories"`
	MaxEpisodicMemories *int                  `toml:"max_episodic_memories"`
	ProfileMaxStatic    *int                  `toml:"profile_max_static"`
	ProfileMaxEpisodic  *int                  `toml:"profile_max_episodic"`
	HumanTrust          *float32              `toml:"human_trust"`
	SystemTrust         *float32              `toml:"system_trust"`
	DefaultAgentTrust   *float32              `toml:"default_agent_trust"`
	Embedder            RuntimeEmbedderConfig `toml:"embedder"`
}

type RuntimeEmbedderConfig struct {
	ModelID    string `toml:"model_id"`
	Dimensions int    `toml:"dimensions"`
}

func ResolveLayout(root string) (Layout, error) {
	if strings.TrimSpace(root) == "" {
		return Layout{}, memory.ErrInvalidContent
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Layout{}, err
	}
	return Layout{
		RootDir:    absRoot,
		ConfigPath: filepath.Join(absRoot, "config.toml"),
		DataDir:    filepath.Join(absRoot, "data"),
		DataPath:   filepath.Join(absRoot, "data", "memory.db"),
	}, nil
}

func OpenWorkspace(root string) (*storebolt.Store, Layout, error) {
	layout, err := ResolveLayout(root)
	if err != nil {
		return nil, Layout{}, err
	}
	if err := os.MkdirAll(layout.DataDir, 0o755); err != nil {
		return nil, Layout{}, err
	}
	store, err := storebolt.Open(layout.DataPath)
	if err != nil {
		return nil, Layout{}, err
	}
	return store, layout, nil
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{Spaces: map[string]SpaceSettings{}}
	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	if cfg.Spaces == nil {
		cfg.Spaces = map[string]SpaceSettings{}
	}
	return cfg, nil
}

func (c Config) SpaceInit(spaceID string, fallback storebolt.SpaceInit) storebolt.SpaceInit {
	settings, ok := c.Spaces[spaceID]
	if !ok {
		return fallback
	}

	init := fallback
	if settings.DefaultWeight != nil {
		init.DefaultWeight = *settings.DefaultWeight
	}
	if settings.HalfLifeDays != nil {
		init.HalfLifeDays = *settings.HalfLifeDays
	}

	p := policy.NormalizeSpaceWritePolicy(init.WritePolicy)
	if settings.MaxStaticMemories != nil {
		p.MaxStaticMemories = *settings.MaxStaticMemories
	}
	if settings.MaxEpisodicMemories != nil {
		p.MaxEpisodicMemories = *settings.MaxEpisodicMemories
	}
	if settings.ProfileMaxStatic != nil {
		p.ProfileMaxStatic = *settings.ProfileMaxStatic
	}
	if settings.ProfileMaxEpisodic != nil {
		p.ProfileMaxEpisodic = *settings.ProfileMaxEpisodic
	}
	if settings.HumanTrust != nil {
		p.HumanTrust = *settings.HumanTrust
	}
	if settings.SystemTrust != nil {
		p.SystemTrust = *settings.SystemTrust
	}
	if settings.DefaultAgentTrust != nil {
		p.DefaultAgentTrust = *settings.DefaultAgentTrust
	}
	init.WritePolicy = p

	return init
}

func (c Config) SpaceSettings(spaceID string) (SpaceSettings, bool) {
	settings, ok := c.Spaces[spaceID]
	return settings, ok
}
