#!/bin/bash

echo "Running go fmt..."
gofmt -s -w ./..

echo "Running unit tests..."
go test ./... || exit

echo "Building application..."
go build -ldflags="-s -w" || exit

echo "Running integration tests..."

GREEN='\033[1;32m'
RED='\033[0;31m'
NC='\033[0m'

if ./scc --not-a-real-option > /dev/null ; then
	echo -e "${RED}================================================="
    echo -e "FAILED Invalid option should produce error code "
    echo -e "================================================="
    exit
else
    echo -e "${GREEN}PASSED invalid option test"
fi

if ./scc > /dev/null ; then
    echo -e "${GREEN}PASSED no directory specified test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with no directory specified"
    echo -e "======================================================="
    exit
fi

if ./scc processor > /dev/null ; then
    echo -e "${GREEN}PASSED directory specified test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with directory specified"
    echo -e "================================================="
    exit
fi

if ./scc --avg-wage 10000 --binary --by-file --cocomo --debug --exclude-dir .git -f tabular -i go -c -d -M something -s name -w processor > /dev/null ; then
    echo -e "${GREEN}PASSED multiple options test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with multiple options"
    echo -e "================================================="
    exit
fi

if ./scc -i sh -M "vendor|examples|p.*" > /dev/null ; then
    echo -e "${GREEN}PASSED regular expression ignore test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run with regular expression ignore"
    echo -e "================================================="
    exit
fi

# Simple test to see if we get any concurrency issues
for i in {1..100}
do
    if ./scc > /dev/null ; then
        :
    else
        echo -e "${RED}======================================================="
        echo -e "FAILED Should not have concurrency issue"
        echo -e "================================================="
        exit
    fi
done
echo -e "${GREEN}PASSED concurrency issue test"

if ./scc main.go > /dev/null ; then
    echo -e "${GREEN}PASSED file specified test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with a file is specified"
    echo -e "================================================="
    exit
fi

echo -e "${NC}Cleaning up..."
rm ./scc

echo -e "${GREEN}================================================="
echo -e "ALL TESTS PASSED"
echo -e "================================================="