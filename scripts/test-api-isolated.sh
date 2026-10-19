#!/usr/bin/env bash
# Run database tests on an owned cluster, never on inherited development URLs.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v docker >/dev/null
command -v go >/dev/null
docker info >/dev/null

owner="$(od -An -N12 -tx1 /dev/urandom | tr -d ' \n')"
container="sforum-test-api-${owner}"
image="${SFORUM_TEST_POSTGRES_IMAGE:-postgres:17-alpine}"
# 固定宿主端口让 DATABASE_URL 跨运行逐字节一致。Go 会把测试通过 os.Getenv
# 读到的环境变量值算进结果缓存键，随机端口会让每个数据库包每次运行都重跑
# （实测 45 个包、662 秒）。端口只当缓存键用：集群仍然每次新建并销毁，
# 端口被占用时回退到临时端口，只损失缓存、不损失正确性。
preferred_port="${SFORUM_TEST_POSTGRES_PORT:-55432}"
cid=""
cleanup() {
  local status=$?
  trap - EXIT
  if [[ -n "$cid" ]]; then
    local actual
    if ! actual="$(docker inspect --format '{{index .Config.Labels "com.sforum.test-owner"}}' "$cid")"; then
      echo 'Inspecting isolated test container cleanup failed.' >&2
      exit 1
    fi
    if [[ "$actual" != "$owner" ]]; then
      echo 'Test container ownership changed; cleanup stopped.' >&2
      exit 1
    fi
    if ! docker rm -fv "$cid" >/dev/null; then
      echo 'Removing isolated test container failed.' >&2
      exit 1
    fi
  fi
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# 一次性集群，关掉持久化只影响这台用完即删的测试库（实测三项 GUC 均已生效）。
start_cluster() {
  docker run -d --name "$container" --label "com.sforum.test-owner=$owner" \
    -e POSTGRES_DB=sforum_test_suite -e POSTGRES_USER=sforum -e POSTGRES_PASSWORD=sforum \
    -e POSTGRES_HOST_AUTH_METHOD=scram-sha-256 \
    -p "$1" "$image" \
    -c fsync=off -c synchronous_commit=off -c full_page_writes=off
}

if ! cid="$(start_cluster "127.0.0.1:${preferred_port}:5432" 2>/dev/null)"; then
  docker rm -f "$container" >/dev/null 2>&1 || true
  echo "端口 ${preferred_port} 已被占用，回退临时端口（本轮测试缓存不会命中）。" >&2
  cid="$(start_cluster "127.0.0.1::5432")"
fi
# postgres 官方镜像首次启动会跑 initdb 然后重启一次；pg_isready 在重启前的
# 临时实例上就可能返回 0，紧接着的 psql 会撞上 shutdown。用真实认证连接做
# 就绪判定，并要求连续两次成功，避开首次启动的重启窗口。
ready=0
stable=0
for _ in {1..60}; do
  if docker exec -e PGPASSWORD=sforum "$cid" psql -U sforum -d sforum_test_suite -tAc 'SELECT 1' >/dev/null 2>&1; then
    stable=$((stable + 1))
    if [[ "$stable" == 2 ]]; then
      ready=1
      break
    fi
  else
    stable=0
  fi
  sleep 1
done
if [[ "$ready" != 1 ]]; then
  docker logs "$cid" >&2
  exit 1
fi
docker exec -e PGPASSWORD=sforum "$cid" psql -U sforum -d postgres -v ON_ERROR_STOP=1 \
  -c "COMMENT ON DATABASE postgres IS 'sforum.disposable-test-cluster.v1';" >/dev/null
binding="$(docker port "$cid" 5432/tcp)"
if [[ "$binding" != 127.0.0.1:* || "$binding" == *$'\n'* ]]; then
  echo 'Unexpected isolated PostgreSQL port binding.' >&2
  exit 1
fi
port="${binding##*:}"
export DATABASE_URL="postgres://sforum:sforum@127.0.0.1:${port}/sforum_test_suite?sslmode=disable"
export SFORUM_TEST_DATABASE_URL="$DATABASE_URL"
export SFORUM_COMPAT_DATABASE_URL="$DATABASE_URL"
printf 'Disposable test cluster: %s, database: sforum_test_suite, port: %s\n' "$container" "$port"
cd "$ROOT_DIR/apps/api"
go run ./cmd/migrate
if [[ $# == 0 ]]; then
  set -- ./...
fi
if [[ "${1:-}" == "--compat" ]]; then
  shift
  cd "$ROOT_DIR/tests/compat"
  go run . "$@"
else
  # 默认仍然串行：包之间共享同一个 database，靠 schema 隔离。要压墙钟时可
  # 用 SFORUM_TEST_PARALLEL=4 试跑；想强制真跑一遍加 -count=1 即可。
  go test -p "${SFORUM_TEST_PARALLEL:-1}" "$@"
fi
