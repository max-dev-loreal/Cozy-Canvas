# ADR 003: Graph Relation Consistency in Cozy Canvas

## Status
Accepted (Approved)

## Context
In the Cozy Canvas application, notes and connections form a graph-like database structure:
- Users own notes.
- Users own connections.
- Connections link a source note to a target note.
- Access grants are created between owner and viewer users.

If a user is deleted, or a note that is part of a connection is deleted, any references to those entities must be handled properly. Allowing orphaned records (e.g., a connection referencing a non-existent note) would cause frontend graph rendering failures (physics-based canvas crashes) and backend query errors.

## Decision
We decided to enforce relational integrity strictly at the database level using PostgreSQL **Foreign Keys** with **`ON DELETE CASCADE`** rules:
1. **Notes**: `notes.user_id` references `users.id` with `ON DELETE CASCADE`.
2. **Connections**: 
   - `connections.user_id` references `users.id` with `ON DELETE CASCADE`.
   - `connections.source_note_id` references `notes.id` with `ON DELETE CASCADE`.
   - `connections.target_note_id` references `notes.id` with `ON DELETE CASCADE`.
3. **Access Grants**:
   - `access_grants.owner_user_id` references `users.id` with `ON DELETE CASCADE`.
   - `access_grants.viewer_user_id` references `users.id` with `ON DELETE CASCADE`.

To support fast cascades and queries, we created specific composite and single-column indices:
- `idx_notes_user_id` on `notes(user_id)`
- `idx_connections_user_id` on `connections(user_id)`
- `idx_access_grants_lookup` on `access_grants(owner_user_id, viewer_user_id)`

## Justification
1. **Guaranteed Consistency**:
   Relying on database-level constraints ensures that orphans are physically impossible. PostgreSQL enforces this transactionally, guaranteeing that no connection points to a missing note.
2. **Minimized Application Complexity**:
   Without database-level cascades, the Go backend would need to handle multi-step transaction deletes (e.g., deleting all connections associated with a note before deleting the note itself). This boilerplate increases the risk of bugs and developer overhead.
3. **Efficiency**:
   PostgreSQL executes cascaded deletes natively within a single internal transaction. When combined with proper indices, cascading deletes are highly performant.

## Consequences
- **Pros**:
  - Absolute referential integrity of the canvas graph structure.
  - Simpler, cleaner backend Go codebase without deletion-boilerplates.
  - Automatic cleanup of access grants and files metadata when users or items are removed.
- **Cons**:
  - Cascades are irreversible. Deleting a user deletes everything they own instantly. The application frontend must implement explicit confirmation dialogs to prevent accidental deletions.
