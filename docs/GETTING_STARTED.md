# OmnethDB Getting Started

This guide is for the fastest path from "what is this?" to "I can run it and use it correctly."

If you want the deep product contract, read [ARCHITECTURE.md](./ARCHITECTURE.md) after this guide.

## What You Are Setting Up

An OmnethDB workspace is just a normal directory with two important things:

- `config.toml`: runtime configuration
- `data/memory.db`: persisted state

That is it.
No hidden daemon state. No mystery control plane.

The intended operator path is installed release binaries:

- `omnethdb`
- `omnethdb-mcp`

If you are developing inside this repository, `go run ./cmd/omnethdb ...` and `go run ./cmd/omnethdb-mcp ...` are still fine.

## Step 1: Inspect The CLI

```bash
omnethdb help
```

That shows the supported commands and is the fastest way to see the current operator surface.

## Step 2: Create A Workspace Config

At the root of the workspace, create `config.toml`:

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

What this means:

- `repo:company/app`: the memory namespace for one project
- `default_weight`: how strongly this space should compete in multi-space retrieval
- `half_life_days`: recency decay tuning
- memory limits: guardrails for corpus growth
- profile limits: how much static and episodic context a profile returns
- embedder config: which embedding model defines this space

## Step 3: Initialize The Space

```bash
omnethdb init \
  --workspace . \
  --space repo:company/app
```

This bootstraps the persisted space config and locks the embedder identity for that space.

Important:

- the embedder is not an incidental detail
- writes with a mismatched embedder are rejected
- changing embedder identity later requires an explicit migration

## Step 4: Write Your First Memory

Write a durable project fact:

```bash
omnethdb remember \
  --workspace . \
  --space repo:company/app \
  --kind static \
  --actor-id user:alice \
  --actor-kind human \
  --content "payments use cursor pagination"
```

Good first memories:

- architectural facts
- operational truths that future agents will need
- stable project rules

Bad first memories:

- temporary guesses
- one-off scratch notes
- facts that already exist and should have been updated instead

## Step 5: Query Live Knowledge

```bash
omnethdb recall \
  --workspace . \
  --spaces repo:company/app \
  --query pagination \
  --top-k 5
```

`recall` is for direct knowledge retrieval during work.

If you want a layered context package for agent startup, use `profile` instead:

```bash
omnethdb profile \
  --workspace . \
  --spaces repo:company/app \
  --query pagination \
  --static-top-k 10 \
  --episodic-top-k 5
```

## Step 6: Update Instead Of Duplicating

If a fact changed, do not create a parallel static memory.
Update the lineage explicitly.

First inspect the lineage:

```bash
omnethdb lineage \
  --workspace . \
  --root <root-memory-id>
```

Then write the new version:

```bash
omnethdb remember \
  --workspace . \
  --space repo:company/app \
  --kind static \
  --actor-id user:alice \
  --actor-kind human \
  --update <latest-memory-id> \
  --content "payments use signed cursor pagination"
```

That makes the new memory the latest truth in the lineage.

## Step 7: Inspect History And Relations

Audit trail:

```bash
omnethdb audit \
  --workspace . \
  --space repo:company/app
```

Explicit relations:

```bash
omnethdb related \
  --workspace . \
  --id <memory-id> \
  --relation extends \
  --depth 2
```

Raw candidate search for curation:

```bash
omnethdb candidates \
  --workspace . \
  --space repo:company/app \
  --content "pagination" \
  --top-k 5
```

## Step 8: Use Linting Before Writing

Before writing a candidate memory, you can ask OmnethDB whether it looks like a duplicate or likely update:

```bash
omnethdb lint-remember \
  --workspace . \
  --space repo:company/app \
  --kind static \
  --content "payments use signed cursor pagination"
```

This is a good habit for agent-driven memory writing.

## Step 9: Run Agent Integrations

### MCP

```bash
omnethdb-mcp --workspace .
```

See [examples/claude-code/README.md](../examples/claude-code/README.md) for a Claude Code setup.

### HTTP

```bash
omnethdb serve --workspace . --addr :8080
```

### gRPC

```bash
omnethdb serve-grpc --workspace . --addr :9090
```

## Common Mistakes

- Writing a new top-level memory when `Updates` was the correct move
- Storing temporary observations as `static`
- Using `derived` from fewer than two current sources
- Expecting retrieval to automatically traverse graph relations
- Treating forgotten memory as deleted history

## Recommended Reading After This

1. [CONCEPTS.md](./CONCEPTS.md)
2. [SETUP.md](./SETUP.md)
3. [ARCHITECTURE.md](./ARCHITECTURE.md)
