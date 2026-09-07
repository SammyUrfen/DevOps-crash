#!/usr/bin/env bash
# Smoke test for the task API. Start the app first, then run this.
set -euo pipefail
B="${1:-http://localhost:8080}"

check() { # check <name> <expected> <actual>
  if [ "$2" = "$3" ]; then echo "ok   $1"; else echo "FAIL $1: want $2, got $3"; exit 1; fi
}
code() { curl -sS -o /dev/null -w '%{http_code}' "$@"; }

check health ok "$(curl -sS "$B/healthz")"

id=$(curl -sS -X POST "$B/api/tasks" -d '{"title":"smoke"}' | grep -o '"id":[0-9]*' | cut -d: -f2)
check create-returns-id 1 "$([ -n "$id" ] && echo 1)"
check list-has-task 1 "$(curl -sS "$B/api/tasks" | grep -c "\"id\":$id,")"
check update 200 "$(code -X PUT "$B/api/tasks/$id" -d '{"title":"smoke","done":true}')"
check delete 204 "$(code -X DELETE "$B/api/tasks/$id")"
check delete-twice 404 "$(code -X DELETE "$B/api/tasks/$id")"
check empty-title 400 "$(code -X POST "$B/api/tasks" -d '{"title":"  "}')"
check bad-json 400 "$(code -X POST "$B/api/tasks" -d 'nope')"
check bad-id 400 "$(code -X DELETE "$B/api/tasks/abc")"
check index 200 "$(code "$B/")"
echo "all checks passed"
