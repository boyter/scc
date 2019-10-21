#!/bin/bash

GREEN='\033[1;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Running go fmt..."
gofmt -s -w ./..

echo "Running unit tests..."
go test ./... || exit

# Race Detection
echo "Running race detection..."
if  go run --race . 2>&1 >/dev/null | grep -q "Found" ; then
    echo -e "${RED}======================================================="
    echo -e "FAILED race detection run 'go run --race .' to identify"
    echo -e "=======================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED race detection${NC}"
fi

echo "Building application..."
go build -ldflags="-s -w" || exit

echo '```' > LANGUAGES.md
./scc --languages >> LANGUAGES.md
echo '```' >> LANGUAGES.md

echo "Running integration tests..."

if ./scc --not-a-real-option > /dev/null ; then
    echo -e "${RED}================================================="
    echo -e "FAILED Invalid option should produce error code "
    echo -e "=======================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED invalid option test"
fi

if ./scc "examples/language/" --format cloc-yaml -o .tmp_scc_yaml >/dev/null && python <<EOS
import yaml,sys 
try:
    with open('.tmp_scc_yaml','r') as f:
        data = yaml.load(f.read())
        if type(data) is dict and data.keys(): 
            sys.exit(0)
        else:
            print('data was {}'.format(type(data)))
except Exception as e:
    pass
sys.exit(1)
EOS

then
	echo -e "${GREEN}PASSED cloc-yaml format test"
else
    echo -e "${RED}======================================================="
    echo -e "${RED}FAILED Should accept --format cloc-yaml and should generate valid output"
    echo -e "=======================================================${NC}"
    rm -f .tmp_scc_yaml
    exit
fi

if ./scc "examples/language/" --format cloc-yml -o .tmp_scc_yaml >/dev/null && python <<EOS
import yaml,sys
try:
    with open('.tmp_scc_yaml','r') as f:
        data = yaml.load(f.read())
        if type(data) is dict and data.keys():
            sys.exit(0)
        else:
            print('data was {}'.format(type(data)))
except Exception as e:
    pass
sys.exit(1)
EOS

then
	echo -e "${GREEN}PASSED cloc-yml format test"
else
    echo -e "${RED}======================================================="
    echo -e "${RED}FAILED Should accept --format cloc-yml and should generate valid output"
    echo -e "=======================================================${NC}"
    rm -f .tmp_scc_yaml
    exit
fi

if ./scc NOTAREALDIRECTORYORFILE > /dev/null ; then
    echo -e "${RED}================================================="
    echo -e "FAILED Invalid file/directory should produce error code "
    echo -e "=======================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED invalid file/directory test"
fi

if ./scc > /dev/null ; then
    echo -e "${GREEN}PASSED no directory specified test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with no directory specified"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc processor > /dev/null ; then
    echo -e "${GREEN}PASSED directory specified test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with directory specified"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --avg-wage 10000 --binary --by-file --no-cocomo --debug --exclude-dir .git -f tabular -i go -c -d -M something -s name -w processor > /dev/null ; then
    echo -e "${GREEN}PASSED multiple options test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with multiple options"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc -i sh -M "vendor|examples|p.*" > /dev/null ; then
    echo -e "${GREEN}PASSED regular expression ignore test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run with regular expression ignore"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc "examples/shared_extension/" | grep -q "Coq"; then
    echo -e "${GREEN}PASSED shared extension test 1"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to work with shared extension 1"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc "examples/shared_extension/" | grep -q "Verilog"; then
    echo -e "${GREEN}PASSED shared extension test 2"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to work with shared extension 2"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc "examples/shared_extension/" | grep -q "V "; then
    echo -e "${GREEN}PASSED shared extension test 3"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to work with shared extension 3"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --ci | grep -q "\-\-\-\-"; then
    echo -e "${GREEN}PASSED ci param test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to work with ci flag"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc "examples/denylist/" | grep -q "Java"; then
    echo -e "${RED}======================================================="
    echo -e "FAILED Should hit default .git denylist "
    echo -e "=======================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED denylist test"
