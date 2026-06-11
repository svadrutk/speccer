#!/bin/bash
set -e

SPEC_DIR="$(dirname "$0")/specs"
WORKSPACE="${1:-/Users/svadrut/Documents/songranker-app/songranker-backend}"
SOURCE="${2:-app/**/*.py}"
TEST_BIN="$(dirname "$0")/runner.go"

echo "=========================================="
echo " Drift Detection Test Suite"
echo " Workspace: $WORKSPACE"
echo " Source:    $SOURCE"
echo "=========================================="
echo ""

cd "$(dirname "$0")/.."

for spec in "$SPEC_DIR"/*.md; do
  name=$(basename "$spec" .md)
  echo "------------------------------------------"
  echo " TEST: $name"
  echo " Spec: $(basename "$spec")"
  echo "------------------------------------------"

  output=$(go run "$TEST_BIN" "$spec" "$WORKSPACE" "$SOURCE" 2>&1)
  exit_code=$?

  if echo "$output" | grep -q '\[P'; then
    echo "$output" | grep --color=always '\[P'
  else
    echo " ✓ No drift detected"
  fi
  echo ""
done
