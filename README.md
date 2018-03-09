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

Quick comparsion using ripgrep as the best in class of directory scanning performance against the redis source code

```
# bboyter @ SurfaceBook2 in ~/Projects/redis on git:unstable o [9:12:34]
$ hyperfine -w 3 -m 10 'rg a .'
Benchmark #1: rg a .

  Time (mean ± σ):      86.6 ms ±  25.2 ms    [User: 47.5 ms, System: 221.2 ms]

  Range (min … max):    50.4 ms … 136.1 ms


# bboyter @ SurfaceBook2 in ~/Projects/redis on git:unstable o [9:12:53]
$ hyperfine -w 3 -m 10 'tokei .'
Benchmark #1: tokei .

  Time (mean ± σ):     147.7 ms ±  38.0 ms    [User: 180.1 ms, System: 180.0 ms]

  Range (min … max):   106.6 ms … 219.4 ms


# bboyter @ SurfaceBook2 in ~/Projects/redis on git:unstable o [9:13:08]
$ hyperfine -w 3 -m 10 'loc .'
Benchmark #1: loc .

  Time (mean ± σ):     357.5 ms ±  10.2 ms    [User: 118.4 ms, System: 184.5 ms]

  Range (min … max):   343.9 ms … 374.3 ms


# bboyter @ SurfaceBook2 in ~/Projects/redis on git:unstable o [9:13:21]
$ hyperfine -w 3 -m 10 'scc .'
Benchmark #1: scc .

  Time (mean ± σ):      68.2 ms ±  22.9 ms    [User: 48.9 ms, System: 169.3 ms]

  Range (min … max):    47.3 ms … 135.2 ms


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


Quick comparsion using ripgrep as the baseline best in class of file scanning performance against the linux source code


```
root@ubuntu-c-8-16gib-sgp1-01:~/linux# hyperfine -w 3 -m 10 'rg a .'
Benchmark #1: rg a .

  Time (mean ± σ):     601.3 ms ±   8.7 ms    [User: 3.763 s, System: 0.791 s]

  Range (min … max):   590.9 ms … 616.1 ms

root@ubuntu-c-8-16gib-sgp1-01:~/linux# hyperfine -w 3 -m 10 'tokei .'
Benchmark #1: tokei .

  Time (mean ± σ):      4.049 s ±  0.952 s    [User: 22.917 s, System: 1.107 s]

  Range (min … max):    3.222 s …  5.646 s

root@ubuntu-c-8-16gib-sgp1-01:~/linux# hyperfine -w 3 -m 10 'loc .'
Benchmark #1: loc .

  Time (mean ± σ):      3.065 s ±  0.605 s    [User: 7.413 s, System: 3.894 s]

  Range (min … max):    2.354 s …  4.179 s

root@ubuntu-c-8-16gib-sgp1-01:~/linux# hyperfine -w 3 -m 10 'scc .'
Benchmark #1: scc .

  Time (mean ± σ):      1.805 s ±  0.064 s    [User: 5.386 s, System: 0.982 s]

  Range (min … max):    1.721 s …  1.929 s
```


Trying things out limited to a single CPU

```
$ hyperfine 'taskset 0x01 scc .'
Benchmark #1: taskset 0x01 scc .

  Time (mean ± σ):      54.7 ms ±   5.1 ms    [User: 2.2 ms, System: 18.4 ms]

  Range (min … max):    46.0 ms …  66.8 ms


# bboyter @ SurfaceBook2 in ~/Go/src/github.com/boyter/scc on git:master x [11:57:27]
$ hyperfine 'taskset 0x01 tokei .'
Benchmark #1: taskset 0x01 tokei .

  Time (mean ± σ):      86.5 ms ±   9.5 ms    [User: 4.8 ms, System: 30.0 ms]

  Range (min … max):    73.0 ms … 115.3 ms


# bboyter @ SurfaceBook2 in ~/Go/src/github.com/boyter/scc on git:master x [11:57:39]
$ hyperfine 'taskset 0x01 loc .'
Benchmark #1: taskset 0x01 loc .

  Time (mean ± σ):     100.0 ms ±   9.2 ms    [User: 4.1 ms, System: 27.6 ms]

  Range (min … max):    80.5 ms … 126.1 ms

```