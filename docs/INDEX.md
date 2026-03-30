# OmnethDB Docs Index

This is the entry point for the `v1` planning and execution stack.

It organizes the documents in this repository from highest-level product contract down to execution-ready tickets.

## Reading Order

Recommended order if you are getting oriented:

1. [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
2. [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
3. [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)
4. [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
5. [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
6. [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md)
7. sprint plans and milestone ticket directories

## Core Planning Stack

### Architecture

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
Purpose:
Source of truth for product and system semantics.

### Spec Mapping

- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
Purpose:
Decomposes `v1` into capability slices and dependency order.

### Capability Mapping

- [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)
Purpose:
Translates internal slices into externally meaningful outcomes.

### UAT Mapping

- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
Purpose:
Defines acceptance scenarios and release-blocking UAT coverage.

### Backlog

- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
Purpose:
Defines the spec-driven backlog with traceability to spec, capability, and UAT.

### Milestones

- [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md)
Purpose:
Groups backlog work into delivery waves with milestone goals and exit criteria.

## Sprint Plans

- [SPRINT_PLAN_M0.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M0.md)
- [SPRINT_PLAN_M1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M1.md)
- [SPRINT_PLAN_M2.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M2.md)
- [SPRINT_PLAN_M3.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M3.md)
- [SPRINT_PLAN_M4.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M4.md)
- [SPRINT_PLAN_M5.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M5.md)
- [SPRINT_PLAN_M6.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M6.md)
- [SPRINT_PLAN_M7.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M7.md)

Purpose:
Translate each milestone into scope, workstreams, execution order, risks, and definition of done.

## Ticketing

### Ticket Template

- [TICKET_TEMPLATE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/TICKET_TEMPLATE.md)
Purpose:
Reusable template for execution-ready tickets.

### Ticket Directories

- [docs/tickets/m0/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m0/README.md)
- [docs/tickets/m1/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m1/README.md)
- [docs/tickets/m2/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m2/README.md)
- [docs/tickets/m3/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m3/README.md)
- [docs/tickets/m4/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m4/README.md)
- [docs/tickets/m5/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m5/README.md)
- [docs/tickets/m6/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m6/README.md)
- [docs/tickets/m7/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m7/README.md)

Purpose:
Contain execution-ready tickets grouped by milestone.

## Suggested Usage

If you are planning:

1. start with [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
2. move through spec, capability, and UAT maps
3. use [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md) and [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md) to decide sequencing

If you are executing:

1. open the relevant sprint plan
2. go to the matching ticket directory
3. work from the tickets in dependency order

If you are doing release validation:

1. start from [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
2. review [SPRINT_PLAN_M7.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M7.md)
3. use [docs/tickets/m7/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m7/README.md) as the release-closure work queue

## Current State

The planning and ticketing stack for `v1` is now fully scaffolded from architecture through release closure.

The next practical move is no longer planning expansion. It is execution, starting with:

- [docs/tickets/m1/M1-001.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m1/M1-001.md)
- [docs/tickets/m1/M1-002.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m1/M1-002.md)
