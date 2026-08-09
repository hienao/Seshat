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

case "$release_kind" in
  beta)
    version_file="VERSION_BETA"
    tag_prefix="beta-"
    ;;
  release|hotfix)
    version_file="VERSION_RELEASE"
    tag_prefix=""
    ;;
  *)
    usage
    exit 2
    ;;
esac

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/../../../.." && pwd)"
version_path="$repo_root/$version_file"

if [[ ! -f "$version_path" ]]; then
  echo "ERROR: missing $version_file" >&2
  exit 1
fi

line_count="$(awk 'END { print NR }' "$version_path")"
version="$(tr -d '\r\n' < "$version_path")"
if [[ "$line_count" != "1" || ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "ERROR: $version_file must contain exactly one vX.Y.Z line" >&2
  exit 1
fi

if [[ -n "$expected_version" && "$version" != "$expected_version" ]]; then
  echo "ERROR: $version_file is $version, expected $expected_version" >&2
  exit 1
fi

echo "OK: $release_kind uses $version_file=$version and image tag ${tag_prefix}${version}"
