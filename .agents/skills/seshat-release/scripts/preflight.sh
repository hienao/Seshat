#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <beta|release|hotfix> [expected-vX.Y.Z]" >&2
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 2
fi

release_kind="$1"
expected_version="${2:-}"
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/../../../.." && pwd)"

"$script_dir/validate-version.sh" "$release_kind" ${expected_version:+"$expected_version"}

(
  cd "$repo_root/backend"
  go test ./...
)

(
  cd "$repo_root/frontend"
  npm run test
  npm run build
)

case "$release_kind" in
  beta) target_branch="beta" ;;
  release|hotfix) target_branch="main" ;;
  *) usage; exit 2 ;;
esac

workflow_changed="$(
  git -C "$repo_root" diff --name-only -- .github/workflows/docker-publish.yml
  git -C "$repo_root" diff --cached --name-only -- .github/workflows/docker-publish.yml
  if git -C "$repo_root" rev-parse --verify --quiet "origin/$target_branch" >/dev/null; then
    git -C "$repo_root" diff --name-only "origin/$target_branch...HEAD" -- .github/workflows/docker-publish.yml
  elif git -C "$repo_root" rev-parse --verify --quiet "$target_branch" >/dev/null; then
    git -C "$repo_root" diff --name-only "$target_branch...HEAD" -- .github/workflows/docker-publish.yml
  fi
)"

if [[ -n "$workflow_changed" ]]; then
  (
    cd "$repo_root"
    go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/docker-publish.yml
  )
fi

echo "OK: Seshat $release_kind pre-release checks passed"
