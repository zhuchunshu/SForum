#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C
export LANG=C

VERSION_INPUT="${1:-}"
VERSION="${VERSION_INPUT#v}"
ASSET_DIR="${2:-}"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] || [[ ! -d "$ASSET_DIR" ]]; then
  echo "Usage: finalize-release-assets.sh VERSION ASSET_DIR" >&2
  exit 2
fi

EXPECTED=(
  "sforum-cli_${VERSION}_darwin_amd64.tar.gz"
  "sforum-cli_${VERSION}_darwin_arm64.tar.gz"
  "sforum-cli_${VERSION}_linux_amd64.tar.gz"
  "sforum-cli_${VERSION}_linux_arm64.tar.gz"
  "sforum-server_${VERSION}_linux_amd64.tar.gz"
  "sforum-server_${VERSION}_linux_arm64.tar.gz"
)

for asset in "${EXPECTED[@]}"; do
  if [[ ! -s "$ASSET_DIR/$asset" ]]; then
    echo "Missing release asset: $asset" >&2
    exit 1
  fi
done

ACTUAL="$(find "$ASSET_DIR" -maxdepth 1 -type f ! -name SHA256SUMS -exec basename {} \; | sort)"
SORTED_EXPECTED="$(printf '%s\n' "${EXPECTED[@]}" | sort)"
if [[ "$ACTUAL" != "$SORTED_EXPECTED" ]]; then
  echo "Release asset set does not match the expected matrix" >&2
  printf 'Actual:\n%s\n' "$ACTUAL" >&2
  exit 1
fi

for os in darwin linux; do
  for arch in amd64 arm64; do
    cli_root="sforum-cli_${VERSION}_${os}_${arch}"
    cli_listing="$(tar -tzf "$ASSET_DIR/$cli_root.tar.gz")"
    grep -qx "$cli_root/sforum" <<< "$cli_listing" || {
      echo "CLI archive is missing $cli_root/sforum" >&2
      exit 1
    }
  done
done

for arch in amd64 arm64; do
  server_root="sforum-server_${VERSION}_linux_${arch}"
  server_listing="$(tar -tzf "$ASSET_DIR/$server_root.tar.gz")"
  for binary in sforum-api sforum-worker sforum-migrate sforum; do
    grep -qx "$server_root/bin/$binary" <<< "$server_listing" || {
      echo "Server archive is missing $server_root/bin/$binary" >&2
      exit 1
    }
  done
  grep -q "^$server_root/extensions/builtin/" <<< "$server_listing" || {
    echo "Server archive is missing protected built-ins" >&2
    exit 1
  }
done

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$ASSET_DIR" && sha256sum "${EXPECTED[@]}" > SHA256SUMS && sha256sum -c SHA256SUMS)
else
  (cd "$ASSET_DIR" && shasum -a 256 "${EXPECTED[@]}" > SHA256SUMS && shasum -a 256 -c SHA256SUMS)
fi

echo "Release asset matrix and checksums verified for SForum $VERSION"