fi

# Simple test to see if we get any concurrency issues
for i in {1..100}
do
    if ./scc > /dev/null ; then
        :
    else
        echo -e "${RED}======================================================="
        echo -e "FAILED Should not have concurrency issue"
        echo -e "=================================================${NC}"
        exit
    fi
done
echo -e "${GREEN}PASSED concurrency issue test"

if ./scc main.go > /dev/null ; then
    echo -e "${GREEN}PASSED file specified test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with a file is specified"
    echo -e "=================================================${NC}"
    exit
fi

# Multiple directory or file arguments
if ./scc main.go README.md | grep -q "Go " ; then
    echo -e "${GREEN}PASSED multiple file argument test 1"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should work with multiple file arguments 1"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc main.go README.md | grep -q "Markdown " ; then
    echo -e "${GREEN}PASSED multiple file argument test 2"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should work with multiple file arguments 2"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc processor scripts > /dev/null ; then
    echo -e "${GREEN}PASSED multiple directory specified test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should run correctly with multiple directory specified"
    echo -e "=================================================${NC}"
    exit
fi

if ./scc -v . | grep -q "skipping directory due to ignore: vendor" ; then
    echo -e "${GREEN}PASSED ignore file directory check"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED ignore file directory check"
    echo -e "=======================================================${NC}"
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
        echo -e "=======================================================${NC}"
        exit
    fi
done
echo -e "${GREEN}PASSED duplicates test"

# Ensure deterministic output
a=$(./scc .)
for i in {1..100}
do
    b=$(./scc .)
    if [ "$a" == "$b" ]; then
        :
    else
        echo -e "${RED}======================================================="
        echo -e "FAILED Runs should be deterministic"
        echo -e "=======================================================${NC}"
        exit
    fi
done
echo -e "${GREEN}PASSED deterministic test"

# Check for multiple regex via https://github.com/andyfitzgerald
a=$(./scc --not-match="(.*\.hex|.*\.d|.*\.o|.*\.csv|^(./)?[0-9]{8}_.*)" . | grep Estimated | md5sum)
b=$(./scc --not-match=".*\.hex" --not-match=".*\.d" --not-match=".*\.o" --not-match=".*\.csv" --not-match="^(./)?[0-9]{8}_.*" . | grep Estimated | md5sum)
if [ "$a" == "$b" ]; then
    echo -e "${GREEN}PASSED multiple regex test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED multiple regex test"
    echo -e "=================================================${NC}"
    exit
fi

# Regression issue https://github.com/boyter/scc/issues/82
a=$(./scc . | grep Total)
b=$(./scc ${PWD} | grep Total)
if [ "$a" == "$b" ]; then
    echo -e "${GREEN}PASSED git filter"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED git filter"
    echo -e "=================================================${NC}"
    exit
fi

# Turn off gitignore https://github.com/boyter/scc/issues/53
touch ignored.xml
a=$(./scc | grep Total)
b=$(./scc --no-gitignore | grep Total)
if [ "$a" == "$b" ]; then
    echo -e "${RED}================================================="
    echo -e "FAILED git ignore filter"
    echo -e "=================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED git ignore filter"
fi

# Regression issue https://github.com/boyter/scc/issues/115
if ./scc "examples/issue115/.test/file" 2>&1 >/dev/null | grep -q "Perl" ; then
    echo -e "${RED}======================================================="
    echo -e "FAILED hidden directory issue"
    echo -e "=======================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED hidden directory${NC}"
fi

a=$(./scc | grep Total)
b=$(./scc --no-ignore | grep Total)
if [ "$a" == "$b" ]; then
    echo "$a"
    echo "$b"
    echo -e "${RED}======================================================="
    echo -e "FAILED ignore filter"
    echo -e "=================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED ignore filter"
fi

if ./scc "examples/ignore/" | grep -q "Java "; then
    echo -e "${RED}======================================================="
    echo -e "FAILED multiple gitignore"
    echo -e "=======================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED multiple gitignore"
fi

