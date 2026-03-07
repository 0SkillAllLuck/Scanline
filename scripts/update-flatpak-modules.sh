#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

go run github.com/dennwc/flatpak-go-mod@latest

sed -i 's|path: modules.txt|path: assets/meta/modules.txt|' go.mod.yml

mv go.mod.yml assets/meta/go.mod.yml
mv modules.txt assets/meta/modules.txt
