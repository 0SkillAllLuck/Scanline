#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

hash=$(nix build .#scanline.goModules --no-link 2>&1 | grep -oP 'got:\s+\K\S+' || true)

if [ -z "$hash" ]; then
  echo "No hash mismatch detected — vendorHash is already up to date."
  exit 0
fi

sed -i "s|vendorHash = \".*\"|vendorHash = \"$hash\"|" flake.nix
echo "Updated vendorHash to $hash"
