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

## Working Standard

We are trying to build something best-in-class, which means:

- architecture must stay coherent end to end
- specs must be testable
- every milestone must tie back to UAT
- agent-facing behavior must be predictable, not magical
- operational behavior must be simple enough to trust

This repo should move like a product with a strong point of view, not like a pile of disconnected implementation tasks.
