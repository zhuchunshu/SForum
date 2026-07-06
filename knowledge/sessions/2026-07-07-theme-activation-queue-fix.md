# 2026-07-07 Theme Activation Queue Fix

## Changed

- Fixed uploaded theme activation enqueue failures by using a River insert-only
  client in the API process.
- Wrapped River's official migrator in the SForum database migrator so fresh
  databases create `river_job`, `river_queue`, and `river_migration` before
  jobs are enqueued.
- Kept worker execution registration in `bootstrap.NewWorker`; API only
  dispatches jobs.
- Kept local development extension/theme paths aligned with `.env.example` so
  uploads and theme releases default to repository-local storage.

## Decisions

- Do not pass an empty `river.NewWorkers()` bundle to API enqueue clients.
  River treats that as a checked worker bundle and rejects unregistered job
  kinds such as `extension.theme_activate`.
- Use River's built-in migrator instead of hand-writing River's internal schema
  into SForum migrations.

## Verification

- `go test ./...`
- `ruby scripts/validate-openapi-refs.rb`
- `DATABASE_URL=postgres://sforum:sforum@127.0.0.1:15432/sforum?sslmode=disable go test ./database/migrator ./app/Support/Jobs ./bootstrap ./app/Models/Extensions ./app/Http/Controllers/Extensions -count=1`

## Open Questions

- Full `./scripts/test.sh` currently fails on an unrelated admin framework
  validation issue: `/forum/categories` is registered before the related admin
  page change is fully visible to the validator.
