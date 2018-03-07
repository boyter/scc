[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)


A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform license analysis and bring in COCOMO calculation like sloccount and to estimate code complexity. In short one tool to rule them all.

https://github.com/Aaronepower/tokei
https://github.com/AlDanial/cloc
https://www.dwheeler.com/sloccount/

http://www.locmetrics.com/alternatives.html

Running against the linux kernel compared to tokei

```
$ hyperfine 'scc .' && hyperfine 'tokei .'
Benchmark #1: scc .

  Time (mean ± σ):      4.129 s ±  0.071 s    [User: 3.408 s, System: 12.125 s]

  Range (min … max):    4.041 s …  4.290 s

Benchmark #1: tokei .

  Time (mean ± σ):     10.263 s ±  2.077 s    [User: 31.570 s, System: 22.635 s]

  Range (min … max):    8.418 s … 14.675 s

```

```
# bboyter @ SurfaceBook2 in ~/Projects/linux on git:master o [10:11:43]
$ scc .
-------------------------------------------------------------------
Language           Files  Lines     Code  Comment  Blank  Byte
-------------------------------------------------------------------
TeX                1      1015      0     0        0      50901
CSS                1      89        0     0        0      2258
C Header           20516  5163057   0     0        0      224102981
Autoconf           7      182       0     0        0      10943
Unreal Script      5      694       0     0        0      17261
Plain Text         4127   446032    0     0        0      15988404
Assembly           1477   418489    0     0        0      10719553
HEX                2      87        0     0        0      4144
ReStructuredText   850    168686    0     0        0      5492158
Vim Script         1      42        0     0        0      1355
C++                7      2202      0     0        0      53118
Perl               43     29724     0     0        0      783087
Markdown           1      1297      0     0        0      65732
LD Script          20     607       0     0        0      11906
C++ Header         2      125       0     0        0      3859
Module-Definition  1      8         0     0        0      147
Makefile           2469   58205     0     0        0      1874552
Python             80     18918     0     0        0      614271
Shell              204    20122     0     0        0      488723
Device Tree        2587   644437    0     0        0      15749617
Objective C++      1      244       0     0        0      10878
Happy              9      5667      0     0        0      119490
JSON               214    108649    0     0        0      4507604
SVG                57     39430     0     0        0      1916717
HTML               5      6161      0     0        0      245751
C                  26191  17908117  0     0        0      485987093
-------------------------------------------------------------------
Total              58878  25042286  0     0        0      768822503
-------------------------------------------------------------------
```

To benchmark,

```
go test -bench .
```

Quick comparsion using ripgrep as the 'king' of directory scanning performance against the linux source code

```
# bboyter @ SurfaceBook2 in ~/Projects/linux on git:master o [10:05:05]
$ hyperfine 'rg a .'
Benchmark #1: rg a .

  Time (mean ± σ):      3.537 s ±  0.458 s    [User: 5.651 s, System: 18.141 s]

  Range (min … max):    3.045 s …  4.480 s

# bboyter @ SurfaceBook2 in ~/Projects/linux on git:master o [10:06:41]
$ hyperfine 'scc .'
Benchmark #1: scc .

  Time (mean ± σ):      4.257 s ±  0.149 s    [User: 3.343 s, System: 12.779 s]

  Range (min … max):    4.116 s …  4.576 s

```