#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime="$repo_root/runtime/smoke"
database="$repo_root/testdata/smoke/app.db"
binary="$repo_root/bin/lazymarking-server"

mkdir -p "$runtime/db/data" "$runtime/assets/images" "$repo_root/bin"

ln -sfn "$database" "$runtime/db/data/app.db"
ln -sfn "$repo_root/internal" "$runtime/internal"

echo "== build =="
cd "$repo_root"
go build -o "$binary" ./cmd/server

echo "== smoke environment =="
cd "$runtime"

SESSION_SECURE=false \
SESSION_KEY="$(openssl rand -hex 16)" \
CSRF_AUTH_KEY="$(openssl rand -hex 16)" \
"$binary"
