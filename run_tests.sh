#!/usr/bin/env bash
set -uo pipefail

# Integration test runner for the organizational structure API.
# Requirements: docker, docker compose, curl, python3.
#
# Run from the repository root:
#   ./run_tests.sh
#
# Optional environment variables from .env are loaded automatically.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

COMPOSE_CMD=(docker compose)
APP_PORT="${APP_PORT:-8088}"
DB_PORT="${DB_PORT:-5433}"
POSTGRES_DB="${POSTGRES_DB:-organization}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
BASE_URL="http://localhost:${APP_PORT}"

if [[ -f .env ]]; then
  # shellcheck disable=SC1091
  set -a
  # shellcheck disable=SC1090
  source ./.env
  set +a
  APP_PORT="${APP_PORT:-8088}"
  DB_PORT="${DB_PORT:-5433}"
  POSTGRES_DB="${POSTGRES_DB:-organization}"
  POSTGRES_USER="${POSTGRES_USER:-postgres}"
  POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
  BASE_URL="http://localhost:${APP_PORT}"
fi

FAILS=0
TMPDIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMPDIR"
}
trap cleanup EXIT

pass() { printf '[OK] %s\n' "$1"; }
fail() { printf '[FAIL] %s\n' "$1" >&2; FAILS=$((FAILS + 1)); }

assert_eq() {
  local expected="$1"
  local actual="$2"
  local msg="$3"
  if [[ "$expected" == "$actual" ]]; then
    pass "$msg"
  else
    fail "$msg (expected: $expected, got: $actual)"
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local msg="$3"
  if grep -Fq "$needle" <<<"$haystack"; then
    pass "$msg"
  else
    fail "$msg (missing: $needle)"
  fi
}

json_get() {
  local file="$1"
  local path="$2"
  python3 - "$file" "$path" <<'PY'
import json, sys

file_path = sys.argv[1]
path = sys.argv[2].split('.')

with open(file_path, 'r', encoding='utf-8') as f:
    data = json.load(f)

cur = data
for part in path:
    if isinstance(cur, list):
        cur = cur[int(part)]
    else:
        cur = cur[part]
print(cur)
PY
}

request() {
  local method="$1"
  local url="$2"
  local body="${3:-}"
  local out="$4"
  local http_code

  if [[ -n "$body" ]]; then
    http_code="$(
      curl -sS -o "$out" -w '%{http_code}' \
        -X "$method" "$url" \
        -H 'Content-Type: application/json' \
        --data "$body" || true
    )"
  else
    http_code="$(
      curl -sS -o "$out" -w '%{http_code}' \
        -X "$method" "$url" || true
    )"
  fi
  printf '%s' "$http_code"
}

wait_for_api() {
  printf 'Waiting for API at %s ...\n' "$BASE_URL"
  for _ in {1..60}; do
    code="$(curl -sS -o /dev/null -w '%{http_code}' "$BASE_URL/departments/1" 2>/dev/null || true)"
    if [[ "$code" == "404" || "$code" == "200" || "$code" == "400" || "$code" == "500" ]]; then
      return 0
    fi
    sleep 1
  done
  return 1
}

echo "==> Clean start"
"${COMPOSE_CMD[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

echo "==> Build and start"
"${COMPOSE_CMD[@]}" up -d --build

if ! wait_for_api; then
  fail "API did not become ready"
  "${COMPOSE_CMD[@]}" logs --no-color app || true
  exit 1
fi
pass "API is reachable"


create_department() {
  local name="$1"
  local parent_id="${2:-}"
  local body
  if [[ -n "$parent_id" ]]; then
    body="$(printf '{"name":"%s","parent_id":%s}' "$name" "$parent_id")"
  else
    body="$(printf '{"name":"%s"}' "$name")"
  fi
  local out="$TMPDIR/department.json"
  code="$(request POST "$BASE_URL/departments/" "$body" "$out")"
  echo "$code" > "$TMPDIR/last_code"
  cat "$out"
}

create_employee() {
  local dept_id="$1"
  local full_name="$2"
  local position="$3"
  local hired_at="${4:-}"
  local body out code
  if [[ -n "$hired_at" ]]; then
    body="$(printf '{"full_name":"%s","position":"%s","hired_at":"%s"}' "$full_name" "$position" "$hired_at")"
  else
    body="$(printf '{"full_name":"%s","position":"%s"}' "$full_name" "$position")"
  fi
  out="$TMPDIR/employee.json"
  code="$(request POST "$BASE_URL/departments/${dept_id}/employees/" "$body" "$out")"
  echo "$code" > "$TMPDIR/last_code"
  cat "$out"
}

