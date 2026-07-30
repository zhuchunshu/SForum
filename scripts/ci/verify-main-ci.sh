#!/usr/bin/env bash
set -euo pipefail

RELEASE_COMMIT="${1:-}"
WAIT_SECONDS="${SFORUM_CI_WAIT_SECONDS:-15}"
MAX_MISSING_ATTEMPTS="${SFORUM_CI_MAX_MISSING_ATTEMPTS:-20}"
MAX_ATTEMPTS="${SFORUM_CI_MAX_ATTEMPTS:-240}"

if [[ ! "$RELEASE_COMMIT" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Release commit must be a full lowercase Git SHA" >&2
  exit 2
fi
if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
  echo "GITHUB_REPOSITORY is required" >&2
  exit 2
fi
if [[ ! "$WAIT_SECONDS" =~ ^[0-9]+$ ]] \
  || [[ ! "$MAX_MISSING_ATTEMPTS" =~ ^[1-9][0-9]*$ ]] \
  || [[ ! "$MAX_ATTEMPTS" =~ ^[1-9][0-9]*$ ]]; then
  echo "CI wait settings are invalid" >&2
  exit 2
fi

for ((attempt = 1; attempt <= MAX_ATTEMPTS; attempt++)); do
  run_data="$(gh run list \
    --repo "$GITHUB_REPOSITORY" \
    --workflow ci.yml \
    --branch main \
    --event push \
    --commit "$RELEASE_COMMIT" \
    --limit 1 \
    --json databaseId,status,conclusion,url,headSha \
    --jq 'if length == 0 then empty else .[0] | [.databaseId, .status, (if (.conclusion // "") == "" then "pending" else .conclusion end), .url, .headSha] | map(tostring) | join("\u001f") end')"

  if [[ -z "$run_data" ]]; then
    if ((attempt >= MAX_MISSING_ATTEMPTS)); then
      echo "No main CI push run appeared for $RELEASE_COMMIT after $MAX_MISSING_ATTEMPTS checks" >&2
      exit 1
    fi
    echo "Waiting for main CI run for $RELEASE_COMMIT to appear..."
    sleep "$WAIT_SECONDS"
    continue
  fi

  IFS=$'\x1f' read -r run_id run_status run_conclusion run_url run_sha <<< "$run_data"
  if [[ "$run_sha" != "$RELEASE_COMMIT" ]]; then
    echo "Main CI lookup returned unexpected commit $run_sha" >&2
    exit 1
  fi

  if [[ "$run_status" == "completed" ]]; then
    if [[ "$run_conclusion" != "success" ]]; then
      echo "Main CI run $run_id completed with $run_conclusion: $run_url" >&2
      exit 1
    fi
    echo "Verified successful main CI run $run_id: $run_url"
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
      echo "Main CI: $run_url" >> "$GITHUB_STEP_SUMMARY"
    fi
    exit 0
  fi

  echo "Main CI run $run_id is $run_status; waiting: $run_url"
  sleep "$WAIT_SECONDS"
done

echo "Main CI did not complete after $MAX_ATTEMPTS checks" >&2
exit 1
