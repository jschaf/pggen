#!/usr/bin/env bash

# integration-test runs pggen end-to-end and verifies that there are no diffs.
# If there's a diff it means either the generated files are out of date or
# the codegen is broken.

set -euo pipefail

export DOCKER_API_VERSION=1.39

pggen="$(mktemp -t pggen.XXXX)"
go build -o "${pggen}" ./cmd/pggen

function test_header() {
  printf "\n# Test: %s\n" "$*"
}

function assert_no_diff() {
  if ! git update-index --refresh > /dev/null; then
    echo 'FAIL: integration test has diff'
    git diff
    exit 1
  fi
  if ! git diff-index --quiet HEAD --; then
    echo 'FAIL: integration test has diff'
    git diff
    exit 1
  fi
}

echo 'Running integration tests'

test_header 'example/author: direct file'
${pggen} gen go \
    --schema-glob 'example/author/schema.sql' \
    --query-glob 'example/author/query.sql'
assert_no_diff

test_header 'example/erp: *.sql glob for schema and query'
${pggen} gen go \
    --schema-glob 'example/erp/*.sql' \
    --query-glob 'example/erp/order/*.sql'
assert_no_diff

test_header 'example/erp: ?? for schema'
${pggen} gen go \
    --schema-glob 'example/erp/??_schema.sql' \
    --query-glob 'example/erp/order/*.sql'
assert_no_diff

test_header 'example/syntax: *.sql for query'
${pggen} gen go \
    --schema-glob 'example/syntax/schema.sql' \
    --query-glob 'example/syntax/*.sql'
assert_no_diff

test_header 'example/syntax: direct file for query'
${pggen} gen go \
    --schema-glob 'example/syntax/schema.sql' \
    --query-glob 'example/syntax/query.sql'
assert_no_diff

printf '\n\n'
echo 'All integration tests passed!'
