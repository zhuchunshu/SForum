#!/usr/bin/env bash
set -euo pipefail

# Failure-path tests for tests/validate-docs.mjs. Each case builds a minimal
# fixture tree, runs the validator against it with SFORUM_VALIDATE_ROOT, and
# asserts the expected failure message. A positive case proves the base
# fixture passes, so these checks are not only exercised on their happy path.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sforum-validate-docs-test.XXXXXX")"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

fail() {
  printf 'validate-docs_test.sh: %s\n' "$1" >&2
  exit 1
}

write_base_fixture() {
  local dir="$1"
  mkdir -p \
    "$dir/docs/zh-CN/development" \
    "$dir/docs/en-US/development" \
    "$dir/knowledge/modules" \
    "$dir/apps/api/cmd/sforum" \
    "$dir/apps/api/app/Support/Routes" \
    "$dir/contracts/openapi/paths" \
    "$dir/.github/workflows" \
    "$dir/scripts/ci"

  cat > "$dir/README.md" <<'EOF'
# Fixture README

Install: curl -fsSLo sforum-deploy.tar.gz https://github.com/zhuchunshu/SForum/releases/latest/download/sforum-deploy.tar.gz
  set -eu
  awk '$2 == "sforum-deploy.tar.gz" { print }' SHA256SUMS
  test "$(wc -l < sforum-deploy.sha256 | tr -d '[:space:]')" = 1
Prereleases require --channel prerelease.
EOF

  cat > "$dir/docs/README.md" <<'EOF'
# Docs hub
EOF

  for locale in zh-CN en-US; do
    cat > "$dir/docs/$locale/README.md" <<'EOF'
# Home

API smoke test: `POST /auth/register`.
EOF
    cat > "$dir/docs/$locale/deployment.md" <<'EOF'
# Deployment

curl -fsSLo sforum-deploy.tar.gz https://github.com/zhuchunshu/SForum/releases/latest/download/sforum-deploy.tar.gz
  set -eu
  awk '$2 == "sforum-deploy.tar.gz" { print }' SHA256SUMS
  test "$(wc -l < sforum-deploy.sha256 | tr -d '[:space:]')" = 1
  awk '$2 == "upgrade.sh" { print }' SHA256SUMS
Use --channel prerelease for prereleases.
Historical compatibility: the v3.0.0-alpha.13 bundled updater stays pinned.
EOF
    cat > "$dir/docs/$locale/development/cli.md" <<'EOF'
# CLI

version make:plugin make:theme seed:forum seed:perf users:reset-password revisions backfill extension validate extension digest extension test extension package extension docs generate extension list extension disable extension disable-all extension quarantine extension command extension command list extension command run extension api-lts extension system-tier extension system-tier list extension system-tier upsert extension system-tier disable dev:cleanup-orphan-plugins
EOF
    cat > "$dir/docs/$locale/getting-started.md" <<'EOF'
# Getting started

Requires Go 1.26.6+.
EOF
    cat > "$dir/docs/$locale/development/setup.md" <<'EOF'
# Setup

Requires Go 1.26.6+.
EOF
  done

  cat > "$dir/AGENTS.md" <<'EOF'
# Agent guide

Go 1.26.6 anchored by apps/api/go.mod.
EOF

  cat > "$dir/knowledge/modules/backend.md" <<'EOF'
# Backend

Go 1.26.6 toolchain anchor.
EOF

  cat > "$dir/apps/api/go.mod" <<'EOF'
module fixture

go 1.26.6
EOF

  cat > "$dir/apps/api/Dockerfile" <<'EOF'
FROM golang:1.26.6-alpine AS base
EOF

  cat > "$dir/contracts/openapi.yaml" <<'EOF'
openapi: 3.1.0
servers:
  - url: "/api/v1"
paths:
  "/auth/register":
    "$ref": "./openapi/paths/identity.yaml#/authRegister"
EOF

  cat > "$dir/contracts/openapi/paths/identity.yaml" <<'EOF'
authRegister:
  post:
    responses: {}
EOF

  cat > "$dir/apps/api/app/Support/Routes/core_catalog_gen.go" <<'EOF'
package routes

var generatedCoreRouteCatalog = [...]CoreRoute{
	{Method: "POST", Path: "/api/v1/auth/register"},
}
EOF

  cat > "$dir/apps/api/cmd/sforum/command.go" <<'EOF'
package main

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.AddCommand(
		newVersionCommand(),
		newMakeCommand("plugin"),
		newMakeCommand("theme"),
		newSeedCommand(),
		newSeedPerfCommand(),
		newUsersResetPasswordCommand(),
		newRevisionsCommand(),
		newExtensionCommand(),
		newDevCleanupOrphanPluginsCommand(),
	)
	return cmd
}
EOF

  cat > "$dir/apps/api/cmd/sforum/validate.go" <<'EOF'
