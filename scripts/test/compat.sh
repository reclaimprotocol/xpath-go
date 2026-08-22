#!/bin/sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
tests_dir="$repo_root/tests"
report="$tests_dir/comprehensive_compatibility_report.json"
backup=$(mktemp "${TMPDIR:-/tmp}/xpath-compat-report.XXXXXX")
report_existed=false

cleanup() {
  if [ "$report_existed" = true ]; then
    cp "$backup" "$report"
  else
    rm -f "$report"
  fi
  rm -f "$backup"
}
trap cleanup EXIT HUP INT TERM

if [ -f "$report" ]; then
  cp "$report" "$backup"
  report_existed=true
fi

cd "$tests_dir"
if [ ! -d node_modules ]; then
  npm ci
fi
npm run test:all
