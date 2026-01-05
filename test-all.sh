#!/bin/bash

# make sure this script can be executed from any dir
cd $(dirname $0)

GREEN='\033[1;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Running go generate..."
go generate

echo "Running go fmt..."
go fmt ./...

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


echo "Building HTML report..."

./scc --format html -a --by-file -i go -o SCC-OUTPUT-REPORT.html

echo "Running integration tests..."

## NB you need to have pyyaml installed via pip install pyyaml for this to work
#if ./scc "examples/language/" --format cloc-yaml -o .tmp_scc_yaml >/dev/null && python <<EOS
#import yaml,sys
#try:
#    with open('.tmp_scc_yaml','r') as f:
#        data = yaml.load(f.read())
#        if type(data) is dict and data.keys():
#            sys.exit(0)
#        else:
#            print('data was {}'.format(type(data)))
#except Exception as e:
#    pass
#sys.exit(1)
#EOS
#
#then
#	echo -e "${GREEN}PASSED cloc-yaml format test"
#else
#    echo -e "${RED}======================================================="
#    echo -e "${RED}FAILED Should accept --format cloc-yaml and should generate valid output"
#    echo -e "=======================================================${NC}"
#    rm -f .tmp_scc_yaml
#    exit
#fi
#
#if ./scc "examples/language/" --format cloc-yml -o .tmp_scc_yaml >/dev/null && python <<EOS
#import yaml,sys
#try:
#    with open('.tmp_scc_yaml','r') as f:
#        data = yaml.load(f.read())
#        if type(data) is dict and data.keys():
#            sys.exit(0)
#        else:
#            print('data was {}'.format(type(data)))
#except Exception as e:
#    pass
#sys.exit(1)
#EOS
#
#then
#	echo -e "${GREEN}PASSED cloc-yml format test"
#else
#    echo -e "${RED}======================================================="
#    echo -e "${RED}FAILED Should accept --format cloc-yml and should generate valid output"
#    echo -e "=======================================================${NC}"
#    rm -f .tmp_scc_yaml
#    exit
#fi

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

if ./scc --avg-wage 10000 --binary --by-file --no-cocomo --no-size --size-unit si --include-symlinks --debug --exclude-dir .git -f tabular -i go -c -d -M something -s name -w processor > /dev/null ; then
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

a=$(./scc -i js ./examples/minified/ --no-scc-ignore)
b=$(./scc -i js -z ./examples/minified/ --no-scc-ignore)
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

a=$(./scc -i js -z ./examples/minified/ --no-scc-ignore)
b=$(./scc -i js -z --no-min-gen ./examples/minified/ --no-scc-ignore)
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

a=$(./scc ./examples/symlink/ --no-scc-ignore)
b=$(./scc --include-symlinks ./examples/symlink/ --no-scc-ignore)
if [ "$a" == "$b" ]; then
    echo "$a"
    echo "$b"
    echo -e "${RED}======================================================="
    echo -e "FAILED symlink check"
    echo -e "=================================================${NC}"
    exit
else
    echo -e "${GREEN}PASSED minified ignored check"
fi