package main

const _ = `
	Use: "extension"
	Use: "validate [path]"
`
EOF

  cat > "$dir/apps/api/cmd/sforum/manifest_digest.go" <<'EOF'
package main

const _ = `Use: "digest [path]"`
EOF

  cat > "$dir/apps/api/cmd/sforum/test_extension.go" <<'EOF'
package main

const _ = `Use: "test [path]"`
EOF

  cat > "$dir/apps/api/cmd/sforum/package_extension.go" <<'EOF'
package main

const _ = `Use: "package [path]"`
EOF

  cat > "$dir/apps/api/cmd/sforum/docs.go" <<'EOF'
package main

const _ = `Use: "docs"
Use: "generate"`
EOF

  cat > "$dir/apps/api/cmd/sforum/recovery.go" <<'EOF'
package main

const _ = `Use: "list"
Use: "disable <extension-id>"
Use: "disable-all"
Use: "quarantine <extension-id>"`
EOF

  cat > "$dir/apps/api/cmd/sforum/plugin_command.go" <<'EOF'
package main

const _ = `Use: "command"
Use: "list"
Use: "run <command-id>"`
EOF

  cat > "$dir/apps/api/cmd/sforum/api_lts.go" <<'EOF'
package main

const _ = `Use: "api-lts"`
EOF

  cat > "$dir/apps/api/cmd/sforum/system_tier.go" <<'EOF'
package main

const _ = `Use: "system-tier"
Use: "list"
Use: "upsert <extension-id>"
Use: "disable <extension-id>"`
EOF

  cat > "$dir/apps/api/cmd/sforum/revisions.go" <<'EOF'
package main

const _ = `Use: "revisions"
Use: "backfill"`
EOF

  cat > "$dir/.github/workflows/release.yml" <<'EOF'
name: Release
jobs:
  deploy-asset:
    runs-on: ubuntu-latest
    steps:
      - name: Build deploy asset
        run: ./scripts/ci/build-deploy-asset.sh
      - name: Upload deploy asset
        uses: actions/upload-artifact@v4
        with:
          name: release-asset-deploy
EOF

  cat > "$dir/.github/dependabot.yml" <<'EOF'
version: 2
updates:
  - package-ecosystem: "docker-compose"
    directory: "/"
    schedule:
      interval: "weekly"
  - package-ecosystem: "gomod"
    directory: "/apps/api"
    schedule:
      interval: "weekly"
EOF

  cat > "$dir/scripts/ci/finalize-release-assets.sh" <<'EOF'
EXPECTED=(
  "sforum-deploy.tar.gz"
  "upgrade.sh"
)
EOF

  : > "$dir/scripts/ci/build-deploy-asset.sh"
}

run_validator() {
  local dir="$1"
  local output
  set +e
  output="$(SFORUM_VALIDATE_ROOT="$dir" node "$ROOT_DIR/tests/validate-docs.mjs" 2>&1)"
  local status=$?
  set -e
  printf '%s' "$output"
  return "$status"
}

