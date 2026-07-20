#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="$ROOT_DIR/apps/api"
TEST_PATTERN='^(TestIdentitySessionPolicyPostgres(LifecycleReplayAndInvalidation|LifecycleAllowsDeletedActorProvenance|LifecycleUpgradeAssociationAndRollback|LifecycleFailureRollsBackRegistryAndSelection|LifecycleSelectRace|SelectFirstLifecycleRace|EffectDoesNotStarveHostPool|ConcurrentEffectsDelayWriterUntilAllReturn|NonCooperativeEffectRetainsAdmissionAfterCancellation|EffectSerializesSelectionMutations|EffectSerializesLifecycleInvalidation|EffectSerializesRegistryAuthorityMutations|EffectHoldsExactArtifactAndUserRows|EffectRejectsInactiveUser|EffectRejectsStaleResolutionWithoutMutation|EffectAllowsUnrelatedRegistryDrift|CanceledEffectWaitersReleaseAdmission|EffectExitReleasesAdmission)|TestIdentitySessionPolicyEffectGate(PrefersWriter|WaitIsCancelable)|TestIdentitySessionPolicyEffectContextRejectsMutationReentry|TestP7IdentitySessionAuthorityPostgresJoinedMatrix)$'
EXPECTED_TESTS=22
REGISTRY_TEST_PATTERN='^(TestRegistrySessionPolicyLease(SerializesAuthorityWriters|ReleasesAfterErrorAndPanic|ClaimsOnlyExactPublicBinding)|TestSessionPolicyMutationGate(RequiresOneSynchronousCallback|RejectsEscapedAndInflightCallbacks|PreservesPanics|TransfersAsyncCallbackPanic|RejectsCanceledAdmission))$'
EXPECTED_REGISTRY_TESTS=8

: "${SFORUM_TEST_DATABASE_URL:?SFORUM_TEST_DATABASE_URL is required}"
if [[ -z "${SFORUM_TEST_DATABASE_URL//[[:space:]]/}" ]]; then
  echo "SFORUM_TEST_DATABASE_URL must not be blank" >&2
  exit 1
fi

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/sforum-identity-session-policy.XXXXXX")"
gocache="${SFORUM_IDENTITY_SESSION_POLICY_GOCACHE:-$tmp_dir/go-build}"

cleanup() {
  local status=$?
  trap - EXIT
  if [[ -d "$tmp_dir" ]]; then
    find "$tmp_dir" -depth -delete
  fi
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

cd "$API_DIR"

test_count="$(
  TMPDIR="$tmp_dir" GOCACHE="$gocache" go test ./app/Models/Identity -list "$TEST_PATTERN" \
    | awk '/^Test/ { count++ } END { print count + 0 }'
)"
if [[ "$test_count" != "$EXPECTED_TESTS" ]]; then
  echo "P7 Session Policy lifecycle gate discovered $test_count tests, want $EXPECTED_TESTS" >&2
  exit 1
fi

race_test_count="$(
  TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -race ./app/Models/Identity -list "$TEST_PATTERN" \
    | awk '/^Test/ { count++ } END { print count + 0 }'
)"
if [[ "$race_test_count" != "$EXPECTED_TESTS" ]]; then
  echo "P7 Session Policy race gate discovered $race_test_count tests, want $EXPECTED_TESTS" >&2
  exit 1
fi

registry_test_count="$(
  TMPDIR="$tmp_dir" GOCACHE="$gocache" go test ./app/Support/IdentityRegistry -list "$REGISTRY_TEST_PATTERN" \
    | awk '/^Test/ { count++ } END { print count + 0 }'
)"
if [[ "$registry_test_count" != "$EXPECTED_REGISTRY_TESTS" ]]; then
  echo "P7 Identity Registry lease gate discovered $registry_test_count tests, want $EXPECTED_REGISTRY_TESTS" >&2
  exit 1
fi

registry_race_test_count="$(
  TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -race ./app/Support/IdentityRegistry -list "$REGISTRY_TEST_PATTERN" \
    | awk '/^Test/ { count++ } END { print count + 0 }'
)"
if [[ "$registry_race_test_count" != "$EXPECTED_REGISTRY_TESTS" ]]; then
  echo "P7 Identity Registry race lease gate discovered $registry_race_test_count tests, want $EXPECTED_REGISTRY_TESTS" >&2
  exit 1
fi

TMPDIR="$tmp_dir" GOCACHE="$gocache" go test ./app/Models/Identity \
  -run "$TEST_PATTERN" -count=3 -timeout=5m
TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -race ./app/Models/Identity \
  -run "$TEST_PATTERN" -count=3 -timeout=5m
TMPDIR="$tmp_dir" GOCACHE="$gocache" go test ./app/Support/IdentityRegistry \
  -run "$REGISTRY_TEST_PATTERN" -count=3 -timeout=5m
TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -race ./app/Support/IdentityRegistry \
  -run "$REGISTRY_TEST_PATTERN" -count=3 -timeout=5m
