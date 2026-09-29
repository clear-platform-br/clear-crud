# Napkin Runbook

## Curation Rules

- Re-prioritize on every read.
- Keep recurring, high-value notes only; max 10 items per category.
- Each item includes date + "Do instead".

## Contract Boundaries

1. **[2026-09-28] Never mutate consumer or demo data without explicit user authorization.**
   Do instead: keep database, SQLite, seed and API checks read-only; ask before any insert, update, delete or persistent fixture-data change.
2. **[2026-09-28] Consumer data must never force a core, adapter or renderer change.**
   Do instead: stop and review the generic contract; evolve it only for a reusable behavior, never to fit a specific catalog, dataset or schema.
3. **[2026-09-24] Products consume public CRUD releases; they do not recreate generic CRUD infrastructure.**
   Do instead: require a declared release, adapter and renderer before a product integrates a resource; otherwise record a capability gap.
4. **[2026-09-24] Domain workflow must not leak into the generic capability.**
   Do instead: model only neutral relations, scope, validation, transactions and rendering contracts in clear-crud; keep domain meaning in consumer definitions.
5. **[2026-09-26] Automatic SQLite definitions must not infer security from schema names.**
   Do instead: require server-owned scope mappings and permission gates; inspect schema once at bootstrap only for structural defaults.
5. **[2026-09-26] Automatically generated text fields need finite input bounds.**
   Do instead: use the generic 255-character maximum and set any different, reviewed bound directly in the consumer's `RegisterAutoTenantTable` call.
6. **[2026-09-26] Soft-deleted records are not ordinary inactive records.**
   Do instead: exclude them from normal list, read and mutation paths; only an explicitly declared audit definition may expose them.
7. **[2026-09-26] Adapters translate contracts; they do not decide product behavior.**
   Do instead: put stable CRUD rules in the public core contract and conformance suite, then require every SQLite, PostgreSQL, MySQL or other adapter to implement them.
8. **[2026-09-27] Active-row ordering cannot treat the soft-delete column as an equality prefix.**
   Do instead: require the tenant/equality prefix followed directly by the business sort columns; the conventional `NULL OR 0` archive predicate does not guarantee index order.
9. **[2026-09-27] Every generic CRUD path must be designed for thousands of concurrent users.**
   Do instead: bound inputs and pages, enforce tenant-scoped indexed queries, avoid N+1 and repeated expensive work, use timeouts, and define cache/invalidation behavior before calling a path production-ready.

## Validation

1. **[2026-09-28] Unit tests do not prove the disposable demo can bootstrap against its existing SQLite file.**
   Do instead: after demo/schema/definition edits, run `go run ./cmd/clear-crud-demo` and verify it reaches the listener; do not report tests as operational validation.
2. **[2026-09-24] Validate public contract changes completely.**
   Do instead: run make validate and adapter conformance tests before committing a release-facing change.
3. **[2026-09-29] Read-model adapters must prepare immutable query metadata once.**
   Do instead: precompute aliases, scope order and select lists at bootstrap; keep per-request allocations limited to filters, arguments and returned records.

## Renderer Boundary

1. **[2026-09-25] The standard renderer owns transport state and request concurrency.**
   Do instead: mount `CrudScreen` with the public client; product components provide only route context, theme and message resolution, never parallel CRUD fetching or mutation state.
2. **[2026-09-27] `CrudScreen` is always embeddable content, never the host shell.**
   Do instead: let the consumer own viewport, header, sidebar, footer and global theme; do not add standalone/embed mode switches to the CRUD renderer.
3. **[2026-09-27] Automatic grids default to eight columns and forms have an explicit projection.**
   Do instead: keep the canonical field catalog in `Definition.Fields`, use `Grid.Columns` as the grid allowlist, use `Form.Fields` for form order/projection, and require explicit `WithGridColumns(...)` for another grid projection.
4. **[2026-09-25] Frontend packages are release artifacts, not merely source folders.**
   Do instead: clean generated output, emit declarations, build the bundle and run `npm pack --dry-run` in every validation gate.

## Translation Operations

1. **[2026-09-29] Weblate is the shared translation-management standard for clear*.**
   Do instead: use one self-hosted Docker/Compose instance with separate projects/components, keep Git catalogs as the source of truth, keep Weblate out of runtime, and defer a dedicated server until scale, isolation, compliance or independent maintenance requires it.
