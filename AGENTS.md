# SearchLens repository instructions

## Global workflow

- Prefix every shell command with `rtk`, including Git, Go, and npm commands.
- Use the `caveman` skill for concise conversation.
- Use the `ponytail` skill for implementation and review; prefer the smallest complete solution and avoid speculative abstractions.
- Treat each phase in `docs/chapter-1/PLAN.md` as a separate delivery unit.
- Before starting a phase, update local `main`, then create a new phase branch from `main`. Never continue phase work on `main` or reuse the previous phase branch.
- When a phase is complete and its checks pass, commit it on the phase branch, push the branch, and create a pull request targeting `main`.
- Never merge a pull request or commit directly to `main`. The user reviews and merges manually.
- Start the next phase only after the user confirms the previous phase is accepted and `main` contains it. Then branch again from the updated `main`.
- If Git or the remote is not initialized yet, stop before branch, push, or pull-request steps and report the missing prerequisite; do not invent repository or remote details.

## Scope and source of truth

These instructions apply to the entire repository. A closer `AGENTS.md` may add or override rules for its subtree.

Before changing product behavior, read the Chapter 1 documents:

- `docs/chapter-1/PRD.md` defines scope and acceptance criteria.
- `docs/chapter-1/ARCHITECTURE.md` defines the approved technical shape and API contract.
- `docs/chapter-1/PLAN.md` defines implementation order and exit gates.

When they disagree, use PRD requirements first, then Architecture, then Plan. Update the documents when an agreed product decision changes; do not silently implement a different design.

## Repository map

- `apps/frontend`: existing Next.js App Router UI using React, TypeScript, and Tailwind CSS 4.
- `apps/backend`: Go/Gin API and corpus importer. This directory may be empty until backend implementation starts.
- `docs/repository-data`: starter corpus JSON files. More degree programs may be added with the same record shape.
- `docs/chapter-1`: current product, architecture, and delivery documents.
- `docs/tmp`: temporary design-reference artifacts, not runtime dependencies.

## Current product boundary

Chapter 1 is thesis metadata and available-abstract discovery across degree programs.

- Do not claim access to restricted PDFs or chapter-level content.
- Do not add embeddings, LLM generation, reranking, authentication, scraping, or a separate vector database in Chapter 1.
- Preserve each repository `uri` as the authoritative source link.
- Missing abstracts are normal and must be represented honestly, never synthesized.
- Use `divisions` for degree-program identity; do not hard-code Computer Science assumptions.

## Working rules

- Inspect the affected flow and existing callers before editing.
- Preserve unrelated and user-authored changes; never reset or broadly rewrite the working tree.
- Prefer the standard library, native framework features, and existing dependencies.
- Do not add an abstraction, dependency, service, or configuration option without a current requirement.
- Keep changes within the relevant app unless the API contract or product documentation must change too.
- Validate inputs at HTTP, file, and database boundaries. Return useful errors without leaking credentials or full user queries.
- Keep secrets in environment variables and out of source, fixtures, logs, and example values.
- Add the smallest runnable test for non-trivial logic. A change is unfinished if its relevant checks fail.

## Frontend (`apps/frontend`)

- Preserve the existing SearchLens layout and visual language unless the task explicitly changes the design.
- Treat `apps/frontend/DESIGN.md` as the source for colors, typography, spacing, radii, and interaction styling.
- Use App Router, React, TypeScript, Tailwind CSS 4, and existing local assets.
- Prefer server components. Add `"use client"` only where browser state or events require it.
- Do not add a component library, state library, data-fetching library, or icon package for behavior supported by React, CSS, or the platform.
- Keep API response types explicit. Render loading, empty, error, and success states separately.
- Preserve semantic HTML, keyboard operation, visible focus, reduced-motion support, and reasonable touch targets.
- Do not display the backend relevance score as a calibrated percentage.

Run from `apps/frontend` after relevant changes:

```sh
npm run lint
npx tsc --noEmit
npm exec -- next build --webpack
```

## Backend (`apps/backend`)

- Use the architecture in `docs/chapter-1/ARCHITECTURE.md`: one Go module, Gin, PostgreSQL, and `pgx/v5`.
- Keep the API and importer as separate commands in the same module; do not split them into services.
- Use Go's standard library for JSON, logging (`log/slog`), configuration, dates, graceful shutdown, and tests.
- Use Gin binding/validation for request inputs and pgx parameters for all database values.
- Choose sort expressions from a fixed server-side allowlist. Never interpolate client-provided SQL identifiers.
- Keep SQL close to the store code. Do not add an ORM, query generator, repository interface, or migration framework unless repetition demonstrates a need.
- Use explicit `http.Server` timeouts and propagate request contexts to PostgreSQL.
- Keep handlers thin: bind input, call the store operation, and map the result or error to the documented JSON contract.
- Use migrations for schema changes. Do not mutate schema during API startup.

Run from `apps/backend` after relevant changes:

```sh
gofmt -w .
go vet ./...
go test ./...
```

Do not run `go mod tidy` merely to reformat files; run it when imports or module requirements change.

## Corpus and importer

The current files use this record shape:

```text
title, abstract, authors, item_type, subjects, divisions,
depositing_user, date_deposited, uri
```

- Recursively discover `repository_*_data.json` under the configured corpus directory.
- Derive `source_year` from the filename, not `date_deposited`.
- Treat `uri` as the stable unique key and make repeated imports idempotent.
- Require non-empty `title` and `uri`; normalize missing optional fields without inventing values.
- Parse non-empty deposit dates using the documented source format and report invalid values with file and array position.
- Import one file per transaction so a failure cannot leave a partially imported file.
- Do not edit, regenerate, or delete corpus files unless the task explicitly requests data maintenance.
- When new data is added, rerun the corpus audit and expand retrieval evaluation to cover its degree programs.

## Completion reporting

Report changed files, checks run, and any remaining limitation. Do not describe a check as passing unless it was executed successfully.
