#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo "== go mod verify =="
go mod verify

echo "== gofmt =="
mapfile -d '' go_files < <(find . -path './.git' -prune -o -type f -name '*.go' -print0)
unformatted="$(gofmt -l "${go_files[@]}")"
if [[ -n "$unformatted" ]]; then
  echo "Go files must be formatted with gofmt:" >&2
  echo "$unformatted" >&2
  exit 1
fi

echo "== go vet =="
go vet ./...

echo "== go test =="
go test ./...

echo "== go build =="
go build ./...

echo "== git diff --check =="
git diff --check
