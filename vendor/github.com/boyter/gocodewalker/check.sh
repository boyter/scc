#!/usr/bin/env bash

# SPDX-License-Identifier: MIT OR Unlicense

set -e

if [ -t 1 ]
then
  YELLOW='\033[0;33m'
  GREEN='\033[0;32m'
  RED='\033[0;31m'
  NC='\033[0m'
fi

yellow() { printf "${YELLOW}%s${NC}" "$*"; }
green() { printf "${GREEN}%s${NC}" "$*"; }
red() { printf "${RED}%s${NC}" "$*"; }

good() {
  echo "$(green "● success:")" "$@"
}

bad() {
  ret=$1
  shift
  echo "$(red "● failed:")" "$@"
  exit $ret
}

try() {
  "$@" || bad $? "$@" && good "$@"
}


echo "Running go fmt..."
gofmt -s -w ./..

echo "Running unit tests..."
go test -cover -race ./... || exit

{
  {
    opt='shopt -s extglob nullglob'
    gofmt='gofmt -s -w -l !(vendor)/ *.go'
    notice="    running: ( $opt; $gofmt; )"
    prefix="    $(yellow modified)"
    trap 'echo "$notice"; $opt; $gofmt | sed -e "s#^#$prefix #g"' EXIT
  }

  # comma separate linters (e.g. "gofmt,stylecheck")
  additional_linters="gofmt"
  try golangci-lint run --enable $additional_linters ./...
  trap '' EXIT
}

echo "Running fuzz tests..."
go test -fuzz=FuzzTestGitIgnore -fuzztime 30s

echo -e "${GREEN}================================================="
echo -e "ALL CHECKS PASSED"
echo -e "=================================================${NC}"
