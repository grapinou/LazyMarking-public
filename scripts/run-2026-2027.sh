#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime="$repo_root/runtime/2026-2027"
database="$repo_root/testdata/2026-2027/app.db"
binary="$repo_root/bin/lazymarking-server"
env_file="$repo_root/.env"
if [[ ! -f "$database" ]]; then
  echo "Missing 2026-2027 database: $database" >&2
  exit 1
fi
if [[ ! -f "$env_file" ]]; then
  echo "Missing environment file: $env_file" >&2
  exit 1
fi

session_key="$(sed -n 's/^SESSION_KEY=//p' "$env_file")"
session_key="${session_key%\"}"
session_key="${session_key#\"}"
session_key="${session_key%\'}"
session_key="${session_key#\'}"


mkdir -p "$runtime/db/data" "$runtime/assets/images" "$runtime/assets/tmp" "$repo_root/bin"

ln -sfn "$env_file" "$runtime/.env"

ln -sfn "$database" "$runtime/db/data/app.db"
ln -sfn "$repo_root/internal" "$runtime/internal"

echo "== build =="
cd "$repo_root"
go build -o "$binary" ./cmd/server

echo "== 2026-2027 environment =="
cd "$runtime"

SESSION_SECURE=false \
SESSION_KEY="$session_key" \
SESSION_COOKIE_NAME=lazymarking_2026_2027_session \
CSRF_AUTH_KEY="$(openssl rand -hex 16)" \
"$binary"
