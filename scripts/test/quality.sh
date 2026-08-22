#!/bin/sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$repo_root"

unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "Go files need formatting:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go vet ./...

if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run --enable=unused
else
  echo "golangci-lint not installed; skipping optional local lint" >&2
fi

git diff --check
