[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)


A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform license analysis and bring in COCOMO calculation like sloccount and to estimate code complexity. In short one tool to rule them all.

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


Output should look something like the below

```
$ scc .
------------------------------------------------------------------
Language    Files  Lines  Code  Comment  Blank  Complexity  Byte
------------------------------------------------------------------
Go          51     6901   4750  1226     925    1170        176376
YAML        7      119    95    6        18     3           1942
JSON        3      2584   1225  1359     0      0           47121
Python      7      77     42    10       25     14          1854
Java        1      1      1     0        0      1           3
Markdown    9      2393   1743  165      485    135         71258
TOML        1      38     9     25       4      0           829
Plain Text  1      20     17    0        3      0           1089
------------------------------------------------------------------
Total       80     12133  7882  2791     1460   1323        300472
------------------------------------------------------------------
```

To benchmark,

```
go test -bench .
```
