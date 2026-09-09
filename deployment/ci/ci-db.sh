#!/usr/bin/env bash

set -euo pipefail

readonly env_file="${CI_ENV_FILE:-backend/.env.ci}"
readonly expected_host="127.0.0.1"
readonly expected_port="55432"
readonly expected_name="expense_tracker_ci"
readonly ci_container="expense-tracker-ci-postgres-1"

fail() {
  printf 'ci database check: %s\n' "$*" >&2
  exit 1
}

[[ -f "$env_file" ]] || fail "missing $env_file"
[[ ! -L "$env_file" ]] || fail "$env_file must not be a symbolic link"
[[ "$(stat -c '%a' "$env_file")" == "600" ]] || fail "$env_file must have permissions 600"

set -a
# shellcheck disable=SC1090
source "$env_file"
set +a

: "${CI_DB_HOST:?CI_DB_HOST is required in $env_file}"
: "${CI_DB_PORT:?CI_DB_PORT is required in $env_file}"
: "${CI_DB_NAME:?CI_DB_NAME is required in $env_file}"
: "${CI_DB_USER:?CI_DB_USER is required in $env_file}"
: "${CI_DB_PASSWORD:?CI_DB_PASSWORD is required in $env_file}"

[[ "$CI_DB_HOST" == "$expected_host" ]] || fail "CI_DB_HOST must be $expected_host"
[[ "$CI_DB_PORT" == "$expected_port" ]] || fail "CI_DB_PORT must be $expected_port"
[[ "$CI_DB_NAME" == "$expected_name" ]] || fail "CI_DB_NAME must be $expected_name"

export DB_PUBLIC_HOST="$CI_DB_HOST"
export DB_PORT="$CI_DB_PORT"
export DB_NAME="$CI_DB_NAME"
export DB_USER="$CI_DB_USER"
export DB_PASSWORD="$CI_DB_PASSWORD"
export DB_SSLMODE="disable"
export PGPASSWORD="$CI_DB_PASSWORD"

psql_ci() {
  psql \
    --host "$CI_DB_HOST" \
    --port "$CI_DB_PORT" \
    --username "$CI_DB_USER" \
    --dbname "$CI_DB_NAME" \
    --no-password \
    --set ON_ERROR_STOP=1 \
    "$@"
}

verify_database() {
  command -v psql >/dev/null || fail "psql is required"
  local identity
  identity="$(psql_ci --tuples-only --no-align --command 'SELECT current_database()')" ||
    fail "cannot connect to the dedicated CI database"
  [[ "$identity" == "$expected_name" ]] || fail "connected to unexpected database $identity"
}

stop_container() {
  local status=$?
  trap - EXIT
  if ! docker stop "$ci_container" >/dev/null; then
    printf 'ci database check: failed to stop container %s\n' "$ci_container" >&2
    if ((status == 0)); then
      status=1
    fi
  fi
  exit "$status"
}

run_checks() {
  shift
  (($# > 0)) || fail "run requires a command"
  command -v docker >/dev/null || fail "docker is required"
  command -v psql >/dev/null || fail "psql is required"

  docker start "$ci_container" >/dev/null ||
    fail "cannot start the dedicated CI container $ci_container"
  trap stop_container EXIT

  for _ in {1..30}; do
    if psql_ci --tuples-only --no-align --command 'SELECT 1' >/dev/null 2>&1; then
      verify_database
      "$@"
      return
    fi
    sleep 1
  done
  fail "the dedicated CI database did not become ready"
}

case "${1:-}" in
  run)
    run_checks "$@"
    ;;
  prepare)
    verify_database
    psql_ci --command 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;' >/dev/null
    go run ./backend/cmd/migrate up
    ;;
  test)
    verify_database
    shift
    exec go test -count=1 "$@"
    ;;
  verify)
    verify_database
    ;;
  *)
    fail "expected run, prepare, test, or verify"
    ;;
esac
