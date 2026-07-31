#!/usr/bin/env bash
set -euo pipefail

REGISTRY="${1:-}"
RELEASE_TAG="${2:-}"

if [[ ! "$REGISTRY" =~ ^ghcr\.io/[A-Za-z0-9_.-]+$ ]]; then
  echo "Registry must look like ghcr.io/owner" >&2
  exit 2
fi
if [[ ! "$RELEASE_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
  echo "Release tag must look like v2.8.0 or v2.8.0-beta.1" >&2
  exit 2
fi

DOCKER_CONFIG="$(mktemp -d "${TMPDIR:-/tmp}/sforum-public-image-pull.XXXXXX")"
export DOCKER_CONFIG
unset DOCKER_AUTH_CONFIG || true

cleanup() {
  rm -rf "$DOCKER_CONFIG"
}
trap cleanup EXIT

images=(
  sforum-api
  sforum-worker
  sforum-migrate
  sforum-web
)

for image in "${images[@]}"; do
  reference="$REGISTRY/$image:$RELEASE_TAG"
  echo "Anonymous pull: $reference"
  docker pull "$reference"
done

echo "All SForum release images are anonymously pullable: $RELEASE_TAG"
