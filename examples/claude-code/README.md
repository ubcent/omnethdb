# Claude Code Integration

This folder contains a practical starter pack for using OmnethDB as durable memory from Claude Code.

It is intentionally opinionated:

- recall before work
- write only when the information has future value
- update existing knowledge instead of duplicating it
- keep episodic, static, and derived memories semantically distinct

## Files

- `mcp-server.json`
  Example MCP server config pointing Claude Code at `omnethdb-mcp`.
- `CLAUDE.md`
  A ready-to-use memory policy prompt for Claude Code.

## Suggested Setup

1. Start OmnethDB MCP against a real workspace:

```bash
go run ./cmd/omnethdb-mcp --workspace /absolute/path/to/omnethdb-workspace
```

2. Register the MCP server in your Claude Code MCP configuration using `mcp-server.json` as a template.

3. Copy or adapt `CLAUDE.md` into the target repository where Claude Code will work.

4. Use one stable OmnethDB `space_id` per repository, for example:

```text
repo:company/app
```

## Recommended Workflow

At task start:

- call `memory_profile` or `memory_recall`
- read current durable knowledge before exploring the repo

During work:

- use `memory_lineage` before updating a fact that may already exist
- use `memory_related` when authoring or auditing explicit relations

At task end:

- write at most a few high-value memories
- prefer `Updates` over duplicate restatements
- only write `Derived` when there are at least two current sources and a real rationale

## What Good Usage Looks Like

Good:

- "payments service now uses signed cursor pagination" as a `static` update to the prior pagination fact
- "deploy failed on 2026-03-31 because smoke test was skipped" as an `episodic` memory
- "all DB migrations require smoke tests before deploy" as a `derived` memory backed by multiple incidents

Bad:

- writing every temporary observation from a single session
- storing guesses as `static`
- creating a new top-level memory when an `Updates` relation was the correct action
- deriving a general rule from only one source memory
