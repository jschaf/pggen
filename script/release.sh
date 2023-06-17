#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo 'error: no GITHUB_TOKEN env var'
  exit 1
fi


# Download github-release if necessary. Only used to delete existing releases.
# We use GoReleaser to create new releases.
githubRelease='github-release'
if ! command -v "$githubRelease"; then
  echo 'downloading github-release'
  githubRelease="$(mktemp)"
  url=https://github.com/github-release/github-release/releases/download/v0.10.0/linux-amd64-github-release.bz2
  curl -L --fail --silent "${url}" | bzip2 -dc >"$githubRelease"
  chmod +x "$githubRelease"
else
  echo 'github-release already downloaded'
fi

goReleaser='goreleaser'
if ! command -v "$goReleaser"; then
  goReleaserUrl='https://github.com/goreleaser/goreleaser/releases/download/v1.10.2/goreleaser_Linux_x86_64.tar.gz'
  curl -L --fail --silent "${goReleaserUrl}" | tar xvz >"$goReleaser"
  chmod +x "$goReleaser"
else
  echo 'goreleaser already downloaded'
fi

day="$(date '+%Y-%m-%d')"

# Delete the remote tag since we're creating a new release tagged today.
echo 'deleting existing release tag'
git push origin ":refs/tags/$day" 2>/dev/null
# Create or move the day tag to the latest commit.
git tag -f "$day"
git push origin "$day"

# Delete any existing releases. We only support 1 release per day.
# Ignore errors if we try to delete a release that doesn't exist.
echo 'deleting existing releases'
"$githubRelease" delete --user jschaf --repo pggen --tag "$day" || true

echo
echo "creating release $day"
goreleaser release --config ./script/.goreleaser.yaml --clean