if ./scc ./examples/minified/ --no-min-gen --no-scc-ignore | grep -q "\$0"; then
    echo -e "${GREEN}PASSED removed min"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED removed min"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/generated/ --no-min-gen --no-scc-ignore | grep -q "\$0"; then
    echo -e "${GREEN}PASSED removed gen"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED removed gen"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc -z ./examples/generated/ --no-scc-ignore | grep -q "C Header (gen)"; then
    echo -e "${GREEN}PASSED flagged as gen"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED flagged as gen"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/minified/ -i js -z --no-scc-ignore | grep -q "JavaScript (mi"; then
    echo -e "${GREEN}PASSED flagged as min"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED flagged as min"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc -z --min-gen-line-length 1 --no-min-gen . | grep -q "\$0"; then
    echo -e "${GREEN}PASSED min gen line length"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED min gen line length"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --no-large --large-byte-count 0 ./examples/language | grep -q "\$0"; then
    echo -e "${GREEN}PASSED no large byte test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED no large byte test"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --no-large --large-line-count 0 ./examples/language | grep -q "\$0"; then
    echo -e "${GREEN}PASSED no large line test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED no large line test"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --format html | grep -q "html"; then
    echo -e "${GREEN}PASSED html output test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to output to html"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --format html-table | grep -q "table"; then
    echo -e "${GREEN}PASSED html-table output test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to output to html-table"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --format sql | grep -q "create table metadata"; then
    echo -e "${GREEN}PASSED sql output test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to output to sql"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --format sql-insert | grep -q "insert into t values"; then
    echo -e "${GREEN}PASSED sql-insert output test"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED Should be able to output to sql-insert"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/countas/ --count-as jsp:html | grep -q "HTML"; then
    echo -e "${GREEN}PASSED counted JSP as HTML"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED counted JSP as HTML"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/countas/ --count-as JsP:html | grep -q "HTML"; then
    echo -e "${GREEN}PASSED counted JSP as HTML case"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED counted JSP as HTML case"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/countas/ --count-as jsp:j2 | grep -q "Jinja"; then
    echo -e "${GREEN}PASSED counted JSP as Jinja"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED counted JSP as Jinja"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/countas/ --count-as jsp:html,new:java | grep -q "Java"; then
    echo -e "${GREEN}PASSED counted new as Java"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED counted new as Java"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc ./examples/countas/ --count-as jsp:html,new:"C Header" | grep -q "C Header"; then
    echo -e "${GREEN}PASSED counted new as C Header"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED counted new as C Header"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --file-gc-count 10 ./examples/duplicates/ -v | grep -q "read file limit exceeded GC re-enabled"; then
    echo -e "${GREEN}PASSED gc file count"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED gc file count"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc -f json | grep -q "Bytes"; then
    echo -e "${GREEN}PASSED json bytes check"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED json bytes check"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc | grep -q "megabytes"; then
    echo -e "${GREEN}PASSED bytes check"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED bytes check"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --remap-unknown "-*- C++ -*-":"C Header" ./examples/remap/unknown | grep -q "C Header"; then
    echo -e "${GREEN}PASSED remap unknown"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED remap unknown"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --remap-all "-*- C++ -*-":"C Header" ./examples/remap/java.java | grep -q "C Header"; then
    echo -e "${GREEN}PASSED remap all"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED remap all"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --cocomo-project-type organic | grep -q "organic"; then
    echo -e "${GREEN}PASSED cocomo organic"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED cocomo organic"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --cocomo-project-type doesnotexist | grep -q "organic"; then
    echo -e "${GREEN}PASSED cocomo fallback"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED cocomo fallback"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --cocomo-project-type semi-detached | grep -q "semi-detached"; then
    echo -e "${GREEN}PASSED cocomo semi-detached"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED cocomo semi-detached"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --cocomo-project-type embedded | grep -q "embedded"; then
    echo -e "${GREEN}PASSED cocomo embedded"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED cocomo embedded"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --cocomo-project-type custom,1,1,1,1 | grep -q "custom,1,1,1,1"; then
    echo -e "${GREEN}PASSED cocomo custom,1,1,1,1"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED cocomo custom,1,1,1,1"
    echo -e "=======================================================${NC}"
    exit
fi

if ./scc --cocomo-project-type custom,1,1,1 | grep -q "organic"; then
    echo -e "${GREEN}PASSED cocomo custom fallback"
else
    echo -e "${RED}======================================================="
    echo -e "FAILED cocomo custom fallback"
    echo -e "=======================================================${NC}"
    exit
fi

echo -e  "${NC}Checking compile targets..."

echo "   darwin..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w"
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w"
echo "   windows..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w"
GOOS=windows GOARCH=386 go build -ldflags="-s -w"
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w"
echo "   linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"
GOOS=linux GOARCH=386 go build -ldflags="-s -w"
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w"
GOOS=linux GOARCH=riscv64 go build -ldflags="-s -w"
GOOS=linux GOARCH=loong64 go build -ldflags="-s -w"

echo -e "${NC}Cleaning up..."
rm ./scc
rm ./scc.exe
rm .tmp_scc_yaml
rm ./code.db


echo -e "${GREEN}================================================="
echo -e "ALL TESTS PASSED"
echo -e "=================================================${NC}"
