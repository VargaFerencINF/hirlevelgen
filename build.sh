#!/usr/bin/env bash
# Windows exe fordítása (Linuxról vagy Windowsról, Git Bash-ben is).
# Használat: ./build.sh [verzió]
set -euo pipefail
cd "$(dirname "$0")"
VERSION="${1:-1.3.0}"
mkdir -p dist
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -H windowsgui -X main.version=${VERSION}" \
  -o dist/EnergofishHirlevel.exe .
cp demo/Energofish_partner_hirlevel_minta.xlsx dist/
ls -la dist
