Sloc Cloc and Code (scc)
------------------------

A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform COCOMO calculation like sloccount and to estimate code complexity similar to cyclomatic complexity calculators. In short one tool to rule them all and the one I wish I had before I wrote it.

Also it has a much shorter name than tokei, and shorter than cloc.

It is faster than loc/sloccount/cloc/gocloc with more accuracy. It is however marginally slower than tokei (on linux) but with duplicate detection and complexity count. If you disable garbage collection `GOGC=-1 scc .` it is indeed faster than everything including tokei by a huge margin.

[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)

Dual-licensed under MIT or the [UNLICENSE](http://unlicense.org).

Read all about how it came to be along with performance benchmarks https://boyter.org/posts/sloc-cloc-code/ or why use a code counting tool https://boyter.org/posts/why-count-lines-of-code/

Other similar projects,

 - https://github.com/Aaronepower/tokei
 - https://github.com/AlDanial/cloc
 - https://www.dwheeler.com/sloccount/
 - https://github.com/cgag/loc
 - https://github.com/hhatto/gocloc/
 - http://www.locmetrics.com/alternatives.html

Interesting reading about about code counting about tokei and loc

 - https://www.reddit.com/r/rust/comments/59bm3t/a_fast_cloc_replacement_in_rust/
 - https://www.reddit.com/r/rust/comments/82k9iy/loc_count_lines_of_code_quickly/

Further reading about processing files on the disk performance

 - https://blog.burntsushi.net/ripgrep/

### Usage

Command line usage of `scc` is designed to be as simple as possible.
Full details can be found in `scc --help`.

```
$ scc --help
NAME:
   scc - Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation.

USAGE:
   scc DIRECTORY

VERSION:
   1.3.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --languages                         Print out supported languages and their extensions
   --format value, -f value            Set output format [possible values: tabular, wide, json, csv] (default: "tabular")
   --output FILE, -o FILE              Set output file if not set will print to stdout FILE
   --pathblacklist value, --pbl value  Which directories should be ignored as comma seperated list (default: ".git,.hg,.svn")
   --sort value, -s value              Sort languages / files based on column [possible values: files, name, lines, blanks, code, comments, complexity] (default: "files")
   --whitelist value, --wl value       Restrict file extensions to just those provided as a comma seperated list E.G. go,java,js
   --files                             Set to specify you want to see the output for every file
   --verbose, -v                       Set to enable verbose output
   --duplicates, -d                    Set to check for and remove duplicate files from stats and output
   --complexity, -c                    Set to skip complexity calculations note will be overridden if wide is set
   --wide, -w                          Set to check produce more output such as complexity and code vs complexity ranking. Same as setting format to wide
   --averagewage value, --aw value     Set as integer to set the average wage used for basic COCOMO calculation (default: 56286)
   --cocomo, --co                      Set to check remove cocomo calculation output
   --debug                             Set to enable debug output
   --trace                             Set to enable trace output, not reccomended for multiple files
   --help, -h                          show help
   --version, --ver                    Print the version
```

Output should look something like the below for the redis project

```
$ scc .
-------------------------------------------------------------------------------
Language                 Files     Lines     Code  Comments   Blanks Complexity
-------------------------------------------------------------------------------
C                          215    114488    85341     15175    13972      21921
C Header                   144     20042    13308      4091     2643       1073
TCL                         93     15702    12933       922     1847       1482
Lua                         20       524      384        71       69         66
Autoconf                    18     10713     8164      1476     1073        986
Shell                       18       810      513       196      101        102
Makefile                     9      1021      716       100      205         50
Ruby                         8      2416     1798       376      242        365
HTML                         6     11472     8288         5     3179        548
Markdown                     6      1312      964         0      348          0
CSS                          2       107       91         0       16          0
YAML                         2        75       60         4       11          4
C++ Header                   1         9        5         3        1          0
C++                          1        46       30         0       16          5
Batch                        1        28       26         0        2          3
Plain Text                   1       499      499         0        0          0
-------------------------------------------------------------------------------
Total                      545    179264   133120     22419    23725      26605
-------------------------------------------------------------------------------
Estimated Cost to Develop $4,592,517
Estimated Schedule Effort 27.382310 months
Estimated People Required 19.867141
-------------------------------------------------------------------------------
```

### Performance

Generally `scc` will be very close to the runtime of `tokei` or faster than any other code counter out there. However if you want greater performance and you have RAM to spare you can disable the garbage collector like the following on linux `GOGC=-1 scc .` which should speed things up considerably. See the below for example runtimes on a 16 CPU Linux machine running against the linux kernel source.

```
scc                        1.489 s ± 0.055 s
scc (no complexity)        1.713 s ± 0.157 s
scc (duplicates detection) 2.122 s ± 0.054 s
scc (no GC no complexity)  0.744 s ± 0.167 s
```

### Adding/Modifying Languages

To add or modify a language you will need to eddit the `languages.json` file in the root of the project, and then run `go generate` to build it into the application. You can then `go install` or `go build` as normal to produce the binary with your modifications.

### Issues

Its possible that you may see the counts vary between runs. This usually means one of two things. Either something is changing or locking the files under scc, or that you are hitting ulimit restrictions. To change the ulimit see the following links.

 - https://superuser.com/questions/261023/how-to-change-default-ulimit-values-in-mac-os-x-10-6#306555
 - https://unix.stackexchange.com/questions/108174/how-to-persistently-control-maximum-system-resource-consumption-on-mac/221988#221988
 - https://access.redhat.com/solutions/61334
 - https://serverfault.com/questions/356962/where-are-the-default-ulimit-values-set-linux-centos
 - https://www.tecmint.com/increase-set-open-file-limits-in-linux/

To help identify this issue run scc like so `scc -v .` and look for the message `too many open files` in the output. If it is there you can rectify it by setting your ulimit to a higher value.

### Tests

scc is pretty well tested with many unit, integration and benchmarks to ensure that it is fast and complete.

### Package

Run go build for windows and linux then the following in linux, keep in mind need to update the version

```
GOOS=darwin GOARCH=amd64 go build && zip -r9 scc-1.0.0-x86_64-apple-darwin.zip scc
GOOS=windows GOARCH=amd64 go build && zip -r9 scc-1.0.0-x86_64-pc-windows.zip scc.exe
GOOS=linux GOARCH=amd64 go build && zip -r9 scc-1.0.0-x86_64-unknown-linux.zip scc
```

