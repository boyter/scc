#!/bin/bash
echo "Running go fmt..."
go fmt ./...
echo "Running unit tests..."
go test ./... || exit
go build || exit


GREEN='\033[1;32m'
RED='\033[0;31m'

if ./scc --not-a-real-option ; then
	echo -e "${RED}================================================="
    echo -e "TEST FAILED"
    echo -e "Invalid option should produce error code "
    echo -e "================================================="
    exit
fi

if ./scc ; then
    echo ""
else
    echo -e "${RED}================================================="
    echo -e "TEST FAILED"
    echo -e "Should run correctly with no directory specified"
    echo -e "================================================="
    exit
fi

if ./scc processor ; then
    echo ""
else
    echo -e "${RED}================================================="
    echo -e "TEST FAILED"
    echo -e "Should run correctly with directory specified"
    echo -e "================================================="
    exit
fi

if ./scc --avg-wage 1000 --binary --by-file --cocomo --debug --exclude-dir .git -f tabular -i go -c -d -M something -s name -w processor ; then
    echo ""
else
    echo -e "${RED}================================================="
    echo -e "TEST FAILED"
    echo -e "Should run correctly with multiple options"
    echo -e "================================================="
    exit
fi

echo -e "${GREEN}================================================="
echo -e "TESTS PASSED"
echo -e "================================================="