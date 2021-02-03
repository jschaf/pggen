#!/usr/bin/env bash

# integration-test runs pggen end-to-end and verifies that there are no diffs.
# If there's a diff it means either the generated files are out of date or
# the codegen is broken.
#
# Pass the --update flag to update all integration tests.

set -euo pipefail

export DOCKER_API_VERSION=1.39

has_update='n'
for arg in "$@"; do
  if [[ $arg == '--update' ]]; then
    has_update='y'
  fi
done

pggen="$(mktemp -t pggen.XXXX)"
go build -o "${pggen}" ./cmd/pggen

function test_header() {
  printf "\n# Test: %s\n" "$*"
}

function assert_no_diff() {
  if [[ "$has_update" == 'y' ]]; then
    return
  fi
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

if [[ "$has_update" == 'y' ]]; then
  echo 'Updating integration tests'
else
  echo 'Running integration tests'
fi

test_header 'example/author: direct file'
${pggen} gen go \
    --schema-glob 'example/author/schema.sql' \
    --query-glob 'example/author/query.sql'
assert_no_diff

# test_header 'example/enums: direct file for query'
# ${pggen} gen go \
#     --schema-glob 'example/enums/schema.sql' \
#     --query-glob 'example/enums/query.sql' \
# assert_no_diff

test_header 'internal/pg'
${pggen} gen go \
   --schema-glob 'example/author/schema.sql' \
   --query-glob 'internal/pg/query.sql' \
   --acronym oid \
   --acronym oids=OIDs
assert_no_diff

test_header 'example/device'
${pggen} gen go \
   --schema-glob 'example/device/schema.sql' \
   --query-glob 'example/device/query.sql'
assert_no_diff

test_header 'example/erp: *.sql glob for schema and query'
${pggen} gen go \
    --schema-glob 'example/erp/*.sql' \
    --query-glob 'example/erp/order/*.sql' \
    --acronym mrr
assert_no_diff

test_header 'example/erp: ?? for schema'
${pggen} gen go \
    --schema-glob 'example/erp/??_schema.sql' \
    --query-glob 'example/erp/order/*.sql' \
    --acronym mrr
assert_no_diff

test_header 'example/syntax: direct file for query'
${pggen} gen go \
    --schema-glob 'example/syntax/schema.sql' \
    --query-glob 'example/syntax/query.sql' \
assert_no_diff

test_header 'example/custom_types'
${pggen} gen go \
     --schema-glob example/custom_types/schema.sql \
     --query-glob example/custom_types/query.sql \
     --go-type 'text=github.com/jschaf/pggen/example/custom_types/mytype.String' \
     --go-type 'int8=github.com/jschaf/pggen/example/custom_types.CustomInt'
assert_no_diff

printf '\n\n'
if [[ "$has_update" == 'y' ]]; then
  echo 'Updated all integration tests'
else
  echo 'All integration tests passed!'
fi
