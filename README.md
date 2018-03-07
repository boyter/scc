[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)


A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform license analysis and bring in COCOMO calculation like sloccount and to estimate code complexity. In short one tool to rule them all.

https://github.com/Aaronepower/tokei
https://github.com/AlDanial/cloc
https://www.dwheeler.com/sloccount/

http://www.locmetrics.com/alternatives.html

Running against the linux kernel compared to tokei

```
# bboyter @ SurfaceBook2 in ~/Projects/linux on git:master o [21:29:09]
$ hyperfine 'scc .' && hyperfine 'tokei .'
Benchmark #1: scc .

  Time (mean ± σ):      5.094 s ±  0.451 s    [User: 6.014 s, System: 15.770 s]

  Range (min … max):    4.350 s …  5.925 s

Benchmark #1: tokei .

  Time (mean ± σ):     10.333 s ±  1.101 s    [User: 36.295 s, System: 24.282 s]

  Range (min … max):    8.214 s … 11.619 s

```

```
# bboyter @ SurfaceBook2 in ~/Projects/linux on git:master o [21:32:09]
$ scc .
-------------------------------------------------------------------------
Language           Files  Lines     Code      Comment  Blank    Byte
-------------------------------------------------------------------------
Perl               43     29724     25446     0        4278     783087
CSS                1      89        71        0        18       2258
LD Script          20     607       549       0        58       11906
Device Tree        2587   644437    568201    0        76236    15749617
Autoconf           7      182       153       0        29       10943
C++                7      2202      1915      0        287      53118
HEX                2      87        87        0        0        4144
SVG                57     39430     39338     0        92       1916717
JSON               214    108649    108649    0        0        4507604
ReStructuredText   850    168686    126677    0        42009    5492158
Python             80     18918     16499     0        2419     614271
Vim Script         1      42        39        0        3        1355
C Header           20516  5163057   4651285   0        511772   224102981
Module-Definition  1      8         8         0        0        147
Plain Text         4127   446032    354657    0        91375    15988404
Makefile           2469   58205     49119     0        9086     1874552
Objective C++      1      244       189       0        55       10878
TeX                1      1015      907       0        108      50901
C                  26191  17908117  15337258  0        2570859  485987093
Shell              204    20122     17312     0        2810     488723
Markdown           1      1297      1077      0        220      65732
C++ Header         2      125       106       0        19       3859
Happy              9      5667      4975      0        692      119490
Unreal Script      5      694       591       0        103      17261
Assembly           1477   418489    368220    0        50269    10719553
HTML               5      6161      5492      0        669      245751
-------------------------------------------------------------------------
Total              58878  25042286  21678820  0        3363466  768822503
-------------------------------------------------------------------------
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