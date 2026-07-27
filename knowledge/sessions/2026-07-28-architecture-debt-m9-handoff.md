# 2026-07-28 Architecture Debt M9 Handoff

## Changed

- Extracted `CatalogService`, `LifecycleService`, `ThemeService`, and
  `SettingsService` inside the existing Extensions package.
- Reduced the parent `Service` from 151 receiver methods to 72 public
  compatibility methods that delegate to focused collaborators.
- Moved 79 internal helpers to the unexported shared `serviceCore` rather than
  leaving business logic on the facade.
- Kept the Extensions production file count at 95 by placing facade delegates
  in an existing file.
- Introduced one `themePublicationState` instance for theme activation, asset
  publication, and fail-closed state. Theme owns that instance; lifecycle and
  settings compensation paths share the same pointer and lock order.
- Added receiver caps for all four collaborators and the internal core.

## Decisions

- M9 remains a same-package transition. Collaborators may use the compatibility
  facade for cross-capability calls until M11 introduces stable narrow packages.
- Permission checks stay in the focused actor-aware entry methods; facade
  methods only delegate.
- Runtime-only settings access remains explicitly named
  `ListSettingsForRuntime`; ordinary settings mutations remain actor-aware.

## Evidence

- Full `go test ./app/Models/Extensions` passed.
- Race-focused theme publication and settings/disable serialization tests
  passed.
- Architecture boundary validation passed with 1394 production files scanned.
- `git diff --check` passed.
- Receiver ratchets: Service 72, Catalog 22, Lifecycle 29, Theme 5, Settings
  11, internal core 79.

## Next

- M10 extracts RuntimeSupervisor, InstanceAdmission, RuntimeInvoker, and
  RuntimeEvents/Providers from `Support/Extensions.Manager`.
- Preserve the single runtime-set transition lock, per-extension lifecycle
  admission, exact active-instance selection, resilience state, and provider
  selection authority.
- Keep `Manager` as a compatibility facade and lower its 117-method cap after
  focused tests pass.

## Open Questions

- None. M11 package extraction must remove the remaining same-package back
  references rather than treating the M9 transitional core as a final package
  design.
