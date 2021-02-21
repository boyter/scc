Sloc Cloc and Code (scc)
------------------------

<img alt="scc" src=https://github.com/boyter/scc/raw/master/scc.jpg>

A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform COCOMO calculation like sloccount and to estimate code complexity similar to cyclomatic complexity calculators. In short one tool to rule them all.

Also it has a very short name which is easy to type `scc`. 

If you don't like sloc cloc and code feel free to use the name `Succinct Code Counter`.

[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)
[![Go Report Card](https://goreportcard.com/badge/github.com/boyter/scc)](https://goreportcard.com/report/github.com/boyter/scc)
[![Coverage Status](https://coveralls.io/repos/github/boyter/scc/badge.svg?branch=master)](https://coveralls.io/github/boyter/scc?branch=master)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/)](https://github.com/boyter/scc/)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

Dual-licensed under MIT or the [UNLICENSE](http://unlicense.org).

### Install

#### Go Get

If you are comfortable using Go and have >= 1.13 installed:

`$ go get -u github.com/boyter/scc/`

#### Snap

A [snap install](https://snapcraft.io/scc) exists thanks to [Ricardo](https://feliciano.tech/).

`$ sudo snap install scc`

#### Homebrew

Or if you have [homebrew](https://brew.sh/) installed

`$ brew install scc`

#### Scoop

Or if you are using [Scoop](https://scoop.sh/) on Windows

`$ scoop install scc`

#### Manual

Binaries for Windows, GNU/Linux and macOS for both i386 and x86_64 machines are available from the [releases](https://github.com/boyter/scc/releases) page.

#### GitHub Action workflow

https://github.com/marketplace/actions/scc-docker-action https://github.com/iRyanBell/scc-docker-action

_.github/workflows/main.yml_

```
on: [push]

jobs:
  scc_job:
    runs-on: ubuntu-latest
    name: A job to count the lines of code.
    steps:
      - name: Checkout
        uses: actions/checkout@v2
      - name: Get the lines of code.
        id: scc
        uses: iryanbell/scc-docker-action@v1.0.0
        with:
          args: ${{ env.workspace }} -i js,go,html,css
```

#### Other

If you would like to assist with getting `scc` added into apt/chocolatey/etc... please submit a PR or at least raise an issue with instructions.

### Background

Read all about how it came to be along with performance benchmarks,

 - https://boyter.org/posts/sloc-cloc-code/
 - https://boyter.org/posts/why-count-lines-of-code/
 - https://boyter.org/posts/sloc-cloc-code-revisited/
 - https://boyter.org/posts/sloc-cloc-code-performance/
 - https://boyter.org/posts/sloc-cloc-code-performance-update/

Some reviews of `scc`

 - https://nickmchardy.com/2018/10/counting-lines-of-code-in-koi-cms.html
 - https://www.feliciano.tech/blog/determine-source-code-size-and-complexity-with-scc/
 - https://metaredux.com/posts/2019/12/13/counting-lines.html

A talk given at the first GopherCon AU about `scc` (press S to see speaker notes)

 - https://boyter.org/static/gophercon-syd-presentation/
 - https://www.youtube.com/watch?v=jd-sjoy3GZo

For performance see the [Performance](https://github.com/boyter/scc#performance) section

Other similar projects,

 - [SLOCCount](https://www.dwheeler.com/sloccount/) the original sloc counter
 - [cloc](https://github.com/AlDanial/cloc), inspired by SLOCCount; implemented in Perl for portability
 - [gocloc](https://github.com/hhatto/gocloc) a sloc counter in Go inspired by tokei
 - [loc](https://github.com/cgag/loc) rust implementation similar to tokei but often faster
 - [loccount](https://gitlab.com/esr/loccount) Go implementation written and maintained by ESR
 - [ployglot](https://github.com/vmchale/polyglot) ATS sloc counter
 - [tokei](https://github.com/XAMPPRocky/tokei) fast, accurate and written in rust

Interesting reading about other code counting projects tokei, loc, polyglot and loccount

 - https://www.reddit.com/r/rust/comments/59bm3t/a_fast_cloc_replacement_in_rust/
 - https://www.reddit.com/r/rust/comments/82k9iy/loc_count_lines_of_code_quickly/
 - http://blog.vmchale.com/article/polyglot-comparisons
 - http://esr.ibiblio.org/?p=8270

Further reading about processing files on the disk performance

 - https://blog.burntsushi.net/ripgrep/
 
Using `scc` to process 40 TB of files from Github/Bitbucket/Gitlab

 - https://boyter.org/posts/an-informal-survey-of-10-million-github-bitbucket-gitlab-projects/

### Pitch

Why use `scc`?

 - It is very fast and gets faster the more CPU you throw at it
 - Accurate
 - Works very well across multiple platforms without slowdown (Windows, Linux, macOS)
 - Large language support
 - Can ignore duplicate files
 - Has complexity estimations
 - You need to tell the difference between Coq and Verilog in the same directory
 - cloc yaml output support so potentially a drop in replacement for some users
 - Can identify or ignore minified files
 - Able to identify many #! files
 - Can ignore large files by lines or bytes

Why not use `scc`?

 - You don't like Go for some reason
 - It cannot count D source with different nested multi-line comments correctly https://github.com/boyter/scc/issues/27

### Differences

There are some important differences between `scc` and other tools that are out there. Here are a few important ones for you to consider.

Blank lines inside comments are counted as comments. While the line is technically blank the decision was made that 
once in a comment everything there should be considered a comment until that comment is ended. As such the following,
 
```
/* blank lines follow


*/
```

Would be counted as 4 lines of comments. This is noticeable when comparing scc's output to other tools on large
repositories.

`scc` is also able count verbatim strings correctly. For example in C# the following,

```
private const string BasePath = @"a:\";
// The below is returned to the user as a version
private const string Version = "1.0.0";
```

Because of the prefixed @ this string ends at the trailing " by ignoring the escape character \ and as such should be 
counted as 2 code lines and 1 comment. Some tools are unable to
deal with this and instead count up to the "1.0.0" as a string which can cause the middle comment to be counted as
code rather than a comment.

`scc` will also tell you the number of bytes it has processed (for most output formats) allowing you to estimate the
cost of running some static analysis tools. 

### Usage

Command line usage of `scc` is designed to be as simple as possible.
Full details can be found in `scc --help` or `scc -h`. Note that the below reflects the state of master not a release.

```
Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation.
Version 3.0.0 (beta)
Ben Boyter <ben@boyter.org> + Contributors

Usage:
  scc [flags] [files or directories]

Flags:
      --avg-wage int                average wage value used for basic COCOMO calculation (default 56286)
      --binary                      disable binary file detection
      --by-file                     display output for every file
      --ci                          enable CI output settings where stdout is ASCII
      --count-as string             count extension as language [e.g. jsp:htm,chead:"C Header" maps extension jsp to html and chead to C Header]
      --debug                       enable debug output
      --exclude-dir strings         directories to exclude (default [.git,.hg,.svn])
      --file-gc-count int           number of files to parse before turning the GC on (default 10000)
  -f, --format string               set output format [tabular, wide, json, csv, csv-stream, cloc-yaml, html, html-table, sql, sql-insert] (default "tabular")
      --format-multi string         have multiple format output overriding --format [e.g. tabular:stdout,csv:file.csv,json:file.json]
      --gen                         identify generated files
      --generated-markers strings   string markers in head of generated files (default [do not edit])
  -h, --help                        help for scc
  -i, --include-ext strings         limit to file extensions [comma separated list: e.g. go,java,js]
      --include-symlinks            if set will count symlink files
  -l, --languages                   print supported languages and extensions
      --large-byte-count int        number of bytes a file can contain before being removed from output (default 1000000)
      --large-line-count int        number of lines a file can contain before being removed from output (default 40000)
      --min                         identify minified files
  -z, --min-gen                     identify minified or generated files
      --min-gen-line-length int     number of bytes per average line for file to be considered minified or generated (default 255)
      --no-cocomo                   remove COCOMO calculation output
  -c, --no-complexity               skip calculation of code complexity
  -d, --no-duplicates               remove duplicate files from stats and output
      --no-gen                      ignore generated files in output (implies --gen)
      --no-gitignore                disables .gitignore file logic
      --no-ignore                   disables .ignore file logic
      --no-large                    ignore files over certain byte and line size set by max-line-count and max-byte-count
      --no-min                      ignore minified files in output (implies --min)
      --no-min-gen                  ignore minified or generated files in output (implies --min-gen)
      --no-size                     remove size calculation output
  -M, --not-match stringArray       ignore files and directories matching regular expression
  -o, --output string               output filename (default stdout)
      --remap-all string            inspect every file and remap by checking for a string and remapping the language [e.g. "-*- C++ -*-":"C Header"]
      --remap-unknown string        inspect files of unknown type and remap by checking for a string and remapping the language [e.g. "-*- C++ -*-":"C Header"]
      --size-unit string            set size unit [si, binary, mixed, xkcd-kb, xkcd-kelly, xkcd-imaginary, xkcd-intel, xkcd-drive, xkcd-bakers] (default "si")
  -s, --sort string                 column to sort by [files, name, lines, blanks, code, comments, complexity] (default "files")
      --sql-project string          use supplied name as the project identifier for the current run. Only valid with the --format sql or sql-insert option
  -t, --trace                       enable trace output (not recommended when processing multiple files)
  -v, --verbose                     verbose output
      --version                     version for scc
  -w, --wide                        wider output with additional statistics (implies --complexity)
```

Output should look something like the below for the redis project

```
$ scc
───────────────────────────────────────────────────────────────────────────────
Language                 Files     Lines   Blanks  Comments     Code Complexity
───────────────────────────────────────────────────────────────────────────────
C                          285    168526    19069     29354   120103      30054
C Header                   213     31573     3569      6804    21200       1627
TCL                        118     21776     2346      1290    18140       1870
Shell                       44      1614      213       319     1082        183
Autoconf                    22     10871     1038      1326     8507        953
Lua                         20       525       68        70      387         65
Markdown                    16      2591      681         0     1910          0
Makefile                    11      1305      256       113      936         58
Ruby                        10       778       78        71      629        115
gitignore                   10       161       16         0      145          0
YAML                         6       572       40         6      526          0
HTML                         5      9658     2928        12     6718          0
C++                          4       286       48        14      224         31
License                      4       100       20         0       80          0
Plain Text                   3       185       26         0      159          0
CMake                        2       214       43         3      168          4
CSS                          2       107       16         0       91          0
Python                       2       219       12         6      201         34
Systemd                      2        80        6         0       74          0
BASH                         1       118       14         5       99         31
Batch                        1        28        2         0       26          3
C++ Header                   1         9        1         3        5          0
Extensible Styleshe…         1        10        0         0       10          0
Smarty Template              1        44        1         0       43          5
m4                           1       562      116        53      393          0
───────────────────────────────────────────────────────────────────────────────
Total                      785    251912    30607     39449   181856      35033
───────────────────────────────────────────────────────────────────────────────
Estimated Cost to Develop $6,372,492
Estimated Schedule Effort 27.800436 months
Estimated People Required 20.364495
───────────────────────────────────────────────────────────────────────────────
Processed 8665286 bytes, 8.665 megabytes (SI)
───────────────────────────────────────────────────────────────────────────────
```

Note that you don't have to specify the directory you want to run against. Running `scc` will assume you want to run against the current directory.

You can also run against multiple files or directories `scc directory1 directory2 file1 file2` with the results aggregated in the output.

### Ignore Files

`scc` mostly supports .ignore files inside directories that it scans. This is similar to how ripgrep, ag and tokei work. .ignore files are 100% the same as .gitignore files with the same syntax, and as such `scc` will ignore files and directories listed in them. You can add .ignore files to ignore things like vendored dependency checked in files and such. The idea is allowing you to add a file or folder to git and have ignored in the count.

### Interesting Use Cases

Used inside Intel Nemu Hypervisor to track code changes between revisions https://github.com/intel/nemu/blob/topic/virt-x86/tools/cloc-change.sh#L9
Appears to also be used inside both http://codescoop.com/ https://pinpoint.com/ https://github.com/chaoss/grimoirelab-graal

It also is used to count code and guess language types in https://searchcode.com/ which makes it one of the most frequently run code counters in the world. 

You can also hook scc into your gitlab pipeline https://gitlab.com/guided-explorations/ci-cd-plugin-extensions/ci-cd-plugin-extension-scc

### Features

`scc` uses a small state machine in order to determine what state the code is when it reaches a newline `\n`. As such it is aware of and able to count

 - Single Line Comments
 - Multi Line Comments
 - Strings
 - Multi Line Strings
 - Blank lines

Because of this it is able to accurately determine if a comment is in a string or is actually a comment.

It also attempts to count the complexity of code. This is done by checking for branching operations in the code. For example, each of the following `for if switch while else || && != ==` if encountered in Java would increment that files complexity by one.


### Complexity Estimates

Lets take a minute to discuss the complexity estimate itself.

The complexity estimate is really just a number that is only comparable to files in the same language. It should not be used to compare languages directly without weighting them. The reason for this is that its calculated by looking for branch and loop statements in the code and incrementing a counter for that file.

Because some languages don't have loops and instead use recursion they can have a lower complexity count. Does this mean they are less complex? Probably not, but the tool cannot see this because it does not build an AST of the code as it only scans through it.

Generally though the complexity there is to help estimate between projects written in the same language, or for finding the most complex file in a project `scc --by-file -s complexity` which can be useful when you are estimating on how hard something is to maintain, or when looking for those files that should probably be refactored.

### Large File Detection

You can have `scc` exclude large files from the output. 

The option to do so is `--no-large` which by default will exclude files over 1,000,000 bytes or 40,000 lines.

You can control the size of either value using `--large-byte-count` or `--large-line-count`.

For example to exclude files over 1,000 lines and 50kb you could use the following,

`scc --no-large --large-byte-count 50000 --large-line-count 1000`

### Minified/Generated File Detection

You can have `scc` identify and optionally remove files identified as being minified or generated from the output.

You can do so by enabling the `-z` flag like so `scc -z` which will identify any file with an average line byte size >= 255 (by default) as being minified.

Minified files appear like so in the output.

```
$ scc --no-cocomo -z ./examples/minified/jquery-3.1.1.min.js
───────────────────────────────────────────────────────────────────────────────
Language                 Files     Lines   Blanks  Comments     Code Complexity
───────────────────────────────────────────────────────────────────────────────
JavaScript (min)             1         4        0         1        3         17
───────────────────────────────────────────────────────────────────────────────
Total                        1         4        0         1        3         17
───────────────────────────────────────────────────────────────────────────────
Processed 86709 bytes, 0.087 megabytes (SI)
───────────────────────────────────────────────────────────────────────────────
```

Minified files are indicated with the text `(min)` after the language name.

Generated files are indicated with the text `(gen)` after the language name.

You can control the average line byte size using `--min-gen-line-length` such as `scc -z --min-gen-line-length 1`. Please note you need `-z` as modifying this value does not imply minified detection.

You can exclude minified files from the count totally using the flag `--no-min-gen`. Files which match the minified check will be excluded from the output.

### Remapping

Some files may not have an extension. They will be checked to see if they are a #! file. If they are then the language will be remapped to the 
correct language. Otherwise, it will not process.

However, you may have the situation where you want to remap such files based on a string inside it. To do so you can use `--remap-unknown`

```
 scc --remap-unknown "-*- C++ -*-":"C Header"
```

The above will inspect any file with no extension looking for the string `-*- C++ -*-` and if found remap the file to be counted using the C Header rules. 
You can have multiple remap rules if required,

```
 scc --remap-unknown "-*- C++ -*-":"C Header","other":"Java"
```

There is also the `--remap-all` parameter which will remap all files.

Note that in all cases if the remap rule does not apply normal #! rules will apply.

### Output Formats

By default `scc` will output to the console. However you can produce output in other formats if you require.

The different options are `tabular, wide, json, csv, csv-stream, cloc-yaml, html, html-table, sql, sql-insert`. 

Note that you can write `scc` output to disk using the `-o, --output` option. This allows you to specify a file to
write your output to. For example `scc -f html -o output.html` will run `scc` against the current directory, and output
the results in html to the file `output.html`.

You can also write to multiple output files, or multiple types to stdout if you want using the `--format-multi` option. This is 
most useful when working in CI/CD systems where you want HTML reports as an artefact while also displaying the counts in stdout. 

```
scc --format-multi "tabular:stdout,html:output.html,csv:output.csv"
```

The above will run against the current directory, outputting to standard output the the default output, as well as writing
to output.html and output.csv with the appropiate formats.

#### Tabular 

This is the default output format when scc is run.

#### Wide 

Wide produces some additional information which is the complexity/lines metric. This can be useful when trying to
identify the most complex file inside a project based on the complexity estimate.

#### JSON

JSON produces JSON output. Mostly designed to allow `scc` to feed into other programs.

Note that this format will give you the byte size of every file it `scc` reads allowing you to get a breakdown of the
number of bytes processed.

#### CSV

CSV as an option is good for importing into a spreadsheet for analysis. 

Note that this format will give you the byte size of every file it `scc` reads allowing you to get a breakdown of the
number of bytes processed.

#### CSV-Stream

csv-stream is an option useful for processing very large repositories where you are likely to run into memory issues. It's output format is 100% the same as CSV. 

Note that you should not use this with the `format-multi` option as it will always print to standard output, and because of how it works will negate the memory saving it normally gains.
savings that this option provides. Note that there is no sort applied with this option. 

#### cloc-yaml 

Is a drop in replacement for cloc using its yaml output option. This is quite often used for passing into other 
build systems and can help with replacing cloc if required.

```
$ scc -f cloc-yml processor
# https://github.com/boyter/scc/
header:
  url: https://github.com/boyter/scc/
  version: 2.11.0
  elapsed_seconds: 0.008
  n_files: 21
  n_lines: 6562
  files_per_second: 2625
  lines_per_second: 820250
Go:
  name: Go
  code: 5186
  comment: 273
  blank: 1103
  nFiles: 21
SUM:
  code: 5186
  comment: 273
  blank: 1103
  nFiles: 21

$ cloc --yaml processor
      21 text files.
      21 unique files.
       0 files ignored.

---
# http://cloc.sourceforge.net
header :
  cloc_url           : http://cloc.sourceforge.net
  cloc_version       : 1.60
  elapsed_seconds    : 0.196972846984863
  n_files            : 21
  n_lines            : 6562
  files_per_second   : 106.613679608407
  lines_per_second   : 33314.2364566841
Go:
  nFiles: 21
  blank: 1137
  comment: 606
  code: 4819
SUM:
  blank: 1137
  code: 4819
  comment: 606
  nFiles: 21
```

#### HTML and HTML-TABLE

The HTML output options produce a minimal html report using a table that is either standalone `html` or as just a table `html-table`
which can be injected into your own HTML pages. The only difference between the two is that the `html` option includes 
html head and body tags with minimal styling.

The markup is designed to allow your own custom styles to be applied. An example report
[is here to view](SCC-OUTPUT-REPORT.html).

Note that the HTML options follow the command line options, so you can use `scc --by-file -f html` to produce a report with every
file and not just the summary.

Note that this format if it has the `--by-file` option will give you the byte size of every file it `scc` reads allowing you to get a breakdown of the
number of bytes processed.

#### SQL and SQL-Insert

The SQL output format "mostly" compatible with cloc's SQL output format https://github.com/AlDanial/cloc#sql-

While all queries on the cloc documentation should work as expected, you will not be able to append output from `scc` and `cloc` into the same database. This is because the table format is slightly different
to account for scc including complexity counts and bytes.

The difference between `sql` and `sql-insert` is that `sql` will include table creation while the latter will only have the insert commands.

Usage is 100% the same as any other `scc` command but sql output will always contain per file details. You can compute totals yourself using SQL.

The below will run scc against the current directory, name the ouput as the project scc and then pipe the output to sqlite to put into the database code.db

```
scc --format sql --sql-project scc . | sqlite3 code.db
```

Assuming you then wanted to append another project

```
scc --format sql-insert --sql-project redis . | sqlite3 code.db
```

You could then run SQL against the database,

```
sqlite3 code.db 'select project,file,max(nCode) as nL from t
                         group by project order by nL desc;'
```

See the cloc documentation for more examples.


### Performance

Generally `scc` will the fastest code counter compared to any I am aware of and have compared against. The below comparisons are taken from the fastest alternative counters. See `Other similar projects` above to see all of the other code counters compared against. It is designed to scale to as many CPU's cores as you can provide.

However if you want greater performance and you have RAM to spare you can disable the garbage collector like the following on linux `GOGC=-1 scc .` which should speed things up considerably. For some repositories turning off the code complexity calculation via `-c` can reduce runtime as well.

Benchmarks are run on fresh 32 Core CPU Optimised Digital Ocean Virtual Machine 2020/08/19 all done using [hyperfine](https://github.com/sharkdp/hyperfine) with 3 warm-up runs and 10 timed runs.

```
scc v2.13.0
tokei v12.0.4
loc v0.5.0
polyglot v0.5.25
```

See https://github.com/boyter/scc/blob/master/benchmark.sh to see how the benchmarks are run.


#### Redis https://github.com/antirez/redis/

| Program | Runtime |
|---|---|
| scc | 17.9 ms ±   1.8 ms |
| scc (no complexity) | 14.6 ms ±   1.3 ms |
| tokei | 29.3 ms ±   6.8 ms |
| loc | 50.7 ms ±  21.8 ms |
| polyglot | 19.6 ms ±   0.7 ms |

#### CPython https://github.com/python/cpython

| Program | Runtime |
|---|---|
| scc | 52.3 ms ±   4.9 ms |
| scc (no complexity) | 45.2 ms ±   4.4 ms |
| tokei | 92.2 ms ±   7.4 ms |
| loc | 132.7 ms ±  17.7 ms |
| polyglot | 88.0 ms ±   1.6 ms |

#### Linux Kernel https://github.com/torvalds/linux

| Program | Runtime |
|---|---|
| scc | 584.8 ms ±  15.8 ms |
| scc (no complexity) | 420.7 ms ±  11.3 ms |
| tokei | 871.9 ms ±  23.1 ms |
| loc | 1.999 s ±  0.057 s |
| polyglot | 1.171 s ±  0.043 s |

If you enable duplicate detection expect performance to fall by about 20% in `scc`.

Performance is tracked over each release and presented below.

<img alt="scc" src=https://github.com/boyter/scc/raw/master/performance-over-time.png>

https://jsfiddle.net/0ardq2g7/

### CI/CD Support

Some CI/CD systems which will remain nameless do not work very well with the box-lines used by `scc`. To support those systems better there is an option `--ci` which will change the default output to ASCII only.

```
$ scc --ci main.go
-------------------------------------------------------------------------------
Language                 Files     Lines   Blanks  Comments     Code Complexity
-------------------------------------------------------------------------------
Go                           1       272        7         6      259          4
-------------------------------------------------------------------------------
Total                        1       272        7         6      259          4
-------------------------------------------------------------------------------
Estimated Cost to Develop $6,539
Estimated Schedule Effort 2.268839 months
Estimated People Required 0.341437
-------------------------------------------------------------------------------
Processed 5674 bytes, 0.006 megabytes (SI)
-------------------------------------------------------------------------------
```

The `--format-multi` option is especially useful in CI/CD where you want to get multiple output formats useful for storage or reporting.

### Development

If you want to hack away feel free! PR's are accepted. Some things to keep in mind. If you want to change a language definition you need to update `languages.json` and then run `go generate` which will convert it into the `processor/constants.go` file.

For all other changes ensure you run all tests before submitting. You can do so using `go test ./...`. However for maximum coverage please run `test-all.sh` which will run `gofmt`, unit tests, race detector and then all of the integration tests. All of those must pass to ensure a stable release.

### API Support

The core part of `scc` which is the counting engine is exposed publicly to be integrated into other Go applications. See https://github.com/pinpt/ripsrc for an example of how to do this. 

It also powers all of the code calculations displayed in https://searchcode.com/ such as https://searchcode.com/file/169350674/main.go/ making it one of the more used code counters in the world.

However as a quick start consider the following,

Note that you must pass in the number of bytes in the content in order to ensure it is counted!

```
package main

import (
	"fmt"
	"io/ioutil"

	"github.com/boyter/scc/processor"
)

type statsProcessor struct{}

func (p *statsProcessor) ProcessLine(job *processor.FileJob, currentLine int64, lineType processor.LineType) bool {
	switch lineType {
	case processor.LINE_BLANK:
		fmt.Println(currentLine, "lineType", "BLANK")
	case processor.LINE_CODE:
		fmt.Println(currentLine, "lineType", "CODE")
	case processor.LINE_COMMENT:
		fmt.Println(currentLine, "lineType", "COMMENT")
	}
	return true
}

func main() {
	bts, _ := ioutil.ReadFile("somefile.go")

	t := &statsProcessor{}
	filejob := &processor.FileJob{
		Filename: "test.go",
		Language: "Go",
		Content:  bts,
		Callback: t,
		Bytes:    int64(len(bts)),
	}

	processor.ProcessConstants() // Required to load the language information and need only be done once
	processor.CountStats(filejob)
}
```


### Adding/Modifying Languages

To add or modify a language you will need to edit the `languages.json` file in the root of the project, and then run `go generate` to build it into the application. You can then `go install` or `go build` as normal to produce the binary with your modifications.

### Issues

Its possible that you may see the counts vary between runs. This usually means one of two things. Either something is changing or locking the files under scc, or that you are hitting ulimit restrictions. To change the ulimit see the following links.

 - https://superuser.com/questions/261023/how-to-change-default-ulimit-values-in-mac-os-x-10-6#306555
 - https://unix.stackexchange.com/questions/108174/how-to-persistently-control-maximum-system-resource-consumption-on-mac/221988#221988
 - https://access.redhat.com/solutions/61334
 - https://serverfault.com/questions/356962/where-are-the-default-ulimit-values-set-linux-centos
 - https://www.tecmint.com/increase-set-open-file-limits-in-linux/

To help identify this issue run scc like so `scc -v .` and look for the message `too many open files` in the output. If it is there you can rectify it by setting your ulimit to a higher value.

### Low Memory

If you are running `scc` in a low memory environment < 512 MB of RAM you may need to set `--file-gc-count` to a lower value such as `0` to force the garbage collector to be on at all times.

A sign that this is required will be `scc` crashing with panic errors.

### Tests

scc is pretty well tested with many unit, integration and benchmarks to ensure that it is fast and complete.

### Package

Run go build for windows and linux then the following in linux, keep in mind need to update the version

```
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-3.0.0-x86_64-apple-darwin.zip scc
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" && zip -r9 scc-3.0.0-arm64-apple-darwin.zip scc
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-3.0.0-x86_64-pc-windows.zip scc.exe
GOOS=windows GOARCH=386 go build -ldflags="-s -w" && zip -r9 scc-3.0.0-i386-pc-windows.zip scc.exe
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-3.0.0-x86_64-unknown-linux.zip scc
GOOS=linux GOARCH=386 go build -ldflags="-s -w" && zip -r9 scc-3.0.0-i386-unknown-linux.zip scc
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" && zip -r9 scc-3.0.0-arm64-unknown-linux.zip scc
```

### Containers

Note if you plan to run `scc` in Alpine containers you will need to build with CGO_ENABLED=0.

See the below dockerfile as an example on how to achieve this based on this issue https://github.com/boyter/scc/issues/208

```
FROM golang as scc-get

ENV GOOS=linux \
GOARCH=amd64 \
CGO_ENABLED=0

ARG VERSION
RUN git clone --branch $VERSION --depth 1 https://github.com/boyter/scc
WORKDIR /go/scc
RUN go build -ldflags="-s -w"

FROM alpine
COPY --from=scc-get /go/scc/scc /bin/
ENTRYPOINT ["scc"]
```

### Badges (beta)

You can use `scc` to provide badges on your github/bitbucket/gitlab open repositories. For example, [![Scc Count Badge](https://sloc.xyz/github/boyter/scc/)](https://github.com/boyter/scc/)
 The format to do so is,

https://sloc.xyz/PROVIDER/USER/REPO

An example of the badge for `scc` is included below, and is used on this page.

```
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/)](https://github.com/boyter/scc/)
```

By default the badge will show the repo's lines count. You can also specify for it to show a different category, by using the `?category=` query string. 

Valid values include `code, blanks, lines, comments, cocomo` and examples of the appearance are included below.

[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=code)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=blanks)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=lines)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=comments)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=cocomo)](https://github.com/boyter/scc/)


For `cocomo` you can also set the `avg-wage` value similar to `scc` itself. For example,

https://sloc.xyz/github/boyter/scc/?category=cocomo&avg-wage=1
https://sloc.xyz/github/boyter/scc/?category=cocomo&avg-wage=100000 

Note that the avg-wage value must be a positive integer otherwise it will revert back to the default value of 56286.

*NB* it may not work for VERY large repositories (has been tested on Apache hadoop/spark without issue).

### Languages

List of supported languages. The master version of `scc` supports 239 languages at last count. Note that this is always assumed that you built from master, and it might trail behind what is actually supported. To see what your version of `scc` supports run `scc --languages`

[Click here to view all languages supported by master](LANGUAGES.md)
