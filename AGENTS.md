# Agent Guide

This file is the shared working agreement for SForum. Read it before starting any new session.

## Project Intent

SForum is intended to become a maintainable forum project. The current repository is intentionally documentation-first: do not add application code until the stack, module boundaries, and first milestone are agreed.

## Working Principles

- Prefer mature third-party libraries before building custom infrastructure. Before starting a module or feature, briefly check whether established libraries already solve the core problem well.
- Avoid messy, tightly coupled code. Keep responsibilities clear, name things plainly, and make the easy path maintainable.
- Split long implementations across multiple focused files. Avoid placing more than 1000 lines in one file unless there is a strong, documented reason.
- Keep changes scoped to the current task. Do not refactor unrelated areas just because they are nearby.
- Record decisions in the knowledge base when they will matter to future sessions.

## Avoiding Hard-To-Maintain Code

Bad pattern:

- One giant file mixes routing, database queries, validation, rendering, and side effects.
- Feature logic depends on global mutable state without a clear owner.
- Copy-pasted branches handle similar cases with tiny differences.
- A function silently does many things: parses input, checks permission, writes data, sends notifications, and formats a response.

Better pattern:

- Keep routing, business logic, data access, validation, and presentation in separate modules once the codebase needs those boundaries.
- Use small functions with explicit inputs and outputs.
- Extract shared behavior when duplication becomes meaningful, not before.
- Add comments only where they clarify non-obvious decisions.

## Library-First Development

Before implementing a feature, do a short library survey:

1. Define the problem precisely.
2. Search for mature libraries or framework-native solutions.
3. Compare maintenance status, documentation quality, license, ecosystem fit, and complexity.
4. Record the chosen option in `knowledge/decisions/` when the choice affects architecture.

Examples:

- Use a proven authentication/session library instead of hand-rolling password storage and session security.
- Use a migration tool from the selected backend ecosystem instead of inventing a migration format.
- Use a mature rich-text/Markdown sanitizer instead of custom HTML filtering.

## File Size And Module Boundaries

- Aim for cohesive files that are easy to scan.
- Treat 500 lines as a prompt to review structure.
- Treat 1000 lines as a hard warning sign.
- Split by responsibility, not by arbitrary size alone.
- If a large generated file is unavoidable, label it clearly and keep handwritten logic elsewhere.

## Knowledge Base Workflow

The `knowledge/` directory is the project memory. It exists so a new AI session or human contributor can quickly understand where the project stands.

When starting work:

1. Read `knowledge/index.md`.
2. Read the relevant module note under `knowledge/modules/`.
3. Read recent handoffs under `knowledge/sessions/`.
4. Read relevant decisions under `knowledge/decisions/`.

When finishing work:

1. Update `knowledge/index.md` if navigation or project status changed.
2. Add or update module notes when a feature area changes.
3. Add a decision record for important technical/product choices.
4. Add a short session handoff when the next session will need context.

Recommended handoff format:

```md
# YYYY-MM-DD Session Handoff

## Changed

- ...

## Decisions

- ...

## Next

- ...

## Open Questions

- ...
```

## Current Status

- Git repository initialized.
- Documentation and knowledge-base skeleton created.
- First application scaffold has been added under `apps/web` and `apps/api`.
