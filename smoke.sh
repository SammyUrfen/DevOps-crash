#!/usr/bin/env bash
# Smoke test for the task API. Start the app first, then run this.
# Usage: ./smoke.sh [base-url]
set -euo pipefail
B="${1:-http://localhost:8080}"

pass=0
check() { # check <name> <expected> <actual>
  if [ "$2" = "$3" ]; then
    echo "ok   $1"
    pass=$((pass + 1))
  else
    echo "FAIL $1: want $2, got $3"
    exit 1
  fi
}
code() { curl -sS -o /dev/null -w '%{http_code}' "$@"; }
errcode() { curl -sS "$@" | sed -n 's/.*"code":"\([^"]*\)".*/\1/p'; }

check live 200 "$(code "$B/healthz")"
check ready 200 "$(code "$B/readyz")"
check index 200 "$(code "$B/")"

id=$(curl -sS -X POST "$B/api/tasks" -d '{"title":"smoke"}' | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
check create-returns-id 1 "$([ -n "$id" ] && echo 1)"
check list-has-task 1 "$(curl -sS "$B/api/tasks" | grep -c "\"id\":$id,")"
check update 200 "$(code -X PUT "$B/api/tasks/$id" -d '{"title":"smoke","done":true}')"
check delete 204 "$(code -X DELETE "$B/api/tasks/$id")"
check delete-twice 404 "$(code -X DELETE "$B/api/tasks/$id")"

check empty-title-status 400 "$(code -X POST "$B/api/tasks" -d '{"title":"  "}')"
check empty-title-code title_required "$(errcode -X POST "$B/api/tasks" -d '{"title":"  "}')"
check bad-json-code invalid_body "$(errcode -X POST "$B/api/tasks" -d 'nope')"
check unknown-field-code invalid_body "$(errcode -X POST "$B/api/tasks" -d '{"title":"a","id":9}')"
check bad-id-code invalid_id "$(errcode -X DELETE "$B/api/tasks/abc")"
check missing-task-code task_not_found "$(errcode -X DELETE "$B/api/tasks/999999")"

echo "$pass checks passed"
