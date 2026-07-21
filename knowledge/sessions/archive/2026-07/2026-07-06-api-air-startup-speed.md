# 2026-07-06 API Air Startup Speed

## Changed

- Updated API and worker Air configs to use `build.entrypoint` instead of the
  deprecated `build.bin` field.
- Limited Air watch roots to source, bootstrap, config, command, and database
  directories.
- Excluded runtime attachment storage and generated SQLC output from Air
  watching.
- Ignored local attachment runtime data in `apps/api/.gitignore`.
- Moved the misplaced local attachment file from
  `apps/api/app/Models/Attachments/storage/...` to
  `apps/api/storage/app/attachments/...`.

## Decisions

- Keep this as a low-risk development-environment fix; do not split optional
  storage or queue dependencies from the API binary in this pass.
- Treat first-run API compilation after Go version or cache changes as expected
  cold-build cost.

## Next

- If cold API builds remain painful, plan a separate dependency-slimming pass
  for optional attachment storage adapters such as Aliyun OSS, Tencent COS,
  FTP, and SFTP.

## Verification

- Hot-cache `go build -o ./tmp/sforum-api ./cmd/api` previously measured about
  0.47 seconds.
- Temporary cold-cache API build previously measured about 38.68 seconds.
- Air v1.65.1 generated default config confirms `entrypoint = ["./tmp/main"]`
  is the current field shape.
