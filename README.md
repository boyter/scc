[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)


A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform COCOMO calculation like sloccount and to estimate code complexity similar to cyclomatic complexity calculators. In short one tool to rule them all.

https://github.com/Aaronepower/tokei
https://github.com/AlDanial/cloc
https://www.dwheeler.com/sloccount/
https://github.com/cgag/loc

http://www.locmetrics.com/alternatives.html

Interesting read https://www.reddit.com/r/rust/comments/59bm3t/a_fast_cloc_replacement_in_rust/
https://www.reddit.com/r/rust/comments/82k9iy/loc_count_lines_of_code_quickly/

Quick benchmark working on the redis souce code.

```
$ hyperfine -w 3 -m 10 'scc .'
Benchmark #1: scc .

  Time (mean ± σ):      68.2 ms ±  22.9 ms    [User: 48.9 ms, System: 169.3 ms]

  Range (min … max):    47.3 ms … 135.2 ms
```

And when limiting to a single CPU

```
$ hyperfine 'taskset 0x01 scc .'
Benchmark #1: taskset 0x01 scc .

  Time (mean ± σ):     231.5 ms ±  12.1 ms    [User: 73.3 ms, System: 119.8 ms]

  Range (min … max):   216.6 ms … 254.1 ms

```


Output should look something like the below for redis

```
$ scc .
-------------------------------------------------------------------------------
Language                 Files     Lines     Code  Comments   Blanks Complexity
-------------------------------------------------------------------------------
C                          215    114595    82383     18209    14003      21771
C Header                   144     20508    10029      7775     2704        833
TCL                         93     15702    12726      1157     1819       1376
Lua                         20       525      456         0       69         72
Shell                       18       796      459       237      100         81
Autoconf                    18     10500     7498      1966     1036        915
Makefile                     9      1021      716       100      205         45
Ruby                         8      2400     1783       377      240        355
Markdown                     6      1308      915        45      348        108
HTML                         6     11452     8264         6     3182        549
CSS                          2       107       91         0       16          0
YAML                         2        75       60         4       11          4
C++                          1        46       26         4       16          5
Plain Text                   1       499      499         0        0          0
Batch                        1        28       26         0        2          3
C++ Header                   1         9        2         6        1          0
-------------------------------------------------------------------------------
Total                      545    179571   125933     29886    23752      26117
-------------------------------------------------------------------------------
Estimated Cost to Develop $4,332,532
Estimated Schedule Effort 26.782597 months
Estimated People Required 19.162132
-------------------------------------------------------------------------------
```

To benchmark,

```
go test -bench .
```
