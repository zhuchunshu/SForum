#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="$ROOT_DIR/apps/api"
TEST_NAME='^TestP7QueryRegistryMutationCacheRestartJoined$'
WORKER_TEST_NAME='^TestProductionQueryWorkerOwnershipAndSafeModeKindJoined$'
QUERY_MATRIX_TEST_NAME='^(TestPlanValidatesShapePaginationCostAndProviders|TestPlanPermissionRecheckIsHostOwned|TestPlanCacheKeyIsolatesActorProvidersAndLocale|TestPlanCacheKeyIsolatesEverySemanticRequestInput|TestExecutionRechecksPermissionBeforeProviderAndRelease|TestExecutionCostAndProviderFailuresFailBeforeRelease|TestExecutionOffsetAndAuthenticatedCursorPagination|TestExecutionCacheIsolationHitAndPoisonFence|TestExecutionCacheHitRechecksPermissionBeforeFinalRelease|TestExecutionCacheFencePreventsStaleProviderRevival|TestExecutionSchemaValidatorRunsAtProviderFilterCacheAndReleaseFences|TestExecutionCacheIsFencedByResolvedProviderMapping)$'
REFERENCE_TEST_NAME='^TestReferenceQueryPluginJoinedGates$'
TRANSACTION_TEST_NAME='^TestPostgresProtocolV2DatabaseRuntimeExactTransactionsAndRevocation$'
LIFECYCLE_TEST_NAME='^(TestProductionQueryProtocolV2ForceDrainJoined|TestProductionQueryLifecycleUpgradeJoined)$'

: "${SFORUM_TEST_DATABASE_URL:?SFORUM_TEST_DATABASE_URL is required}"
command -v docker >/dev/null
command -v nc >/dev/null
command -v openssl >/dev/null
docker info >/dev/null

ownership_token="$(openssl rand -hex 32)"
requested_seed="${SFORUM_QUERY_CACHE_JOINED_SEED:-$(date +%s)-$$}"
seed="${requested_seed}-${ownership_token}"
redis_image="${SFORUM_QUERY_CACHE_TEST_REDIS_IMAGE:-redis:7-alpine}"
container="sforum-query-cache-joined-$$-${ownership_token:0:12}"
tmp_dir=""
binary=""
worker_binary=""
query_binary=""
reference_binary=""
cidfile=""
container_id=""

