#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="${1:-.}"
REPORT_DIR="${REPORT_DIR:-secret-scan-reports}"

if ! command -v gitleaks >/dev/null 2>&1; then
    echo "Error: gitleaks is not installed."
    echo "Install using:"
    echo "  go install github.com/gitleaks/gitleaks/v8@latest"
    exit 1
fi

mkdir -p "$REPORT_DIR"

echo "Scanning current files..."

gitleaks dir "$ROOT_DIR" \
  --redact \
  --report-format json \
  --report-path "$REPORT_DIR/working-tree.json"

if git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Scanning Git history..."

    gitleaks git "$ROOT_DIR" \
      --redact \
      --report-format json \
      --report-path "$REPORT_DIR/git-history.json"
else
    echo "Skipping Git history: not a Git repository."
fi

echo "Secret scan completed."
echo "Reports: $REPORT_DIR"