touch ./examples/ignore/ignorefile.txt
a=$(./scc --by-file | grep ignorefile)
b=$(./scc --by-file --no-ignore | grep ignorefile)
if [ "$a" == "$b" ]; then
    echo "$a"
    echo "$b"
    echo -e "${RED}======================================================="
    echo -e "FAILED ignore recursive filter"
    echo -e "=================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED ignore recursive filter"
fi

touch ./examples/ignore/gitignorefile.txt
a=$(./scc --by-file | grep gitignorefile)
b=$(./scc --by-file --no-gitignore | grep gitignorefile)
if [ "$a" == "$b" ]; then
    echo "$a"
    echo "$b"
    echo -e "${RED}======================================================="
    echo -e "FAILED gitignore recursive filter"
    echo -e "=================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED gitignore recursive filter"
fi

if ./scc "examples/language/" --include-ext go | grep -q "Java "; then
    echo -e "${RED}======================================================="
    echo -e "FAILED include-ext option"
    echo -e "=======================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED include-ext option"
fi

a=$(./scc -i js ./examples/minified/)
b=$(./scc -i js -z ./examples/minified/)
if [ "$a" == "$b" ]; then
    echo "$a"
    echo "$b"
    echo -e "${RED}======================================================="
    echo -e "FAILED Minified check"
    echo -e "=================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED minified check"
fi

a=$(./scc -i js -z ./examples/minified/)
b=$(./scc -i js -z --no-min-gen ./examples/minified/)
if [ "$a" == "$b" ]; then
    echo "$a"
    echo "$b"
    echo -e "${RED}======================================================="
    echo -e "FAILED Minified ignored check"
    echo -e "=================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED minified ignored check"
fi

if ./scc ./examples/minified/ --no-min-gen | grep -q "0.000000"; then
    echo -e "${GREEN}PASSED removed min gen"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED removed min gen"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/minified/ -i js -z | grep -q "JavaScript (min)"; then
    echo -e "${GREEN}PASSED flagged as min"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED flagged as min"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/issue120/ -i java | grep -q "Perl"; then
    echo -e "${RED}======================================================="
    echo -e "FAILED extension param should ignore #!"
    echo -e "=======================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED extension param should ignore #!"
fi

if ./scc -z --min-gen-line-length 1 --no-min-gen . | grep -q "0.000000"; then
    echo -e "${GREEN}PASSED min gen line length"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED min gen line length"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --no-large --large-byte-count 0 ./examples/language | grep -q "0.000000"; then
    echo -e "${GREEN}PASSED no large byte test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED no large byte test"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --no-large --large-line-count 0 ./examples/language | grep -q "0.000000"; then
    echo -e "${GREEN}PASSED no large line test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED no large line test"
    echo -e "=======================================================${NC}"
    exit
fi

# Try out specific languages
for i in 'Bosque ' 'Flow9 ' 'Bitbucket Pipeline ' 'Docker ignore ' 'Q# ' 'Futhark ' 'Alloy ' 'Wren ' 'Monkey C ' 'Alchemist ' 'Luna ' 'ignore ' 'XML Schema ' 'Web Services' 'Go ' 'Java ' 'Boo ' 'License ' 'BASH ' 'C Shell ' 'Korn Shell ' 'Makefile ' 'Shell ' 'Zsh ' 'Rakefile ' 'Gemfile ' 'Dockerfile '
do
    if ./scc "examples/language/" | grep -q "$i "; then
        echo -e "${GREEN}PASSED $i Language Check"
    else
        echo -e "${RED}======================================================="
        echo -e "FAILED Should be able to find $i"
        echo -e "=======================================================${NC}"
        exit
    fi
done

echo -e "${NC}Cleaning up..."
rm ./scc
rm ./ignored.xml
rm .tmp_scc_yaml
rm ./examples/ignore/gitignorefile.txt
rm ./examples/ignore/ignorefile.txt

echo -e "${GREEN}================================================="
echo -e "ALL TESTS PASSED"
echo -e "=================================================${NC}"
