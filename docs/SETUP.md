# OmnethDB Setup

This guide explains how to configure a real workspace without guessing what each setting does.

## Workspace Layout

OmnethDB uses a stable embedded layout:

- `config.toml`
- `data/memory.db`

The runtime configuration lives in the filesystem, not hidden in a service.
The persisted store also lives in the filesystem.

That means backup and restore are operationally simple.

The intended operator path is installed release binaries.
If you are working inside this repository, the equivalent `go run ./cmd/omnethdb ...` commands are still valid for development.

## Minimal Config

```toml
[spaces."repo:company/app"]
default_weight = 1.0
half_life_days = 30
max_static_memories = 500
max_episodic_memories = 10000
profile_max_static = 20
profile_max_episodic = 10

[spaces."repo:company/app".embedder]
model_id = "builtin/hash-embedder-v1"
dimensions = 256
```

## Field Reference

### Space-level retrieval tuning

- `default_weight`: default score multiplier for the space during multi-space retrieval
- `half_life_days`: recency decay horizon

Use lower `default_weight` for noisy run-scoped spaces.

### Corpus limits

- `max_static_memories`
- `max_episodic_memories`

These are governance guardrails, not soft suggestions.

### Profile limits

- `profile_max_static`
- `profile_max_episodic`

These control how much context `profile` returns by default for that space.

### Trust tuning

- `human_trust`
- `system_trust`
- `default_agent_trust`

These affect retrieval ranking and policy evaluation.

### Writer policy tables

Available sections:

- `[spaces."<space-id>".episodic_writers]`
- `[spaces."<space-id>".static_writers]`
- `[spaces."<space-id>".derived_writers]`
- `[spaces."<space-id>".promote_policy]`

Available keys inside those sections:

- `allow_human`
- `allow_system`
- `allow_all_agents`
- `allowed_agent_ids`
- `min_trust_level`

Example:

```toml
[spaces."repo:company/app".static_writers]
allow_human = true
allow_system = true
allow_all_agents = false
allowed_agent_ids = ["agent:claude"]
min_trust_level = 0.3
```

This means:

- humans may write static memories
- system may write static memories
- agents are not allowed by default
- one named agent is explicitly allowed
- actors below trust `0.3` are rejected

## Embedder Configuration

Every space has an embedding identity:

```toml
[spaces."repo:company/app".embedder]
model_id = "builtin/hash-embedder-v1"
dimensions = 256
```

Important behavior:

- the first successful bootstrap locks this identity into persisted space config
- later writes must match it
- changing it requires `migrate`

This is deliberate.
Mixed embedding identity inside one space would make retrieval semantics unreliable.

## Default Policy Shape

Default policy behavior is:

- `human_trust = 1.0`
- `system_trust = 1.0`
- `default_agent_trust = 0.7`
- episodic writers: humans, system, all agents
- static writers: humans and system
- derived writers: humans and all agents
- promote policy: humans
- `max_static_memories = 500`
- `max_episodic_memories = 10000`
- `profile_max_static = 50`
- `profile_max_episodic = 10`

If you do not override a field, the default remains in effect.

## Bootstrap Flow

1. Define the space in `config.toml`
2. Run:

```bash
omnethdb init --workspace . --space repo:company/app
```

3. Inspect persisted config:

```bash
omnethdb space --workspace . --space repo:company/app
```

## Validating Config Changes

If you changed `config.toml` after a space already exists, use the config reconciliation commands:

Validate:

```bash
omnethdb space validate-config --workspace . --space repo:company/app
```

Show the diff:

```bash
omnethdb space diff-config --workspace . --space repo:company/app
```

Apply the new config when valid:

```bash
omnethdb space apply-config --workspace . --space repo:company/app
```

This is the safe path for operator-visible policy and tuning changes.

## Quality And Curation Commands

OmnethDB also includes advisory commands for corpus hygiene:

- `quality`
- `quality-plan`
- `quality-report`
- `synthesis-candidates`
- `promotion-suggestions`
- `forget-batch`

These do not replace human judgment.
They surface explicit review opportunities.

## Example: Slightly More Governed Config

```toml
[spaces."repo:company/app"]
default_weight = 1.0
half_life_days = 30
max_static_memories = 500
max_episodic_memories = 10000
profile_max_static = 25
profile_max_episodic = 10
human_trust = 1.0
system_trust = 1.0
default_agent_trust = 0.5

[spaces."repo:company/app".static_writers]
allow_human = true
allow_system = true
allow_all_agents = false
allowed_agent_ids = ["agent:curator-1"]
min_trust_level = 0.7

[spaces."repo:company/app".derived_writers]
allow_human = true
allow_system = false
allow_all_agents = true
min_trust_level = 0.6

[spaces."repo:company/app".embedder]
model_id = "builtin/hash-embedder-v1"
dimensions = 256
```

## Recommended Operator Habits

- keep one stable project-level space per repository
- only create additional spaces when they solve a real isolation problem
- treat embedder changes as migrations, not casual edits
- run `lint-remember` before automated writes
- use `space diff-config` before applying config changes to an existing space

## Next Step

Return to [README.md](../README.md) or continue into [INDEX.md](./INDEX.md).
