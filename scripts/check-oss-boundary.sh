#!/usr/bin/env bash
# Fails if SaaS / control-plane concerns leak into the open-source data plane.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${ROOT}/internal"

PATTERN='billing|domain/account|domain/user|stripe|oauth'

if matches="$(grep -RInE --include='*.go' -e "$PATTERN" "$TARGET" || true)"; then
  if [[ -n "${matches}" ]]; then
    echo "OSS boundary violation: forbidden patterns found under internal/:"
    echo "${matches}"
    exit 1
  fi
fi

echo "OSS boundary check passed."
