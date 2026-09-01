#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v goose >/dev/null 2>&1; then
  echo "goose is required to validate migration replay" >&2
  exit 1
fi

replay_dir="$(mktemp -d)"
trap 'rm -rf -- "$replay_dir"' EXIT

replay_db="$replay_dir/replay.db"
goose -dir "$repo_root/db/migrations" sqlite "$replay_db" up

latest_migration="$(find "$repo_root/db/migrations" -maxdepth 1 -type f -name '*.sql' -printf '%f\n' | sort | tail -n 1)"
latest_version="$(printf '%s' "$latest_migration" | cut -d_ -f1 | sed 's/^0*//')"
applied_version="$(goose -dir "$repo_root/db/migrations" sqlite "$replay_db" version 2>&1 | sed -n 's/.*version //p' | tail -n 1)"

if [[ -z "$latest_version" || "$applied_version" != "$latest_version" ]]; then
  echo "migration replay reached version ${applied_version:-unknown}, want ${latest_version:-unknown}" >&2
  exit 1
fi

goose -dir "$repo_root/db/migrations" sqlite "$replay_db" status
