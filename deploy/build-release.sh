#!/usr/bin/env bash
# Build and test an isolated source archive as the unprivileged build account.
set -euo pipefail
cd -- "$1"
export ENV_FILE=none
export GOTOOLCHAIN=go1.26.5
export GOEXPERIMENT=nojsonv2
export CGO_ENABLED=1
export GOMAXPROCS=1
export GOMEMLIMIT=450MiB
go test -p 1 ./...
mkdir -p bin
go build -p 1 -trimpath -buildvcs=false -o bin/mishra4you ./cmd/server
npm --prefix web ci --include=dev
NODE_OPTIONS=--max-old-space-size=512 npm --prefix web run build
test -s bin/mishra4you
test -s web/dist/index.html
