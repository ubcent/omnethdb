# OmnethDB V1 Release Evidence

Release date context:

- Evidence recorded on `2026-03-30`
- Execution environment: local repository workspace
- Primary harness: [RELEASE_UAT_HARNESS_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_UAT_HARNESS_V1.md)

Primary execution command:

```bash
go test ./...
```

Observed result:

- `pass`

## Scenario Outcomes

| UAT | Outcome | Evidence | Notes |
| --- | --- | --- | --- |
| `UAT-01` | `pass` | `TestEnsureSpaceBootstrapsAndPersistsConfig`, `TestRememberCreatesRootMemoryAsLatest` | First-write bootstrap and locked embedder state are covered. |
| `UAT-05` | `pass` | `TestRememberUpdateSwitchesLatestWithinLineage`, `TestGetLineageReturnsFullHistoryIncludingForgottenAndRevived` | Latest-switch and retained history are covered. |
| `UAT-07` | `pass` | `TestRememberCreatesDerivedMemoryWithProvenance` | Derived provenance and inspectability are covered. |
| `UAT-10` | `pass` | `TestForgetLatestDeactivatesLineageWithoutRestoringPreviousVersion` | Forget deactivates the live lineage without reverting history. |
| `UAT-12` | `pass` | `TestReviveCreatesNewLatestForInactiveLineage` | Revive creates a new latest instead of unforgetting prior state. |
| `UAT-13` | `pass` | `TestForgetMarksAffectedLiveDerivedMemoryAsOrphanedAndKeepsItLive`, `TestOrphanedSourceFlagIsNotClearedBySourceRevive` | Orphan propagation is explicit and durable. |
| `UAT-14` | `pass` | `TestRecallReturnsOnlyLiveCurrentKnowledge`, `TestRecallDoesNotTraverseRelations` | Live-corpus retrieval contract is covered. |
| `UAT-15` | `pass` | `TestRecallScoresByConfidenceAndTrust`, `TestRecallAppliesEpisodicRecencyAndSpaceWeights` | Full scoring factors and ranking behavior are covered. |
| `UAT-16` | `pass` | `TestGetProfileReturnsSeparateStaticAndEpisodicLayers`, `TestGetProfileUsesIndependentLayerLimits` | Layered profile output and limits are covered. |
| `UAT-19` | `pass` | `TestRememberRejectsUnauthorizedStaticWrite`, `TestRememberAllowsPromotionOnlyWithPromotePolicy` | Kind-specific write governance is covered. |
| `UAT-20` | `pass` | `TestRecallUsesCurrentPolicyTrustAtQueryTime` | Query-time trust behavior is covered. |
| `UAT-21` | `pass` | `TestRememberRejectsRootWriteWhenCorpusLimitReached`, `TestRememberRejectsPromotionWhenStaticCorpusLimitReached`, `TestGetProfileUsesIndependentLayerLimits` | Corpus and profile limits are independently covered. |
| `UAT-24` | `pass` | `TestGetAuditLogReturnsSpaceScopedChronologicalHistory`, `TestInspectionAndAuditCanExplainCurrentCorpusState` | Explainability and audit reconstruction are covered. |
| `UAT-25` | `pass` | `TestRememberRejectsUpdateConflictWhenIfLatestIDIsStale`, `TestConcurrentUpdatesProduceOneWinnerAndOneConflict` | Contested updates reject stale writers cleanly. |
| `UAT-27` | `pass` | `TestBeginEmbeddingMigrationRejectsWritesButKeepsReadsAvailable`, `TestMigrateEmbeddingsReembedsLiveAndHistoricalMemories`, `TestMigrateEmbeddingsCanResumeFromPersistedMigratingState` | Migration lock, full re-embedding, and rerun safety are covered. |

## Residual Risks

- The release gate is strong on semantic correctness and invariants, but it is not a performance certification.
- The current release evidence is built from automated verification coverage rather than a separate operator CLI flow because `cmd/omnethdb` does not exist yet.
- Runtime config is operator-facing and stable for `v1`, but richer operational tooling remains future work.

## Conclusion

The release-blocking `v1` acceptance set currently passes under the defined automated harness.
