#!/usr/bin/env bash

set -euo pipefail

day="$(date '+%Y-%m-%d')"
echo "preparing assets for release $day"

targets=(
  darwin-amd64
  darwin-arm64
  linux-amd64
  windows-amd64
)

# Create zipped binaries.
# chdir into dist so zip files sit at root of zip instead of in 'dist/pggen'.
pushd dist >/dev/null
for target in "${targets[@]}"; do
  binary="pggen-${target}"
  if [[ "$binary" == *windows* ]]; then
    binary+='.exe'
  fi
  echo -n "zipping ${binary} ... "
  zip --quiet -9 "pggen-${target}.zip" "${binary}"
  echo "done"
done
popd >/dev/null

# Delete any existing releases. We only support 1 release per day.
# Ignore errors if we try to delete a release that doesn't exist.
github-release delete --user jschaf --repo pggen --tag "$day" 2>/dev/null || true

echo
echo "creating release $day"
github-release release --user jschaf --repo pggen --tag "$day" --name "$day"

# Upload each of the zipped binaries.
for target in "${targets[@]}"; do
  echo -n "uploading pggen-${target}.zip ... "
  github-release upload \
    --user jschaf \
    --repo pggen \
    --tag "$day" \
    --name "pggen-${target}.zip" \
    --file "dist/pggen-${target}.zip"
  echo "done"
done