assert_validator_fails_with() {
  local dir="$1" expected="$2" label="$3"
  local output status
  set +e
  output="$(SFORUM_VALIDATE_ROOT="$dir" node "$ROOT_DIR/tests/validate-docs.mjs" 2>&1)"
  status=$?
  set -e
  if [ "$status" -eq 0 ]; then
    fail "$label: validator unexpectedly passed"
  fi
  if [[ "$output" != *"$expected"* ]]; then
    fail "$label: expected failure message '$expected' but got: $output"
  fi
}

# Positive: the base fixture passes.
BASE="$TEMP_DIR/base"
write_base_fixture "$BASE"
if ! output="$(SFORUM_VALIDATE_ROOT="$BASE" node "$ROOT_DIR/tests/validate-docs.mjs" 2>&1)"; then
  fail "base fixture unexpectedly failed: $output"
fi

# Broken local link in an extensions README.
CASE_LINK="$TEMP_DIR/link"
write_base_fixture "$CASE_LINK"
mkdir -p "$CASE_LINK/extensions/builtin/plugins/demo"
cat > "$CASE_LINK/extensions/builtin/plugins/demo/README.md" <<'EOF'
# Demo

See [missing](./does-not-exist.md).
EOF
assert_validator_fails_with "$CASE_LINK" "broken local link" "extensions README link check"

# Stable install snippets must stop on failure and select the checksum entry by
# its exact filename field.
CASE_INSTALL_SAFETY="$TEMP_DIR/install-safety"
write_base_fixture "$CASE_INSTALL_SAFETY"
cat > "$CASE_INSTALL_SAFETY/README.md" <<'EOF'
# Unsafe install

curl -fsSLo sforum-deploy.tar.gz https://github.com/zhuchunshu/SForum/releases/latest/download/sforum-deploy.tar.gz
Use --channel prerelease for prereleases.
EOF
assert_validator_fails_with "$CASE_INSTALL_SAFETY" "stable install commands must be fail-closed" "install verification safety check"

# Unknown CLI constructor registered at the root.
CASE_CLI="$TEMP_DIR/cli"
write_base_fixture "$CASE_CLI"
cat > "$CASE_CLI/apps/api/cmd/sforum/command.go" <<'EOF'
package main

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.AddCommand(
		newVersionCommand(),
		newMysteryCommand(),
	)
	return cmd
}
EOF
assert_validator_fails_with "$CASE_CLI" "unknown command constructor" "CLI constructor check"

# A root command expression that does not use the new*Command constructor
# convention must also fail instead of disappearing from the expected set.
CASE_CLI_UNPARSED="$TEMP_DIR/cli-unparsed"
write_base_fixture "$CASE_CLI_UNPARSED"
cat > "$CASE_CLI_UNPARSED/apps/api/cmd/sforum/command.go" <<'EOF'
package main

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.AddCommand(
		newVersionCommand(),
		existingCommand,
	)
	return cmd
}
EOF
assert_validator_fails_with "$CASE_CLI_UNPARSED" "unparsed AddCommand argument: existingCommand" "CLI unparsed argument check"

# A method shown in docs must exist under that path in OpenAPI.
CASE_HTTP_OPENAPI="$TEMP_DIR/http-openapi"
write_base_fixture "$CASE_HTTP_OPENAPI"
cat >> "$CASE_HTTP_OPENAPI/docs/en-US/README.md" <<'EOF'

Wrong method: `DELETE /auth/register`.
EOF
assert_validator_fails_with "$CASE_HTTP_OPENAPI" "not declared by OpenAPI: DELETE /api/v1/auth/register" "documented OpenAPI method check"

# Matching OpenAPI text is insufficient when the generated Go route catalog
# registers a different method.
CASE_HTTP_GO="$TEMP_DIR/http-go"
write_base_fixture "$CASE_HTTP_GO"
cat > "$CASE_HTTP_GO/apps/api/app/Support/Routes/core_catalog_gen.go" <<'EOF'
package routes

