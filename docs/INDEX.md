# OmnethDB Docs Index

This is the entry point for the `v1` planning and execution stack.

It organizes the documents in this repository from highest-level product contract down to execution-ready tickets.

## New Here?

If you are trying to understand the product quickly, start with these documents first:

1. [GETTING_STARTED.md](./GETTING_STARTED.md)
2. [CONCEPTS.md](./CONCEPTS.md)
3. [SETUP.md](./SETUP.md)
4. [ARCHITECTURE.md](./ARCHITECTURE.md)

Those four give you the practical and conceptual foundation before you dive into the planning stack.

## Reading Order

Recommended order if you are getting oriented:

1. [ARCHITECTURE.md](./ARCHITECTURE.md)
2. [SPEC_MAP_V1.md](./SPEC_MAP_V1.md)
3. [CAPABILITY_MAP_V1.md](./CAPABILITY_MAP_V1.md)
4. [UAT_MAP_V1.md](./UAT_MAP_V1.md)
5. [BACKLOG_V1.md](./BACKLOG_V1.md)
6. [MILESTONE_PLAN_V1.md](./MILESTONE_PLAN_V1.md)
7. [RELEASE_TRACEABILITY_V1.md](./RELEASE_TRACEABILITY_V1.md)
8. [RELEASE_UAT_HARNESS_V1.md](./RELEASE_UAT_HARNESS_V1.md)
9. [RELEASE_EVIDENCE_V1.md](./RELEASE_EVIDENCE_V1.md)
10. [RELEASE_RECOMMENDATION_V1.md](./RELEASE_RECOMMENDATION_V1.md)
11. sprint plans and milestone ticket directories

## Core Planning Stack

### Architecture

- [ARCHITECTURE.md](./ARCHITECTURE.md)
Purpose:
Source of truth for product and system semantics.

### Onboarding

- [GETTING_STARTED.md](./GETTING_STARTED.md)
Purpose:
Fast path from zero to first correct workspace and first correct write/query flow.

- [CONCEPTS.md](./CONCEPTS.md)
Purpose:
Explains the product model in plain language without sacrificing semantic accuracy.

- [SETUP.md](./SETUP.md)
Purpose:
Explains workspace layout, config, writer policy, embedder identity, and safe config reconciliation.

### Post-V1 Advisory Contracts

- [ADVISORY_CURATION_APIS.md](./ADVISORY_CURATION_APIS.md)
Purpose:
Defines the advisory contract for curator-facing synthesis and promotion review APIs in post-`v1` follow-up work.

- [ADVISORY_EVIDENCE_CONTRACTS.md](./ADVISORY_EVIDENCE_CONTRACTS.md)
Purpose:
Defines the shared evidence contract for curator-facing advisory and diagnostic surfaces in post-`v1` follow-up work.

- [INSPECTOR_EVIDENCE_GUIDELINES.md](./INSPECTOR_EVIDENCE_GUIDELINES.md)
Purpose:
Defines the behavioral guidelines for exposing supporting evidence in inspector diagnostics without weakening semantic boundaries.

- [M10_BENCHMARK_GUIDELINES.md](./M10_BENCHMARK_GUIDELINES.md)
Purpose:
Defines the benchmark methodology for evidence-centered memory quality and curator-review evaluation in post-`v1` follow-up work.

### Spec Mapping

- [SPEC_MAP_V1.md](./SPEC_MAP_V1.md)
Purpose:
Decomposes `v1` into capability slices and dependency order.

### Capability Mapping

- [CAPABILITY_MAP_V1.md](./CAPABILITY_MAP_V1.md)
Purpose:
Translates internal slices into externally meaningful outcomes.

### UAT Mapping

- [UAT_MAP_V1.md](./UAT_MAP_V1.md)
Purpose:
Defines acceptance scenarios and release-blocking UAT coverage.

### Backlog

- [BACKLOG_V1.md](./BACKLOG_V1.md)
Purpose:
Defines the spec-driven backlog with traceability to spec, capability, and UAT.

### Milestones

- [MILESTONE_PLAN_V1.md](./MILESTONE_PLAN_V1.md)
Purpose:
Groups backlog work into delivery waves with milestone goals and exit criteria.

### Release Closure

