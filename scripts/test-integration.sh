#!/bin/sh
set -eu
BASE_URL="${OBSERVA_BASE_URL:-https://127.0.0.1:8443}"
CURL_TLS="${OBSERVA_CURL_TLS_ARGS:--k}"

status() {
  # shellcheck disable=SC2086
  curl --silent --show-error $CURL_TLS -o /dev/null -w '%{http_code}' "$@"
}

wait_for_status() {
  path="$1"
  expected="$2"
  tries=0
  actual="000"
  while [ "$tries" -lt 60 ]; do
    actual="$(status "$BASE_URL$path" 2>/dev/null || true)"
    if [ "$actual" = "$expected" ]; then
      echo "PASS $path through Envoy: $actual"
      return 0
    fi
    tries=$((tries + 1))
    sleep 1
  done
  echo "FAIL $path status=$actual after ${tries}s (want $expected)" >&2
  return 1
}

wait_for_status /health/live 200
wait_for_status /health/ready 200
unauthenticated="$(status -X POST -H 'Content-Type: application/cloudevents+json'   --data '{"specversion":"1.0"}' "$BASE_URL/v1/events")"
[ "$unauthenticated" = "401" ] || { echo "FAIL unauthenticated event request status=$unauthenticated (want 401)"; exit 1; }
echo "PASS event route rejects requests without a JWT: $unauthenticated"
echo "PASS local integration smoke test"
