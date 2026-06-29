#!/bin/sh
# Copy patches and contrib into go/embed/ so go:embed can access them.
# Run from the repo root: sh go/copy-assets.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

mkdir -p "$SCRIPT_DIR/embed/patches"
mkdir -p "$SCRIPT_DIR/embed/contrib"

cp "$REPO_ROOT"/patches/*.json "$SCRIPT_DIR/embed/patches/"
cp -r "$REPO_ROOT"/contrib/* "$SCRIPT_DIR/embed/contrib/"

echo "Assets copied to go/embed/"
