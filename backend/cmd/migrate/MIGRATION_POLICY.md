# Schema Migration Policy

This policy governs migrations for the supported unified serverless deployment.
It protects request-serving Worker invocations and database-connected
notification functions that may still run the previous application revision
while the Bootstrap Lambda applies a migration.

## Immutable baseline

Migration `000035` is the grandfathered metadata baseline. Migrations through
that version predate this policy and remain immutable: do not edit their SQL,
rename their files, or add retrospective claims that they were online-safe.

Every migration after `000035` must have one matching entry in
`migrations/manifest.json`. Adding SQL without its manifest entry is an invalid
migration set.

## Expand and contract

Use separate releases for changes that alter an existing application contract:

1. Expand the schema with an additive shape the deployed application can
   ignore or use safely.
2. Deploy application code that works with both the old and expanded shapes.
3. Backfill or validate existing rows with a bounded, retry-safe operation.
4. Contract only in a later release, after the currently deployed application
   no longer reads or writes the old shape.

A later contract migration may be marked `online` only when the application
revision deployed before that migration already tolerates the contracted
schema. A rename is therefore normally an add/copy/dual-read/remove sequence,
not one `ALTER TABLE ... RENAME` operation.

Normal deployment must reject a migration that needs downtime, caller
draining, or an operator override. Such a migration requires a separately
designed and approved maintenance workflow; the first policy implementation
does not provide a generic bypass.

## Manifest contract

`migrations/manifest.json` is the shared, machine-readable review contract for
the deployer, artifact builder, and Bootstrap runtime. It has this shape:

```json
{
  "schemaVersion": 1,
  "baselineVersion": 35,
  "migrations": [
    {
      "version": 36,
      "name": "example_expansion",
      "categories": ["additive"],
      "deployment": "online",
      "backfill": {
        "mode": "none",
        "resumable": false,
        "notes": "No existing rows are rewritten."
      },
      "locking": {
        "risk": "low",
        "notes": "Metadata-only addition; lock acquisition is bounded."
      },
      "rollback": {
        "application": "compatible",
        "schema": "not_required",
        "dataLossRisk": false,
        "notes": "The previous application ignores the added shape."
      }
    }
  ]
}
```

Allowed `categories` values are:

- `additive`: adds an independently usable object or nullable/default-safe
  shape without invalidating existing reads or writes;
- `backfill`: populates or repairs existing rows;
- `constraint_tightening`: narrows values or relationships accepted by the
  database;
- `data_rewrite`: changes the stored representation of existing values;
- `rename`: renames an object consumed by application code;
- `drop`: removes an object or accepted representation;
- `type_change`: changes a column type or its interpretation;
- `index`: creates, replaces, or removes an index.

Categories may be combined. `deployment` is either `online` or
`maintenance_required`. The supported normal deploy path accepts only
`online` entries.

`backfill.mode` is `none` or `bounded`. A bounded backfill must be resumable and
its notes must state the batching or finite-row invariant. Both `backfill` and
`data_rewrite` categories require bounded mode. `locking.risk` is
`low`, `medium`, or `high` and must explain the relevant PostgreSQL lock or
table-rewrite behavior. `rollback.application` is `compatible` or
`follow_up_required`; `rollback.schema` is `not_required`, `manual_only`, or
`unsafe`.

The manifest is an engineering assertion, not proof that arbitrary SQL is
safe. Automated checks may conservatively reject destructive SQL declared as a
simple additive migration, but reviewers remain responsible for PostgreSQL
semantics, old-application behavior, and data correctness.

## Online execution limits

The Bootstrap Lambda has a 300-second execution limit. Normal migrations use a
5-second PostgreSQL `lock_timeout` and a 240-second `statement_timeout`, leaving
time for connection, role/grant reconciliation, error reporting, and cleanup.
A migration that cannot meet those limits is not suitable for the normal
deployment path.

Before accepting an online migration, review:

- the strongest lock each statement takes and whether active Worker or
  Delivery queries need a conflicting lock;
- whether PostgreSQL rewrites a table or scans all rows;
- the maximum rows touched and the behavior when the operation is retried;
- whether constraints can be added without immediate validation and validated
  in a later bounded step;
- whether index work needs an online PostgreSQL strategy and whether that
  strategy is compatible with the migration transaction behavior.

Do not place an unbounded whole-table data rewrite in a normal deployment.
Large data changes must use a separately bounded, resumable backfill before a
later validation or contract migration.

## Deployment and failure rules

The `migrations`, `backend`, and `all` update scopes can apply migrations and
must enforce the same policy. Compatibility preflight must finish before any
remote or database mutation, including publishing Bootstrap code or pausing
the notification Sender. Artifact construction and direct Bootstrap invocation
must validate the same packaged manifest again.

The deployer performs the repository preflight and prints the manifest summary.
When the manifest contains a maintenance-required entry, an update first asks
the currently deployed Bootstrap for its migration state through the read-only
`migration-state` operation. Only maintenance-required entries newer than that
database version are rejected, so a separately approved maintenance operation
does not permanently block later normal deployments. Deploy this state-inspection
capability before introducing the first post-baseline maintenance-required entry.

The artifact builder rejects an invalid migration set and packages the manifest
beside the SQL. Bootstrap validates that packaged copy and, when maintenance
history exists, reads the database migration version and dirty flag before
database preparation. It rejects dirty or unsafe pending versions before
calling `Up()`.

Migration failure must leave the Worker release unchanged. The notification
Sender must not be paused until migrations succeed; if it is paused for later
backend publication and that publication fails, restore its previous
concurrency before returning the failure.

Errors may report the migration version, dirty-state flag, violated policy,
and recovery category. They must not expose a database URL, credentials, SQL
parameters, application rows, or protected runtime configuration.

## Rollback and recovery

Application rollback and schema rollback are different operations. The normal
rollback is to redeploy application code that remains compatible with the
expanded schema. Serverless deployment never automatically runs `down`, a
negative `step`, `migrate`, or `force`.

Those commands in the local migration CLI are operator recovery tools. A
`.down.sql` file documents a possible reverse transformation; its existence
does not prove that reversal preserves data or is safe in production.

Use these recovery decisions:

- **Preflight rejection:** correct the manifest or redesign the migration; no
  remote state should have changed.
- **Lock or statement timeout:** the secret-safe failure reason distinguishes
  `migration_lock_timeout`, `lock_timeout`, and `statement_timeout`. Leave the
  Worker active, identify the conflicting workload or redesign the operation,
  then retry the same bounded migration.
- **Dirty migration state:** stop automated retries, inspect the recorded
  version and database state, restore from a verified backup when necessary,
  and use `force` only after an operator has established the exact schema state.
- **Failed backfill:** preserve resumability markers or finite batching, repair
  the cause, and retry without duplicating or corrupting data.
- **Migration succeeded but later Bootstrap work failed:** treat the schema as
  expanded, keep or restore the compatible old application, correct the later
  failure, and rerun the idempotent Bootstrap path.

Never rewrite an applied migration to recover an environment. Add a new
forward migration or use an explicitly reviewed recovery procedure.

## Compatibility test boundary

The migration-policy tests execute representative legacy query and write
contracts through an expand, bounded backfill, constraint validation, and later
contract sequence. They also exercise the real migration runner's timeout and
dirty-state behavior against PostgreSQL. These are contract fixtures, not an
execution of a historical Worker binary. Full previous-release binary testing
requires immutable release artifacts and remains outside this policy's current
verification boundary.
