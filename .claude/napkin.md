# Napkin Runbook

## Curation Rules

- Re-prioritize on every read.
- Keep recurring, high-value notes only; max 10 items per category.
- Each item includes date + "Do instead".

## Contract Boundaries

1. **[2026-09-24] Products consume public CRUD releases; they do not recreate generic CRUD infrastructure.**
   Do instead: require a declared release, adapter and renderer before a product integrates a resource; otherwise record a capability gap.
2. **[2026-09-24] Domain workflow must not leak into the generic capability.**
   Do instead: model only neutral relations, scope, validation, transactions and rendering contracts in clear-crud; keep domain meaning in consumer definitions.

## Validation

1. **[2026-09-24] Validate public contract changes completely.**
   Do instead: run make validate and adapter conformance tests before committing a release-facing change.

## Renderer Boundary

1. **[2026-09-25] The standard renderer owns transport state and request concurrency.**
   Do instead: mount `CrudScreen` with the public client; product components provide only route context, theme and message resolution, never parallel CRUD fetching or mutation state.
2. **[2026-09-25] Frontend packages are release artifacts, not merely source folders.**
   Do instead: clean generated output, emit declarations, build the bundle and run `npm pack --dry-run` in every validation gate.