var generatedCoreRouteCatalog = [...]CoreRoute{
	{Method: "DELETE", Path: "/api/v1/auth/register"},
}
EOF
assert_validator_fails_with "$CASE_HTTP_GO" "not registered in the Go route catalog: POST /api/v1/auth/register" "documented Go route method check"

# The generic Docker YAML fetcher filters out Compose manifests; the root entry
# must use the dedicated docker-compose ecosystem.
CASE_DEPENDABOT_COMPOSE="$TEMP_DIR/dependabot-compose"
write_base_fixture "$CASE_DEPENDABOT_COMPOSE"
cat > "$CASE_DEPENDABOT_COMPOSE/.github/dependabot.yml" <<'EOF'
version: 2
updates:
  - package-ecosystem: "docker"
    directory: "/"
  - package-ecosystem: "gomod"
    directory: "/apps/api"
EOF
assert_validator_fails_with "$CASE_DEPENDABOT_COMPOSE" "must use the docker-compose ecosystem" "Dependabot Compose ecosystem check"

# A new governed Go module must not silently fall outside update governance.
CASE_DEPENDABOT_GOMOD="$TEMP_DIR/dependabot-gomod"
write_base_fixture "$CASE_DEPENDABOT_GOMOD"
mkdir -p "$CASE_DEPENDABOT_GOMOD/extensions/builtin/plugins/demo/backend"
cat > "$CASE_DEPENDABOT_GOMOD/extensions/builtin/plugins/demo/backend/go.mod" <<'EOF'
module fixture/demo

go 1.26.6
EOF
assert_validator_fails_with "$CASE_DEPENDABOT_GOMOD" "does not cover governed Go module: /extensions/builtin/plugins/demo/backend/go.mod" "Dependabot Go module coverage check"

# Dockerfile golang base image drifts from go.mod.
CASE_DOCKER="$TEMP_DIR/docker"
write_base_fixture "$CASE_DOCKER"
cat > "$CASE_DOCKER/apps/api/Dockerfile" <<'EOF'
FROM golang:1.26.7-alpine AS base
EOF
assert_validator_fails_with "$CASE_DOCKER" "must match the go.mod toolchain" "Dockerfile Go version check"

# The narrow version rule: a line mentioning a prerelease must NOT exempt an
# unlisted stable-shaped token on the same line.
CASE_VERSION="$TEMP_DIR/version"
write_base_fixture "$CASE_VERSION"
cat >> "$CASE_VERSION/docs/zh-CN/README.md" <<'EOF'

Legacy: v3.0.0-alpha.13 examples may not hide v9.9.9.
EOF
assert_validator_fails_with "$CASE_VERSION" "must not contain the concrete release number v9.9.9" "narrow version allowlist check"

# Unlisted stable-shaped token in rolling docs.
CASE_TOKEN="$TEMP_DIR/token"
write_base_fixture "$CASE_TOKEN"
cat >> "$CASE_TOKEN/README.md" <<'EOF'

Deploy v7.7.7 for the demo.
EOF
assert_validator_fails_with "$CASE_TOKEN" "must not contain the concrete release number v7.7.7" "rolling version token check"

# A stable token that shares the prefix of an allowed historical prerelease is
# still forbidden outside that full prerelease token.
CASE_HISTORICAL_PREFIX="$TEMP_DIR/historical-prefix"
write_base_fixture "$CASE_HISTORICAL_PREFIX"
cat >> "$CASE_HISTORICAL_PREFIX/docs/en-US/README.md" <<'EOF'

Deploy v3.0.0 for the demo.
EOF
assert_validator_fails_with "$CASE_HISTORICAL_PREFIX" "must not contain the concrete release number v3.0.0" "historical prerelease prefix check"

printf 'validate-docs_test.sh: all checks passed\n'
