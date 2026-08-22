#!/bin/sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cd "$repo_root"

case "${1:-unit}" in
  unit)
    exec go test -count=1 ./...
    ;;
  race)
    exec go test -race -count=1 ./...
    ;;
  scaling)
    exec go test -count=3 \
      -run 'Scaling|Scales|Performance|DirtyText' \
      ./internal/evaluator ./pkg/utils ./tests/integration
    ;;
  fuzz)
    exec go test ./internal/parser -run '^$' \
      -fuzz FuzzParseXPath -fuzztime "${FUZZTIME:-10s}"
    ;;
  bench)
    exec go test -run '^$' -bench . -benchmem ./...
    ;;
  *)
    echo "usage: $0 {unit|race|scaling|fuzz|bench}" >&2
    exit 2
    ;;
esac