cleanup() {
  local status=$?
  local cleanup_failed=0
  local attempt=0
  local owned_ids=""
  trap - EXIT
  trap '' INT TERM HUP

  if [[ -n "$binary" && -x "$binary" ]]; then
    local database_cleaned=0
    for attempt in 1 2 3; do
      if SFORUM_QUERY_CACHE_JOINED_PHASE=cleanup \
        SFORUM_QUERY_CACHE_JOINED_SEED="$seed" \
        SFORUM_QUERY_CACHE_JOINED_OWNERSHIP_TOKEN="$ownership_token" \
        TMPDIR="$tmp_dir" \
        "$binary" -test.run "$TEST_NAME" -test.count=1 -test.timeout=3m \
        >"$tmp_dir/database-cleanup.log" 2>&1; then
        database_cleaned=1
        break
      fi
      sleep "$attempt"
    done
    if [[ "$database_cleaned" != "1" ]]; then
      echo "joined Query database cleanup failed (token-bound seed: $seed)" >&2
      sed -n '1,200p' "$tmp_dir/database-cleanup.log" >&2
      cleanup_failed=1
    fi
  fi

  inspect_container_label() {
    local id="$1"
    local error_output=""
    local label=""
    for attempt in 1 2 3; do
      if label="$(docker inspect --format '{{ index .Config.Labels "sforum.query-cache-joined-token" }}' "$id" 2>&1)"; then
        printf '%s' "$label"
        return 0
      fi
      error_output="$label"
      if docker info >/dev/null 2>&1 \
        && [[ "$error_output" == *"No such object"* || "$error_output" == *"No such container"* ]]; then
        return 2
      fi
      sleep "$attempt"
    done
    echo "inspect joined Query Redis container failed: $id: $error_output" >&2
    return 1
  }

  remove_owned_container() {
    local id="$1"
    local inspect_status=0
    local label=""
    if label="$(inspect_container_label "$id")"; then
      :
    else
      inspect_status=$?
      if [[ "$inspect_status" == "2" ]]; then
        return
      fi
      cleanup_failed=1
      return
    fi
    if [[ "$label" != "$ownership_token" ]]; then
      echo "refusing to remove unowned joined Query Redis container: $id" >&2
      cleanup_failed=1
      return
    fi
    for attempt in 1 2 3; do
      if docker rm --force "$id" >/dev/null 2>&1; then
        return
      fi
      if label="$(inspect_container_label "$id")"; then
        :
      else
        inspect_status=$?
        if [[ "$inspect_status" == "2" ]]; then
          return
        fi
      fi
      sleep "$attempt"
    done
    echo "joined Query Redis container cleanup failed after retries: $id" >&2
    cleanup_failed=1
  }

  list_owned_containers() {
    local output=""
    for attempt in 1 2 3; do
      if output="$(docker ps -aq --filter "label=sforum.query-cache-joined-token=$ownership_token" 2>&1)"; then
        printf '%s' "$output"
        return 0
      fi
      sleep "$attempt"
    done
    echo "list joined Query Redis containers failed: $output" >&2
    return 1
  }

  if [[ -n "$cidfile" && -f "$cidfile" ]]; then
    remove_owned_container "$(<"$cidfile")"
  fi
  if owned_ids="$(list_owned_containers)"; then
    for id in $owned_ids; do
      remove_owned_container "$id"
    done
  else
    cleanup_failed=1
  fi

  if [[ -n "$tmp_dir" && -d "$tmp_dir" ]]; then
    if ! find "$tmp_dir" -depth -delete; then
      echo "joined Query temporary directory cleanup failed: $tmp_dir" >&2
      cleanup_failed=1
    fi
  fi
  if [[ "$status" == "0" && "$cleanup_failed" != "0" ]]; then
    status=1
  fi
  exit "$status"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/sforum-query-cache-joined.XXXXXX")"
binary="$tmp_dir/query-cache-joined.test"
worker_binary="$tmp_dir/query-worker-ownership.test"
query_binary="$tmp_dir/query-contract-matrix.test"
reference_binary="$tmp_dir/query-reference-matrix.test"
cidfile="$tmp_dir/redis.cid"
gocache="${SFORUM_QUERY_CACHE_JOINED_GOCACHE:-$tmp_dir/go-build}"

(
  cd "$API_DIR"
  if [[ "${SFORUM_QUERY_CACHE_JOINED_RACE:-0}" == "1" ]]; then
    TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -race -c -o "$binary" ./app/Support/HostAPI
    TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -race -c -o "$worker_binary" ./bootstrap
    TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -race -c -o "$query_binary" ./app/Support/QueryRegistry
    TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -race -c -o "$reference_binary" ./app/Support/Extensions/IntegrationTests
  else
    TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -c -o "$binary" ./app/Support/HostAPI
    TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -c -o "$worker_binary" ./bootstrap
    TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -c -o "$query_binary" ./app/Support/QueryRegistry
    TMPDIR="$tmp_dir" GOCACHE="$gocache" go test -c -o "$reference_binary" ./app/Support/Extensions/IntegrationTests
  fi
)

require_test_count() {
  local test_binary="$1"
  local pattern="$2"
  local expected="$3"
  local label="$4"
  local count=""
  count="$(TMPDIR="$tmp_dir" GOCACHE="$gocache" "$test_binary" -test.list "$pattern" \
    | awk '/^Test/ { count++ } END { print count + 0 }')"
  if [[ "$count" != "$expected" ]]; then
    echo "P7 Query $label gate discovered $count tests, want $expected" >&2
    exit 1
  fi
}

require_test_count "$binary" "$TEST_NAME" 1 "restart"
require_test_count "$worker_binary" "$WORKER_TEST_NAME" 1 "worker ownership"
require_test_count "$query_binary" "$QUERY_MATRIX_TEST_NAME" 12 "contract matrix"
require_test_count "$reference_binary" "$REFERENCE_TEST_NAME" 1 "reference plugin"
require_test_count "$binary" "$TRANSACTION_TEST_NAME" 1 "same-transaction rollback"
require_test_count "$worker_binary" "$LIFECYCLE_TEST_NAME" 2 "lifecycle"

docker run --detach \
  --cidfile "$cidfile" \
  --name "$container" \
  --label sforum.query-cache-joined-test=true \
  --label "sforum.query-cache-joined-token=$ownership_token" \
  --publish 127.0.0.1::6379 \
  "$redis_image" \
  redis-server \
  --appendonly yes \
  --appendfsync always \
  --save "" \
  --maxmemory-policy noeviction \
  >/dev/null
container_id="$(<"$cidfile")"
if [[ -z "$container_id" ]]; then
  echo "joined Query Redis cidfile is empty" >&2
  exit 1
fi

require_container_ownership() {
  local label=""
  label="$(docker inspect --format '{{ index .Config.Labels "sforum.query-cache-joined-token" }}' "$container_id")"
  if [[ "$label" != "$ownership_token" ]]; then
    echo "joined Query Redis container ownership changed: $container_id" >&2
    exit 1
  fi
}

endpoint=""
refresh_endpoint() {
  require_container_ownership
  endpoint="$(docker port "$container_id" 6379/tcp)"
  if [[ ! "$endpoint" =~ ^127\.0\.0\.1:[0-9]+$ ]]; then
    echo "unexpected joined Query Redis endpoint: $endpoint" >&2
    exit 1
  fi
}

wait_for_redis() {
  local ready=0
  local port="${endpoint##*:}"
  require_container_ownership
  for _ in $(seq 1 80); do
    if docker exec "$container_id" redis-cli ping 2>/dev/null | grep -qx PONG \
      && nc -z 127.0.0.1 "$port" 2>/dev/null; then
      ready=1
      break
    fi
    sleep 0.25
  done
  if [[ "$ready" != "1" ]]; then
    echo "joined Query Redis did not become ready (container: $container_id)" >&2
    exit 1
  fi
}

redis_run_id() {
  require_container_ownership
  docker exec "$container_id" redis-cli --raw INFO server \
    | tr -d '\r' \
    | awk -F: '$1 == "run_id" { print $2; exit }'
}

run_phase() {
  local phase="$1"
  local expected_run_id="$2"
  SFORUM_QUERY_CACHE_JOINED_PHASE="$phase" \
  SFORUM_QUERY_CACHE_JOINED_SEED="$seed" \
  SFORUM_QUERY_CACHE_JOINED_OWNERSHIP_TOKEN="$ownership_token" \
  SFORUM_QUERY_CACHE_JOINED_EXPECTED_REDIS_RUN_ID="$expected_run_id" \
  SFORUM_QUERY_CACHE_TEST_REDIS_ADDR="$endpoint" \
  SFORUM_QUERY_CACHE_TEST_REDIS_PASSWORD= \
  TMPDIR="$tmp_dir" \
  GOCACHE="$gocache" \
    "$binary" -test.run "$TEST_NAME" -test.count=1 -test.timeout=3m -test.v
}

run_worker_ownership() {
  local expected_run_id="$1"
  SFORUM_QUERY_CACHE_JOINED_SEED="$seed" \
  SFORUM_QUERY_CACHE_JOINED_OWNERSHIP_TOKEN="$ownership_token" \
  SFORUM_QUERY_CACHE_JOINED_EXPECTED_REDIS_RUN_ID="$expected_run_id" \
  SFORUM_QUERY_CACHE_TEST_REDIS_ADDR="$endpoint" \
  SFORUM_QUERY_CACHE_TEST_REDIS_PASSWORD= \
  TMPDIR="$tmp_dir" \
  GOCACHE="$gocache" \
    "$worker_binary" -test.run "$WORKER_TEST_NAME" -test.count=1 -test.timeout=4m -test.v
}

run_query_contract_matrix() {
  echo "running P7 Query contract matrix"
  TMPDIR="$tmp_dir" GOCACHE="$gocache" \
    "$query_binary" -test.run "$QUERY_MATRIX_TEST_NAME" -test.count=1 -test.timeout=3m -test.v
  TMPDIR="$tmp_dir" GOCACHE="$gocache" \
    "$reference_binary" -test.run "$REFERENCE_TEST_NAME" -test.count=1 -test.timeout=5m -test.v
  TMPDIR="$tmp_dir" GOCACHE="$gocache" \
    "$binary" -test.run "$TRANSACTION_TEST_NAME" -test.count=1 -test.timeout=4m -test.v
  TMPDIR="$tmp_dir" GOCACHE="$gocache" \
    "$worker_binary" -test.run "$LIFECYCLE_TEST_NAME" -test.count=1 -test.timeout=5m -test.v
}

refresh_endpoint
wait_for_redis
seed_run_id="$(redis_run_id)"
if [[ -z "$seed_run_id" ]]; then
  echo "joined Query Redis seed run_id is empty" >&2
  exit 1
fi
run_query_contract_matrix
run_phase seed "$seed_run_id"
run_worker_ownership "$seed_run_id"

require_container_ownership
docker restart "$container_id" >/dev/null
refresh_endpoint
wait_for_redis
verify_run_id="$(redis_run_id)"
if [[ -z "$verify_run_id" || "$verify_run_id" == "$seed_run_id" ]]; then
  echo "joined Query Redis process identity did not change across restart" >&2
  exit 1
fi
run_phase verify "$verify_run_id"
