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

If you are comfortable using Go and have >= 1.10 installed:

`$ go get -u github.com/boyter/scc/`

#### Snap

A [snap install](https://snapcraft.io/scc) exists thanks to [Ricardo](https://feliciano.tech/).

`$ sudo snap install scc`

#### Homebrew

Of if you have [homebrew](https://brew.sh/) installed

`$ brew install scc`

#### Manual

Binaries for Windows, GNU/Linux and macOS for both i386 and x86_64 machines are available from the [releases](https://github.com/boyter/scc/releases) page.

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

A talk given at the first GopherCon AU about `scc` (press S to see speaker notes)

 - https://boyter.org/static/gophercon-syd-presentation/
 - https://www.youtube.com/watch?v=jd-sjoy3GZo

For performance see the [Performance](https://github.com/boyter/scc#performance) section

Other similar projects,

 - [cloc](https://github.com/AlDanial/cloc) the original sloc counter
 - [gocloc](https://github.com/hhatto/gocloc) a sloc counter in Go inspired by tokei
 - [loc](https://github.com/cgag/loc) rust implementation similar to tokei but often faster
 - [loccount](https://gitlab.com/esr/loccount) Go implementation written and maintained by ESR
 - [ployglot](https://github.com/vmchale/polyglot) ATS sloc counter
 - [sloccount](https://www.dwheeler.com/sloccount/) written as a faster cloc
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

### Usage

Command line usage of `scc` is designed to be as simple as possible.
Full details can be found in `scc --help` or `scc -h`.

```
Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation.
Version 2.10.0
Ben Boyter <ben@boyter.org> + Contributors

Usage:
  scc [flags]

Flags:
      --avg-wage int              average wage value used for basic COCOMO calculation (default 56286)
      --binary                    disable binary file detection
      --by-file                   display output for every file
      --ci                        enable CI output settings where stdout is ASCII
      --debug                     enable debug output
      --exclude-dir strings       directories to exclude (default [.git,.hg,.svn])
      --file-gc-count int         number of files to parse before turning the GC on (default 10000)
  -f, --format string             set output format [tabular, wide, json, csv, cloc-yaml] (default "tabular")
  -h, --help                      help for scc
  -i, --include-ext strings       limit to file extensions [comma separated list: e.g. go,java,js]
  -l, --languages                 print supported languages and extensions
      --large-byte-count int      number of bytes a file can contain before being removed from output (default 1000000)
      --large-line-count int      number of lines a file can contain before being removed from output (default 40000)
  -z, --min-gen                   identify minified or generated files
      --min-gen-line-length int   number of bytes per average line for file to be considered minified or generated (default 255)
      --no-cocomo                 remove COCOMO calculation output
  -c, --no-complexity             skip calculation of code complexity
  -d, --no-duplicates             remove duplicate files from stats and output
      --no-gitignore              disables .gitignore file logic
      --no-ignore                 disables .ignore file logic
      --no-large                  ignore files over certain byte and line size set by max-line-count and max-byte-count
      --no-min-gen                ignore minified or generated files in output (implies --min-gen)
  -M, --not-match stringArray     ignore files and directories matching regular expression
  -o, --output string             output filename (default stdout)
  -s, --sort string               column to sort by [files, name, lines, blanks, code, comments, complexity] (default "files")
  -t, --trace                     enable trace output. Not recommended when processing multiple files
  -v, --verbose                   verbose output
      --version                   version for scc
  -w, --wide                      wider output with additional statistics (implies --complexity)
  ```

Output should look something like the below for the redis project

```
$ scc .
───────────────────────────────────────────────────────────────────────────────
Language                 Files     Lines   Blanks  Comments     Code Complexity
───────────────────────────────────────────────────────────────────────────────
C                          258    153080    17005     26121   109954      27671
C Header                   200     28794     3252      5877    19665       1557
TCL                        101     17802     1879       981    14942       1439
Shell                       36      1109      133       252      724        118
Lua                         20       525       68        70      387         65
Autoconf                    18     10821     1026      1326     8469        951
Makefile                    10      1082      220       103      759         51
Ruby                        10       778       78        71      629        115
Markdown                     9      1935      527         0     1408          0
gitignore                    9       120       16         0      104          0
HTML                         5      9658     2928        12     6718          0
C++                          4       286       48        14      224         31
License                      4       100       20         0       80          0
YAML                         4       266       20         3      243          0
CSS                          2       107       16         0       91          0
Python                       2       219       39        18      162         68
Batch                        1        28        2         0       26          3
C++ Header                   1         9        1         3        5          0
Extensible Styleshe…         1        10        0         0       10          0
Plain Text                   1        23        7         0       16          0
Smarty Template              1        44        1         0       43          5
m4                           1       562      116        53      393          0
───────────────────────────────────────────────────────────────────────────────
Total                      698    227358    27402     34904   165052      32074
───────────────────────────────────────────────────────────────────────────────
Estimated Cost to Develop $5,755,686
Estimated Schedule Effort 29.835114 months
Estimated People Required 22.851995
───────────────────────────────────────────────────────────────────────────────
```

Note that you don't have to specify the directory you want to run against. Running `scc` will assume you want to run against the current directory.

You can also run against multiple files or directories `scc directory1 directory2 file1 file2` with the results aggregated in the output.

### Interesting Use Cases

Used inside Intel Nemu Hypervisor to track code changes between revisions https://github.com/intel/nemu/blob/topic/virt-x86/tools/cloc-change.sh#L9
Appears to also be used inside both http://codescoop.com/ and https://pinpoint.com/

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
```

Minified files are indicated with the text `(min)` after the language name.

You can control the average line byte size using `--min-gen-line-length` such as `scc -z --min-gen-line-length 1`. Please note you need `-z` as modifying this value does not imply minified detection.

You can exclude minified files from the count totally using the flag `--no-min-gen`. Files which match the minified check will be excluded from the output.

### Performance

Generally `scc` will the fastest code counter compared to any I am aware of and have compared against. The below comparisons are taken from the fastest alternative counters. See `Other similar projects` above to see all of the other code counters compared against. It is designed to scale to as many CPU's cores as you can provide.

However if you want greater performance and you have RAM to spare you can disable the garbage collector like the following on linux `GOGC=-1 scc .` which should speed things up considerably. For some repositories turning off the code complexity calculation via `-c` can reduce runtime as well.

Benchmarks are run on fresh 32 Core CPU Optimised Digital Ocean Virtual Machine 2019/10/10 all done using [hyperfine](https://github.com/sharkdp/hyperfine) with 3 warm-up runs and 10 timed runs.

```
scc v2.8.0
tokei v10.0.1
loc v0.5.0
polyglot v0.5.25
```


#### Redis https://github.com/antirez/redis/

| Program | Runtime |
|---|---|
| scc | 60.0 ms ±   5.8 ms |
| scc (no complexity) | 49.1 ms ±   4.7 ms |
| tokei | 47.1 ms ±   3.9 ms |
| loc | 66.3 ms ±  25.4 ms |
| polyglot | 41.8 ms ±   1.3 ms |

#### CPython https://github.com/python/cpython

| Program | Runtime |
|---|---|
| scc | 112.9 ms ±  19.8 ms |
| scc (no complexity) | 91.7 ms ±  19.2 ms |
| tokei | 103.6 ms ±  10.3 ms |
| loc | 177.0 ms ±  44.2 ms |
| polyglot | 175.8 ms ±   8.0 ms |

#### Linux Kernel https://github.com/torvalds/linux

| Program | Runtime |
|---|---|
| scc | 682.2 ms ±  29.6 ms |
| scc (no complexity) | 538.1 ms ±  26.3 ms |
| tokei | 782.8 ms ±  30.4 ms |
| loc | 1.957 s ±  0.031 s |
| polyglot | 1.736 s ±  0.063 s |

If you enable duplicate detection expect performance to fall by about 20% in `scc`.

Performance is tracked over each release and presented below.

<img alt="scc" src=https://github.com/boyter/scc/raw/master/performance-over-time.png>

### CI/CD Support

Some CI/CD systems which will remain nameless do not work very well with the box-lines used by `scc`. To support those systems better there is an option `--ci` which will change the default output to ASCII only.

```
$ scc --ci main.go
-------------------------------------------------------------------------------
Language                 Files     Lines   Blanks  Comments     Code Complexity
-------------------------------------------------------------------------------
Go                           1       171        6         4      161          2
-------------------------------------------------------------------------------
Total                        1       171        6         4      161          2
-------------------------------------------------------------------------------
Estimated Cost to Develop $3,969
Estimated Schedule Effort 1.876811 months
Estimated People Required 0.250551
-------------------------------------------------------------------------------
```

### Development

If you want to hack away feel free! PR's are accepted. Some things to keep in mind. If you want to change a language definition you need to update `languages.json` and then run `go generate` which will convert it into the `processor/constants.go` file.

For all other changes ensure you run all tests before submitting. You can do so using `go test ./...`. However for maximum coverage please run `test-all.sh` which will run `gofmt`, unit tests, race detector and then all of the integration tests. All of those must pass to ensure a stable release.

### API Support

The core part of `scc` which is the counting engine is exposed publicly to be integrated into other Go applications. See https://github.com/pinpt/ripsrc for an example of how to do this. However as a quick start consider the following,

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

If you are running `scc` in a low memory environment < 512 MB of RAM you may need to set `--filegccount` or `--fgc` to a lower value such as `0` to force the garbage collector to be on at all times.

A sign that this is required will be `scc` crashing with panic errors.

### Tests

scc is pretty well tested with many unit, integration and benchmarks to ensure that it is fast and complete.

### Package

Run go build for windows and linux then the following in linux, keep in mind need to update the version

```
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-2.10.1-x86_64-apple-darwin.zip scc
GOOS=darwin GOARCH=386 go build -ldflags="-s -w" && zip -r9 scc-2.10.1-i386-apple-darwin.zip scc
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-2.10.1-x86_64-pc-windows.zip scc.exe
GOOS=windows GOARCH=386 go build -ldflags="-s -w" && zip -r9 scc-2.10.1-i386-pc-windows.zip scc.exe
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-2.10.1-x86_64-unknown-linux.zip scc
GOOS=linux GOARCH=386 go build -ldflags="-s -w" && zip -r9 scc-2.10.1-i386-unknown-linux.zip scc
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
