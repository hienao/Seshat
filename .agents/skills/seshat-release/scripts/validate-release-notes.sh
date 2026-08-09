#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <beta|release> <vX.Y.Z>" >&2
  exit 2
fi

channel="$1"
version="$2"
if [[ "$channel" != "beta" && "$channel" != "release" ]]; then
  echo "ERROR: channel must be beta or release" >&2
  exit 2
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/../../../.." && pwd)"

(
  cd "$repo_root/backend"
  go run ./cmd/release-notes validate "$channel" "$version"
)
