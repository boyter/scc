Sloc Cloc and Code (scc)
------------------------

A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform COCOMO calculation like sloccount and to estimate code complexity similar to cyclomatic complexity calculators. In short one tool to rule them all.

Also it has a very short name which is easy to type `scc`.

[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)
[![Go Report Card](https://goreportcard.com/badge/github.com/boyter/scc)](https://goreportcard.com/report/github.com/boyter/scc)
[![Coverage Status](https://coveralls.io/repos/github/boyter/scc/badge.svg?branch=master)](https://coveralls.io/github/boyter/scc?branch=master)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

Dual-licensed under MIT or the [UNLICENSE](http://unlicense.org).

Read all about how it came to be along with performance benchmarks,

 - https://boyter.org/posts/sloc-cloc-code/
 - https://boyter.org/posts/why-count-lines-of-code/
 - https://boyter.org/posts/sloc-cloc-code-revisited/
 - https://boyter.org/posts/sloc-cloc-code-performance/
 - https://boyter.org/posts/sloc-cloc-code-performance-update/

A new review of `scc`

 - https://nickmchardy.com/2018/10/counting-lines-of-code-in-koi-cms.html

For performance see the [Performance](https://github.com/boyter/scc#performance) section

Other similar projects,

 - https://github.com/Aaronepower/tokei
 - https://github.com/AlDanial/cloc
 - https://www.dwheeler.com/sloccount/
 - https://github.com/cgag/loc
 - https://github.com/hhatto/gocloc/
 - https://github.com/vmchale/polyglot

Interesting reading about other code counting projects tokei, loc and polyglot

 - https://www.reddit.com/r/rust/comments/59bm3t/a_fast_cloc_replacement_in_rust/
 - https://www.reddit.com/r/rust/comments/82k9iy/loc_count_lines_of_code_quickly/
 - http://blog.vmchale.com/article/polyglot-comparisons

Further reading about processing files on the disk performance

 - https://blog.burntsushi.net/ripgrep/

### Install

If you are comfortable using Go and have >= 1.10 installed the usual `go get -u github.com/boyter/scc/` will install for you.

If you are looking for binaries see [Releases](https://github.com/boyter/scc/releases) for Windows, GNU/Linux and macOS for both i386 and x86_64 bit machines.

If you would like to assist with getting `scc` added into snap/apt/homebrew/chocolatey/etc... please submit a PR or at least raise an issue with instructions.

### Pitch

Why use `scc`?

 - It is very fast and gets faster the more CPU you throw at it
 - Accurate
 - Works very well across multiple platforms without slowdown (Windows, Linux, macOS)
 - Large language support
 - Can ignore duplicate files
 - Has complexity estimations
 - You need to tell the difference between Coq and Verilog in the same directory

Why not use `scc`?

 - You don't like Go for some reason

### Usage

Command line usage of `scc` is designed to be as simple as possible.
Full details can be found in `scc --help`.

```
$ scc --help
Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation.
Ben Boyter <ben@boyter.org> + Contributors

Usage:
  scc [flags]

Flags:
      --avg-wage int          average wage value used for basic COCOMO calculation (default 56286)
      --binary                disable binary file detection
      --by-file               display output for every file
      --cocomo                remove COCOMO calculation output
      --debug                 enable debug output
      --exclude-dir strings   directories to exclude (default [.git,.hg,.svn])
      --file-gc-count int     number of files to parse before turning the GC on (default 10000)
  -f, --format string         set output format [tabular, wide, json, csv] (default "tabular")
  -h, --help                  help for scc
  -i, --include-ext strings   limit to file extensions [comma separated list: e.g. go,java,js]
  -l, --languages             print supported languages and extensions
  -c, --no-complexity         skip calculation of code complexity
  -d, --no-duplicates         remove duplicate files from stats and output
  -M, --not-match string      ignore files and directories matching regular expression
  -o, --output string         output filename (default stdout)
  -s, --sort string           column to sort by [files, name, lines, blanks, code, comments, complexity] (default "files")
  -t, --trace                 enable trace output. Not recommended when processing multiple files
  -v, --verbose               verbose output
      --version               version for scc
  -w, --wide                  wider output with additional statistics (implies --complexity)
```

Output should look something like the below for the redis project

```
$ scc .
───────────────────────────────────────────────────────────────────────────────
Language                 Files     Lines     Code  Comments   Blanks Complexity
───────────────────────────────────────────────────────────────────────────────
C                          249    143701   103269     24163    16269      25906
C Header                   199     27537    18543      5768     3226       1557
TCL                         98     16865    14098       954     1813       1404
Shell                       36      1106      775       215      116         89
Lua                         20       525      387        70       68         65
Autoconf                    18     10821     8469      1326     1026        951
gitignore                   11       151      135         0       16          0
Makefile                     9      1031      722       100      209         50
Markdown                     8      1886     1363         0      523          0
Ruby                         8       722      580        69       73        107
HTML                         5      9658     8791        12      855          0
C++                          5       311      244        16       51         31
YAML                         4       273      254         0       19          0
License                      3        66       55         0       11          0
CSS                          2       107       91         0       16          0
Python                       2       219      162        18       39         68
C++ Header                   1         9        5         3        1          0
Smarty Template              1        44       43         0        1          5
m4                           1       562      393        53      116          0
Plain Text                   1        23       16         0        7          0
Batch                        1        28       26         0        2          3
───────────────────────────────────────────────────────────────────────────────
Total                      682    215645   158421     32767    24457      30236
───────────────────────────────────────────────────────────────────────────────
Estimated Cost to Develop $5,513,136
Estimated Schedule Effort 29.350958 months
Estimated People Required 22.250054
───────────────────────────────────────────────────────────────────────────────
```

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

### Performance

Generally `scc` will be very close to the runtime of `tokei` or faster than any other code counter out there. It is designed to scale to as many CPU's cores as you can provide.

However if you want greater performance and you have RAM to spare you can disable the garbage collector like the following on linux `GOGC=-1 scc .` which should speed things up considerably.

Benchmarks are run on fresh 32 CPU Optimised Digital Ocean Virtual Machine 2019/03/04 all done using [hyperfine](https://github.com/sharkdp/hyperfine) with 3 warm-up runs and 10 timed runs.

```
scc v2.2.0 (compiled with Go 1.12)
tokei v9.0.0 (compiled with Rust 1.33)
loc v0.5.0 (compiled with Rust 1.33)
polyglot v0.5.19 (downloaded from github)
```


#### Redis https://github.com/antirez/redis/

| Program | Runtime |
|---|---|
| scc | 24.0 ms ±   2.7 ms |
| scc (no complexity) | 18.9 ms ±   2.2 ms |
| tokei | 26.6 ms ±   3.3 ms |
| loc | 80.1 ms ±  54.7 ms |
| polyglot | 15.0 ms ±   1.1 ms |

#### CPython https://github.com/python/cpython

| Program | Runtime |
|---|---|
| scc | 64.3 ms ±   6.3 ms |
| scc (no complexity) | 53.8 ms ±   6.5 ms |
| tokei | 74.9 ms ±  11.6 ms |
| loc | 155.1 ms ±  58.9 ms |
| polyglot | 83.9 ms ±   9.4 ms |

#### Linux Kernel https://github.com/torvalds/linux

| Program | Runtime |
|---|---|
| scc | 537.3 ms ±  33.1 ms |
| scc (no complexity) | 438.9 ms ±  30.3 ms |
| tokei | 525.9 ms ±  32.7 ms |
| loc | 1.543 s ±  0.059 s |
| polyglot | 1.022 s ±  0.056 s |

If you enable duplicate detection expect performance to fall by about 50%

### API Support

The core part of `scc` which is the counting engine is exposed publicly to be integrated into other Go applications. See https://github.com/pinpt/ripsrc for an example of how to do this.

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
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-2.3.0-x86_64-apple-darwin.zip scc
GOOS=darwin GOARCH=386 go build -ldflags="-s -w" && zip -r9 scc-2.3.0-i386-apple-darwin.zip scc
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-2.3.0-x86_64-pc-windows.zip scc.exe
GOOS=windows GOARCH=386 go build -ldflags="-s -w" && zip -r9 scc-2.3.0-i386-pc-windows.zip scc.exe
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-2.3.0-x86_64-unknown-linux.zip scc
GOOS=linux GOARCH=386 go build -ldflags="-s -w" && zip -r9 scc-2.3.0-i386-unknown-linux.zip scc
```

### Languages

List of supported languages. The master version of `scc` supports 224 languages at last count. Note that this is always assumed that you built from master, and it might trail behind what is actually supported. To see what your version of `scc` supports run `scc --languages`

[Click here to view all languages supported by master](LANGUAGES.md)