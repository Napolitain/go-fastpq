#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir/.."

: "${GOLANGCI_LINT_VERSION:=v2.11.4}"
: "${DEADCODE_VERSION:=v0.45.0}"

echo "==> gofmt"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    printf '%s\n' "$unformatted"
    echo "Go files must be formatted with gofmt."
    exit 1
fi

echo "==> go mod tidy"
go mod tidy
if [ -n "$(git status --porcelain -- go.mod go.sum)" ]; then
    git status --short -- go.mod go.sum
    echo "go.mod/go.sum changed after go mod tidy."
    exit 1
fi

echo "==> golangci-lint"
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run ./...
else
    go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}" run ./...
fi

echo "==> go test"
go test -v -coverprofile=coverage.out -covermode=atomic ./...

echo "==> go build"
go build ./...

echo "==> deadcode"
go run "golang.org/x/tools/cmd/deadcode@${DEADCODE_VERSION}" -test ./...
