# Decision: Architecture Boundary Guardrails

## Status

Accepted

## Context

The repository has sound top-level layers, but several implementation areas
have accumulated too many unrelated files and responsibilities:

- frontend root directories contain product components, composables, utilities,
  and tests from many domains;
- fixed Core admin tabs are often implemented inline in route pages;
- `Support/Extensions` and `Models/Extensions` have become very large flat Go
  packages;
- several handwritten production files exceed 1000 lines;
- legacy `Manager` and `Service` types continue to attract receiver methods.

Failing every current violation immediately would block unrelated development
and encourage superficial file shuffling. Leaving the guidance unenforced
would allow the debt to continue growing.

## Decision

SForum adopts enforceable, baseline-based architecture guardrails:

1. Root `AGENTS.md` defines mandatory frontend placement, fixed-tab, Go package,
   layer, constructor, and test-boundary rules.
2. `tests/validate-architecture-boundaries.mjs` runs in the full repository test
   gate.
3. New handwritten production files may not exceed 1000 lines. Existing files
   over that limit are recorded at their current line count and may shrink but
   not grow.
4. Crowded frontend roots and the two legacy extension packages have direct-file
   caps. New product work must use a domain subdirectory/package or reduce the
   existing flat surface.
5. Receiver-method counts on the legacy extension runtime `Manager` and major
   domain `Service` God objects may not grow. New capabilities require focused
   collaborators.
6. New fixed Core admin tab pages must import substantial tab components from a
   domain `tabs/` directory. Runtime-defined extension/provider tabs continue
   using generic renderers.
7. Existing inline-tab pages are capped at their current line and literal-branch
   counts until they are decomposed.

The baseline is not a waiver catalog. Increasing it requires a separate
accepted decision record with the reason and a named reduction/removal
condition. Decreasing a measured debt value requires lowering or removing the
matching baseline in the same change, so reclaimed capacity cannot be reused.

## Consequences

- Feature work receives immediate feedback when it increases known
  architectural debt.
- Existing large modules can be reduced incrementally without blocking current
  programs.
- Go refactors must establish responsibility and dependency direction before
  creating subpackages; directory-only reshuffling is insufficient.
- Some legitimate shared primitives may require moving an existing root file or
  an explicitly reviewed baseline decision.
- The first remediation priorities are the site/forum settings pages,
  `bootstrap/api_assembly.go`, and the extension domain/runtime God objects.
