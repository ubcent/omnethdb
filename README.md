# OmnethDB

OmnethDB is a versioned knowledge graph for autonomous agents.

We are not building a toy memory layer, a thin vector wrapper, or a vague agent cache. We are building a best-in-class memory primitive for serious agent systems: explicit lineage, typed relations, auditable state transitions, and retrieval that returns the current truth instead of a contaminated blend of history and present.

## Ambition

The standard here is high on purpose.

We want OmnethDB to be:

- the most rigorous embedded memory system in its category
- trustworthy under real agent workflows, not only demos
- explicit where other systems are fuzzy
- inspectable where other systems are opaque
- operationally simple without sacrificing semantic correctness

If a design tradeoff appears between convenience and correctness, we bias toward correctness first and then work to make it ergonomic.

## Product Direction

OmnethDB is being designed as:

- embedded, not service-heavy by default
- versioned, not overwrite-oriented
- governed, not write-anything-and-pray
- retrieval-first for live knowledge
- inspection-friendly for history, audit, and debugging

The architectural source of truth lives in [docs/ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md).

## Planning Stack

The current `v1` planning system lives in [docs/INDEX.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/INDEX.md).

Key documents:

- [docs/ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [docs/SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [docs/CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)
- [docs/UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [docs/BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [docs/MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md)

## Code Layout

The repository now uses a domain-oriented Go layout:

- `internal/memory/` holds the memory model, lifecycle inputs, validation rules, and domain errors
- `internal/policy/` holds governance logic such as writer authorization, trust resolution, and policy normalization
- `internal/store/bolt/` holds the bbolt-backed implementation, transactional lifecycle methods, and storage-level verification tests
- `embedders/hash/` holds the built-in deterministic embedder used by the CLI and examples
- `cmd/omnethdb/` is the current external entrypoint for local and scriptable use
- the root package `omnethdb` is the public facade that re-exports the supported API surface

## Workspace Layout

The embedded operator-facing layout is intentionally simple:

- `config.toml` lives at the workspace root
- persisted data lives at `data/memory.db`

That separation is now reflected in the runtime helpers exposed by the library, so local operation and backup can be reasoned about as a normal workspace copy instead of a hidden process-managed state bundle.

## Quick Start

There is now a real CLI entrypoint:

```bash
go run ./cmd/omnethdb help
```

Create a workspace config:

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

Bootstrap a space:

```bash
go run ./cmd/omnethdb init \
  --workspace . \
  --space repo:company/app
```

Write a memory:

```bash
go run ./cmd/omnethdb remember \
  --workspace . \
  --space repo:company/app \
  --kind static \
  --actor-id user:alice \
  --actor-kind human \
  --content "payments use cursor pagination"
```

Recall live knowledge:

```bash
go run ./cmd/omnethdb recall \
  --workspace . \
  --spaces repo:company/app \
  --query pagination \
  --top-k 5
```

Raw candidate search:

```bash
go run ./cmd/omnethdb candidates \
  --workspace . \
  --space repo:company/app \
  --content pagination \
  --top-k 5
```

Inspect lineage:

```bash
go run ./cmd/omnethdb lineage \
  --workspace . \
  --root <root-memory-id>
```

Inspect explicit relations:

```bash
go run ./cmd/omnethdb related \
  --workspace . \
  --id <memory-id> \
  --relation extends \
  --depth 2
```

Forget a memory:

```bash
go run ./cmd/omnethdb forget \
  --workspace . \
  --id <memory-id> \
  --actor-id user:alice \
  --actor-kind human \
  --reason "obsolete fact"
```

Revive an inactive lineage:

```bash
go run ./cmd/omnethdb revive \
  --workspace . \
  --root <root-memory-id> \
  --kind static \
  --actor-id user:alice \
  --actor-kind human \
  --content "payments use signed cursor pagination"
```

Inspect audit history:

```bash
go run ./cmd/omnethdb audit \
  --workspace . \
  --space repo:company/app
```

Inspect persisted space config:

```bash
go run ./cmd/omnethdb space \
  --workspace . \
  --space repo:company/app
```

Run an embedding migration:

```bash
go run ./cmd/omnethdb migrate \
  --workspace . \
  --space repo:company/app \
  --model-id builtin/hash-embedder-v2 \
  --dimensions 256
```

The CLI currently ships with a deterministic built-in hash embedder so the system is runnable out of the box. Library users can still provide their own `Embedder` implementation directly through the Go API.

Runnable example:

```bash
go run ./examples/basic
```

## MCP Server

OmnethDB also ships with a local stdio MCP server for agent clients such as Claude Code:

```bash
go run ./cmd/omnethdb-mcp --workspace .
```

The current MCP surface is intentionally narrow and maps directly to the existing store contracts:

- `space_init`
- `memory_remember`
- `memory_recall`
- `memory_profile`
- `memory_lineage`
- `memory_related`
- `memory_export_summary`

Example MCP client config:

```json
{
  "mcpServers": {
    "omnethdb": {
      "command": "go",
      "args": ["run", "./cmd/omnethdb-mcp", "--workspace", "/absolute/path/to/workspace"]
    }
  }
}
```

A Claude Code-oriented starter pack lives in [examples/claude-code/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/examples/claude-code/README.md), including a sample `CLAUDE.md` memory policy and an MCP config template.

## HTTP API

You can also run OmnethDB as a local HTTP service:

```bash
go run ./cmd/omnethdb serve --workspace . --addr :8080
```

Bootstrap a space:

```bash
curl -X POST http://localhost:8080/v1/spaces/init \
  -H 'Content-Type: application/json' \
  -d '{"space_id":"repo:company/app"}'
```

Write a memory:

```bash
curl -X POST http://localhost:8080/v1/memories/remember \
  -H 'Content-Type: application/json' \
  -d '{
    "SpaceID":"repo:company/app",
    "Content":"payments use cursor pagination",
    "Kind":1,
    "Actor":{"ID":"user:alice","Kind":0},
    "Confidence":1.0
  }'
```

Recall:

```bash
curl -X POST http://localhost:8080/v1/recall \
  -H 'Content-Type: application/json' \
  -d '{
    "SpaceIDs":["repo:company/app"],
    "Query":"pagination",
    "TopK":5
  }'
```

Inspect lineage:

```bash
curl http://localhost:8080/v1/lineages/<root-memory-id>
```

Forget a memory:

```bash
curl -X POST http://localhost:8080/v1/memories/<memory-id>/forget \
  -H 'Content-Type: application/json' \
  -d '{
    "actor":{"ID":"user:alice","Kind":0},
    "reason":"obsolete fact"
  }'
```

The current HTTP API is intentionally JSON-first and mirrors the store surface closely. It is a practical external boundary for local tools and services now; a stricter versioned API contract can be layered on top later if needed.

## gRPC API

There is now a native gRPC service boundary as well:

```bash
go run ./cmd/omnethdb serve-grpc --workspace . --addr :9090
```

Proto contract:

- [proto/omnethdb/v1/omnethdb.proto](/Users/dmitrybondarchuk/Projects/my/omnethdb/proto/omnethdb/v1/omnethdb.proto)

Generated Go stubs:

- [gen/omnethdb/v1/omnethdb.pb.go](/Users/dmitrybondarchuk/Projects/my/omnethdb/gen/omnethdb/v1/omnethdb.pb.go)
- [gen/omnethdb/v1/omnethdb_grpc.pb.go](/Users/dmitrybondarchuk/Projects/my/omnethdb/gen/omnethdb/v1/omnethdb_grpc.pb.go)

The gRPC surface mirrors the core store contract:

- `Health`
- `GetRuntimeConfig`
- `InitSpace`
- `GetSpaceConfig`
- `Remember`
- `Recall`
- `GetProfile`
- `FindCandidates`
- `Forget`
- `Revive`
- `GetLineage`
- `GetRelated`
- `GetAuditLog`
- `MigrateEmbeddings`

Transport adapters live in:

- [internal/grpcapi/server.go](/Users/dmitrybondarchuk/Projects/my/omnethdb/internal/grpcapi/server.go)
- [internal/httpapi/server.go](/Users/dmitrybondarchuk/Projects/my/omnethdb/internal/httpapi/server.go)

## Working Standard

We are trying to build something best-in-class, which means:

- architecture must stay coherent end to end
- specs must be testable
- every milestone must tie back to UAT
- agent-facing behavior must be predictable, not magical
- operational behavior must be simple enough to trust

This repo should move like a product with a strong point of view, not like a pile of disconnected implementation tasks.
