#!/bin/sh

set -eu

BASE_URL=${BASE_URL:-http://localhost:8090}
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
ENV_FILE=${ENV_FILE:-$REPO_ROOT/.env}

if [ ! -f "$ENV_FILE" ]; then
	echo ".env file not found: $ENV_FILE" >&2
	exit 1
fi

get_env_value() {
	name=$1
	line=$(grep -E "^${name}=" "$ENV_FILE" | tail -n 1 || true)
	printf '%s' "${line#*=}"
}

json_escape() {
	printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

extract_json_string() {
	key=$1
	printf '%s' "$2" | tr -d '\n' | sed -n "s/.*\"${key}\":\"\([^\"]*\)\".*/\1/p"
}

extract_json_number() {
	key=$1
	printf '%s' "$2" | tr -d '\n' | sed -n "s/.*\"${key}\":\([0-9][0-9]*\).*/\1/p"
}

ADMIN_EMAIL=$(get_env_value PB_ADMIN_EMAIL)
ADMIN_PASSWORD=$(get_env_value PB_ADMIN_PASSWORD)

if [ -z "$ADMIN_EMAIL" ] || [ -z "$ADMIN_PASSWORD" ]; then
	echo "PB_ADMIN_EMAIL and PB_ADMIN_PASSWORD must both be set in $ENV_FILE." >&2
	exit 1
fi

base=${BASE_URL%/}
auth_payload=$(printf '{"identity":"%s","password":"%s"}' \
	"$(json_escape "$ADMIN_EMAIL")" \
	"$(json_escape "$ADMIN_PASSWORD")")

auth_json=$(curl -fsS \
	-H 'Content-Type: application/json' \
	-d "$auth_payload" \
	"$base/api/collections/_superusers/auth-with-password")

token=$(extract_json_string token "$auth_json")
if [ -z "$token" ]; then
	echo 'Superuser login succeeded without returning a token.' >&2
	exit 1
fi

response=$(curl -fsS \
	-H "Authorization: Bearer $token" \
	-H 'Content-Type: application/json' \
	-d '{}' \
	"$base/api/notifications/send-incomplete")

incomplete=$(extract_json_number incomplete "$response")
sent=$(extract_json_number sent "$response")
already_sent=$(extract_json_number alreadySent "$response")
failed=$(extract_json_number failed "$response")

printf 'Unfinished reminder run completed. Incomplete: %s, sent: %s, already sent: %s, failed: %s.\n' \
	"${incomplete:-?}" "${sent:-?}" "${already_sent:-?}" "${failed:-?}"