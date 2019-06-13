#!/bin/bash

echo "Running go fmt..."
gofmt -s -w ./..

echo "Running unit tests..."
go test ./... || exit

echo "Building application..."
go build -ldflags="-s -w" || exit

echo '```' > LANGUAGES.md
./scc --languages >> LANGUAGES.md
echo '```' >> LANGUAGES.md

echo "Running integration tests..."

GREEN='\033[1;32m'
RED='\033[0;31m'
NC='\033[0m'

if ./scc --not-a-real-option > /dev/null ; then
    echo -e "${RED}================================================="
    echo -e "FAILED Invalid option should produce error code "
    echo -e "======================================================="
    exit
else
    echo -e "${GREEN}PASSED invalid option test"
fi

if ./scc NOTAREALDIRECTORYORFILE > /dev/null ; then
    echo -e "${RED}================================================="
    echo -e "FAILED Invalid file/directory should produce error code "
    echo -e "======================================================="
    exit
else
    echo -e "${GREEN}PASSED invalid file/directory test"
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
    echo -e "======================================================="
    exit
fi

if ./scc --avg-wage 10000 --binary --by-file --cocomo --debug --exclude-dir .git -f tabular -i go -c -d -M something -s name -w processor > /dev/null ; then
    echo -e "${GREEN}PASSED multiple options test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with multiple options"
    echo -e "======================================================="
    exit
fi

if ./scc -i sh -M "vendor|examples|p.*" > /dev/null ; then
    echo -e "${GREEN}PASSED regular expression ignore test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run with regular expression ignore"
    echo -e "======================================================="
    exit
fi

if ./scc "examples/shared_extension/" | grep -q "Coq"; then
    echo -e "${GREEN}PASSED shared extension test 1"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to work with shared extension 1"
    echo -e "======================================================="
    exit
fi

if ./scc "examples/shared_extension/" | grep -q "SystemVerilog"; then
    echo -e "${GREEN}PASSED shared extension test 2"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to work with shared extension 2"
    echo -e "======================================================="
    exit
fi

if ./scc "examples/shared_extension/" | grep -q "V "; then
    echo -e "${GREEN}PASSED shared extension test 3"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to work with shared extension 3"
    echo -e "======================================================="
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

# Multiple directory or file arguments
if ./scc main.go README.md | grep -q "Go " ; then
    echo -e "${GREEN}PASSED multiple file argument test 1"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should work with multiple file arguments 1"
    echo -e "======================================================="
    exit
fi

if ./scc main.go README.md | grep -q "Markdown " ; then
    echo -e "${GREEN}PASSED multiple file argument test 2"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should work with multiple file arguments 2"
    echo -e "======================================================="
    exit
fi

if ./scc processor scripts > /dev/null ; then
    echo -e "${GREEN}PASSED multiple directory specified test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with multiple directory specified"
    echo -e "================================================="
    exit
fi


# Try out duplicates
for i in {1..100}
do
    if ./scc -d "examples/duplicates/" | grep -e "Java" | grep -q -e " 1 "; then
        :
    else
        echo -e "${RED}======================================================="
        echo -e "FAILED Duplicates should be consistent"
        echo -e "======================================================="
        exit
    fi
done
echo -e "${GREEN}PASSED duplicates test"

# Check for multiple regex via https://github.com/andyfitzgerald
a=$(./scc --not-match="(.*\.hex|.*\.d|.*\.o|.*\.csv|^(./)?[0-9]{8}_.*)" . | grep Estimated | md5sum)
b=$(./scc --not-match=".*\.hex" --not-match=".*\.d" --not-match=".*\.o" --not-match=".*\.csv" --not-match="^(./)?[0-9]{8}_.*" . | grep Estimated | md5sum)
if [ "$a" == "$b" ]; then
    echo -e "${GREEN}PASSED multiple regex test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED multiple regex test"
    echo -e "================================================="
    exit
fi

# Try out specific languages
for i in 'Bosque ' 'Flow9 ' 'Bitbucket Pipeline ' 'Docker ignore ' 'Q# ' 'Futhark ' 'Alloy ' 'Wren ' 'Monkey C ' 'Alchemist '
do
    if ./scc "examples/language/" | grep -q "$i "; then
        echo -e "${GREEN}PASSED $i Language Check"
    else
        echo -e "${RED}======================================================="
        echo -e "FAILED Should be able to find $i"
        echo -e "======================================================="
        exit
    fi
done

echo -e "${NC}Cleaning up..."
rm ./scc

echo -e "${GREEN}================================================="
echo -e "ALL TESTS PASSED"
echo -e "================================================="
