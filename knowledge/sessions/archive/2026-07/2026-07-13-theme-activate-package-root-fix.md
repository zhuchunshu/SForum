# 2026-07-13 Theme activate package root fix

## Symptom

Admin theme activate showed `extension.preflight_failed` →
「插件预检失败，请检查后端入口文件。」

DB event for Signal Garden:

```text
open .../package.zip/theme.json: not a directory
```

## Root cause

1. **Legacy uploaded themes** store `PackagePath` as `.../package.zip`, with
   extracted content under sibling `files/`.
2. Runtime L0/L1 preflight (`LoadThemePackage` / templates / skin) used
   `PackagePath` as a directory, so it opened `package.zip/theme.json`.
3. Theme preflight failures were wrapped as `ErrPreflightFailed`, whose UI
   copy is plugin-backend oriented (misleading).

## Fix

- `PackageContentRoot(extension)` resolves:
  - digest/builtin snapshot dir as-is
  - legacy zip → `files/` (else parent dir)
- `PageRegistryAdapter` and pages controller asset/template/skin paths use it
- Theme activate preflight/register failures use **`ErrBuildFailed`**
- Soften preflight i18n copy (not “backend entry only”)
- Regression tests: `package_content_root_test.go`
- Local Signal Garden install got L0/L1 files under `files/` for retry

## Verify

```bash
cd apps/api && go test ./app/Models/Extensions/ -run 'PackageContentRoot|ActivateThemeLegacyZip' -count=1
```

Nocturne builtin syncs on API restart (`sforum.nocturne-theme`).

## Operator

Retry activate **Nocturne Harbor** or **Signal Garden** after API reload.
If theme preflight still fails, message should now be theme-oriented
(`extension.build_failed`), not backend-entry.