patch_department() {
  local dept_id="$1"
  local body="$2"
  local out="$TMPDIR/patch.json"
  code="$(request PATCH "$BASE_URL/departments/${dept_id}" "$body" "$out")"
  echo "$code" > "$TMPDIR/last_code"
  cat "$out"
}

get_department() {
  local dept_id="$1"
  local qs="${2:-}"
  local out="$TMPDIR/get.json"
  local url="$BASE_URL/departments/${dept_id}${qs}"
  code="$(request GET "$url" "" "$out")"
  echo "$code" > "$TMPDIR/last_code"
  cat "$out"
}

delete_department() {
  local dept_id="$1"
  local qs="${2:-}"
  local out="$TMPDIR/delete.json"
  local url="$BASE_URL/departments/${dept_id}${qs}"
  code="$(request DELETE "$url" "" "$out")"
  echo "$code" > "$TMPDIR/last_code"
  cat "$out"
}

echo "==> Base CRUD and tree checks"
dep_root_json="$(create_department "Backend")"
dep_root_code="$(cat "$TMPDIR/last_code")"
assert_eq "201" "$dep_root_code" "Create root department returns 201"
dep_root_id="$(json_get <(printf '%s' "$dep_root_json") "id")"

dep_child_json="$(create_department "Platform" "$dep_root_id")"
dep_child_code="$(cat "$TMPDIR/last_code")"
assert_eq "201" "$dep_child_code" "Create child department returns 201"
dep_child_id="$(json_get <(printf '%s' "$dep_child_json") "id")"

dep_grand_json="$(create_department "Infra" "$dep_child_id")"
dep_grand_code="$(cat "$TMPDIR/last_code")"
assert_eq "201" "$dep_grand_code" "Create grandchild department returns 201"
dep_grand_id="$(json_get <(printf '%s' "$dep_grand_json") "id")"

emp_json="$(create_employee "$dep_root_id" "Ivan Ivanov" "Go Developer")"
emp_code="$(cat "$TMPDIR/last_code")"
assert_eq "201" "$emp_code" "Create employee returns 201"
emp_id="$(json_get <(printf '%s' "$emp_json") "id")"

tree_depth_1="$(get_department "$dep_root_id" '?depth=1')"
tree_depth_1_code="$(cat "$TMPDIR/last_code")"
assert_eq "200" "$tree_depth_1_code" "GET department depth=1 returns 200"
assert_contains "$tree_depth_1" "\"employees\"" "GET department includes employees by default"
assert_contains "$tree_depth_1" "\"Platform\"" "GET department depth=1 contains direct child"
if grep -Fq '"Infra"' <<<"$tree_depth_1"; then
  fail "GET department depth=1 must not include grandchild"
else
  pass "GET department depth=1 excludes grandchild"
fi

tree_depth_2="$(get_department "$dep_root_id" '?depth=2')"
tree_depth_2_code="$(cat "$TMPDIR/last_code")"
assert_eq "200" "$tree_depth_2_code" "GET department depth=2 returns 200"
assert_contains "$tree_depth_2" "\"Infra\"" "GET department depth=2 contains grandchild"

tree_no_emp="$(get_department "$dep_root_id" '?include_employees=false')"
tree_no_emp_code="$(cat "$TMPDIR/last_code")"
assert_eq "200" "$tree_no_emp_code" "GET department include_employees=false returns 200"
if grep -Fq '"employees"' <<<"$tree_no_emp"; then
  fail "include_employees=false must omit employees"
else
  pass "include_employees=false omits employees"
fi

echo "==> Validation and business rules"
dup_json="$(create_department "Platform" "$dep_root_id")"
dup_code="$(cat "$TMPDIR/last_code")"
assert_eq "409" "$dup_code" "Duplicate department under same parent returns 409"
assert_contains "$dup_json" "unique within parent" "Duplicate department error is descriptive"

empty_name_out="$TMPDIR/empty_name.json"
empty_code="$(request POST "$BASE_URL/departments/" '{"name":""}' "$empty_name_out")"
assert_eq "400" "$empty_code" "Empty department name must return 400"
if [[ "$empty_code" != "400" ]]; then
  fail "Empty department name currently fails TЗ validation"
