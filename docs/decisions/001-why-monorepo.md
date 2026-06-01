# ADR 001: Choosing Monorepo Structure for Cozy Canvas

## Status
Accepted (Approved)

## Context
The **Cozy Canvas** project consists of two key components:
1. **Interactive Frontend** (`website/`) — a lightweight SPA application built with HTML5, CSS3, D3.js, and Vite, deploying to Vercel.
2. **REST API Backend** (`backend/`) — a high-performance Go application designed according to the 12-Factor App methodology, interacting with PostgreSQL and MinIO.

We needed to choose a source code storage strategy:
- **Multi-repo**: Separate repositories for frontend and backend.
- **Monorepo**: Storing all components, including infrastructure scripts and migrations, in a single repository.

## Decision
We decided to organize the project as a **monorepo** with the following structure:
- `website/` — frontend.
- `backend/` — Go REST API backend.
- `infrastructure/` — Docker Compose manifests and auxiliary initialization scripts.
- `migrations/` — database SQL migrations.

## Justification
1. **Atomic Changes**:
   When adding new features (e.g., a new entity on the Canvas), changes are required in both the frontend (rendering and API calls) and the backend (data models, handlers, DB migrations). In a monorepo, the entire feature is delivered in a single commit and pull request, eliminating version desynchronization.
2. **Simplified Local Environment**:
   A single `Makefile` in the root of the monorepo allows a developer to spin up the entire infrastructure (DB, object storage), apply migrations, and start the front and back with one simple command.
3. **Unified Versioning and Documentation**:
   All architectural documentation (ADR) and deployment descriptions are stored in one place, simplifying the onboarding of new engineers.
4. **CI/CD Optimization**:
   Modern platforms (Vercel, Render, GitHub Actions) natively support monorepos, allowing builds to be triggered only when files in a specific subfolder change (e.g., deploying the front only when changes occur in the `website/` folder).

## Consequences
- **Pros**:
  - Simplified sharing of settings and documentation code.
  - Fast local development thanks to shared Docker and Makefile.
  - Unified change history in Git.
- **Cons**:
  - Repository size increases due to merging code.
  - Requires a careful `.gitignore` to prevent committing garbage from different ecosystems (Go binaries, `node_modules`, `.env`).