- [RELEASE_TRACEABILITY_V1.md](./RELEASE_TRACEABILITY_V1.md)
- [RELEASE_UAT_HARNESS_V1.md](./RELEASE_UAT_HARNESS_V1.md)
- [RELEASE_EVIDENCE_V1.md](./RELEASE_EVIDENCE_V1.md)
- [RELEASE_RECOMMENDATION_V1.md](./RELEASE_RECOMMENDATION_V1.md)
Purpose:
Capture release-blocking traceability, repeatable acceptance execution, recorded evidence, and final ship recommendation.

## Sprint Plans

- [SPRINT_PLAN_M0.md](./SPRINT_PLAN_M0.md)
- [SPRINT_PLAN_M1.md](./SPRINT_PLAN_M1.md)
- [SPRINT_PLAN_M2.md](./SPRINT_PLAN_M2.md)
- [SPRINT_PLAN_M3.md](./SPRINT_PLAN_M3.md)
- [SPRINT_PLAN_M4.md](./SPRINT_PLAN_M4.md)
- [SPRINT_PLAN_M5.md](./SPRINT_PLAN_M5.md)
- [SPRINT_PLAN_M6.md](./SPRINT_PLAN_M6.md)
- [SPRINT_PLAN_M7.md](./SPRINT_PLAN_M7.md)
- [SPRINT_PLAN_M8.md](./SPRINT_PLAN_M8.md)
- [SPRINT_PLAN_M9.md](./SPRINT_PLAN_M9.md)
- [SPRINT_PLAN_M10.md](./SPRINT_PLAN_M10.md)

Purpose:
Translate each milestone into scope, workstreams, execution order, risks, and definition of done.

## Ticketing

### Ticket Template

- [TICKET_TEMPLATE.md](./TICKET_TEMPLATE.md)
Purpose:
Reusable template for execution-ready tickets.

### Ticket Directories

- [docs/tickets/m0/README.md](./tickets/m0/README.md)
- [docs/tickets/m1/README.md](./tickets/m1/README.md)
- [docs/tickets/m2/README.md](./tickets/m2/README.md)
- [docs/tickets/m3/README.md](./tickets/m3/README.md)
- [docs/tickets/m4/README.md](./tickets/m4/README.md)
- [docs/tickets/m5/README.md](./tickets/m5/README.md)
- [docs/tickets/m6/README.md](./tickets/m6/README.md)
- [docs/tickets/m7/README.md](./tickets/m7/README.md)
- [docs/tickets/m8/README.md](./tickets/m8/README.md)
- [docs/tickets/m9/README.md](./tickets/m9/README.md)
- [docs/tickets/m10/README.md](./tickets/m10/README.md)

Purpose:
Contain execution-ready tickets grouped by milestone.

## Suggested Usage

If you are planning:

1. start with [ARCHITECTURE.md](./ARCHITECTURE.md)
2. move through spec, capability, and UAT maps
3. use [BACKLOG_V1.md](./BACKLOG_V1.md) and [MILESTONE_PLAN_V1.md](./MILESTONE_PLAN_V1.md) to decide sequencing

If you are executing:

1. open the relevant sprint plan
2. go to the matching ticket directory
3. work from the tickets in dependency order

If you are doing release validation:

1. start from [UAT_MAP_V1.md](./UAT_MAP_V1.md)
2. review [SPRINT_PLAN_M7.md](./SPRINT_PLAN_M7.md)
3. use [RELEASE_TRACEABILITY_V1.md](./RELEASE_TRACEABILITY_V1.md) to map scope to evidence
4. run [RELEASE_UAT_HARNESS_V1.md](./RELEASE_UAT_HARNESS_V1.md)
5. record and review outcomes in [RELEASE_EVIDENCE_V1.md](./RELEASE_EVIDENCE_V1.md)
6. use [RELEASE_RECOMMENDATION_V1.md](./RELEASE_RECOMMENDATION_V1.md) as the release decision record

## Current State

The planning, implementation, and release-closure stack for `v1` is complete.

Post-`v1` follow-up planning now starts in [SPRINT_PLAN_M8.md](./SPRINT_PLAN_M8.md), focused on memory quality, memory curation control, and embedding-assisted hygiene, continues in [SPRINT_PLAN_M9.md](./SPRINT_PLAN_M9.md), focused on explicit curator-review workflows for synthesis and promotion suggestions, and then extends into [SPRINT_PLAN_M10.md](./SPRINT_PLAN_M10.md), focused on evidence-centered curation and evaluation.