fi

long_name="$(python3 - <<'PY'
print("A" * 201)
PY
)"
long_name_out="$TMPDIR/long_name.json"
long_code="$(request POST "$BASE_URL/departments/" "$(printf '{"name":"%s"}' "$long_name")" "$long_name_out")"
assert_eq "400" "$long_code" "Too long department name must return 400"

bad_emp_dept_out="$TMPDIR/bad_emp_dept.json"
bad_emp_dept_code="$(request POST "$BASE_URL/departments/999/employees/" '{"full_name":"Test","position":"QA"}' "$bad_emp_dept_out")"
assert_eq "404" "$bad_emp_dept_code" "Employee in missing department returns 404"

empty_emp_name_out="$TMPDIR/empty_emp_name.json"
empty_emp_name_code="$(request POST "$BASE_URL/departments/${dep_root_id}/employees/" '{"full_name":"","position":"QA"}' "$empty_emp_name_out")"
assert_eq "400" "$empty_emp_name_code" "Empty employee full_name returns 400"

empty_pos_out="$TMPDIR/empty_pos.json"
empty_pos_code="$(request POST "$BASE_URL/departments/${dep_root_id}/employees/" '{"full_name":"Test User","position":""}' "$empty_pos_out")"
assert_eq "400" "$empty_pos_code" "Empty employee position returns 400"

self_parent_json="$(patch_department "$dep_root_id" '{"parent_id":'"$dep_root_id"'}')"
self_parent_code="$(cat "$TMPDIR/last_code")"
assert_eq "409" "$self_parent_code" "Department cannot be parent of itself"
assert_contains "$self_parent_json" "parent of itself" "Self-parent error message is descriptive"

cycle_json="$(patch_department "$dep_root_id" '{"parent_id":'"$dep_child_id"'}')"
cycle_code="$(cat "$TMPDIR/last_code")"
assert_eq "409" "$cycle_code" "Moving department into its subtree returns 409"
assert_contains "$cycle_json" "subtree" "Cycle error message is descriptive"

echo "==> Cascade delete"
cascade_json="$(delete_department "$dep_root_id" '?mode=cascade')"
cascade_code="$(cat "$TMPDIR/last_code")"
assert_eq "204" "$cascade_code" "Cascade delete returns 204"

dept_count="$(docker compose exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT count(*) FROM departments;" | tr -d '[:space:]')"
emp_count="$(docker compose exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT count(*) FROM employees;" | tr -d '[:space:]')"
assert_eq "0" "$dept_count" "Cascade delete removes all departments"
assert_eq "0" "$emp_count" "Cascade delete removes all employees"

echo "==> Reassign delete"
hr_json="$(create_department "HR")"
hr_code="$(cat "$TMPDIR/last_code")"
assert_eq "201" "$hr_code" "Create HR returns 201"
hr_id="$(json_get <(printf '%s' "$hr_json") "id")"

fin_json="$(create_department "Finance")"
fin_code="$(cat "$TMPDIR/last_code")"
assert_eq "201" "$fin_code" "Create Finance returns 201"
fin_id="$(json_get <(printf '%s' "$fin_json") "id")"

re_emp_json="$(create_employee "$hr_id" "Alice" "HR")"
re_emp_code="$(cat "$TMPDIR/last_code")"
assert_eq "201" "$re_emp_code" "Create employee for reassign test returns 201"

reassign_json="$(delete_department "$hr_id" '?mode=reassign&reassign_to_department_id='"$fin_id")"
reassign_code="$(cat "$TMPDIR/last_code")"
assert_eq "204" "$reassign_code" "Reassign delete returns 204"

reassigned_dept_id="$(docker compose exec -T db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT department_id FROM employees WHERE full_name='Alice' LIMIT 1;" | tr -d '[:space:]')"
assert_eq "$fin_id" "$reassigned_dept_id" "Employee is reassigned to the target department"

echo "==> Final API smoke checks"
final_hr="$(get_department "$fin_id")"
final_code="$(cat "$TMPDIR/last_code")"
assert_eq "200" "$final_code" "Target department is still retrievable"

if [[ "$FAILS" -eq 0 ]]; then
  echo
  echo "ALL TESTS PASSED"
  exit 0
else
  echo
  echo "TESTS FAILED: $FAILS issue(s) found"
  exit 1
fi

