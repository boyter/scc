[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)


A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform license analysis and bring in COCOMO calculation like sloccount and to estimate code complexity. In short one tool to rule them all.

https://github.com/Aaronepower/tokei
https://github.com/AlDanial/cloc
https://www.dwheeler.com/sloccount/

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
# bboyter @ SurfaceBook2 in ~/Projects/linux on git:master o [11:13:34]
$ scc .
----------------------------------------------------------------------------
Language                  Files  Lines     Code  Comment  Blank    Byte
----------------------------------------------------------------------------
LaTeX                     1      1015      0     0        103      50901
Configuration             15     2380      0     0        244      81669
Device Tree Source        1474   286088    0     0        36037    6727525
Objective C++             1      244       0     0        51       10878
git-ignore                211    944       0     0        24       12062
Bourne Shell              204    20120     0     0        2778     488723
C                         26191  17908117  0     0        2538497  485987093
CShell                    204    20120     0     0        2778     488723
CSV                       4      171       0     0        1        10050
m4                        1      111       0     0        15       3325
Text                      4127   446028    0     0        84971    15988404
HTML                      5      6161      0     0        668      245751
Python                    80     18918     0     0        2252     614271
yacc                      9      5667      0     0        685      119490
Gherkin Specification     1      234       0     0        28       7710
Scalable Vector Graphics  57     39425     0     0        92       1916717
ReStructured Text         850    168686    0     0        38551    5492158
NAnt scripts              2      779       0     0        160      24290
XSLT                      5      100       0     0        13       3492
Teamcenter def            1      8         0     0        0        147
sed                       1      12        0     0        2        379
make                      3      160       0     0        26       5340
JSON                      214    108494    0     0        0        4507604
awk                       9      1896      0     0        177      45929
Perl                      42     29706     0     0        4239     782182
VimScript                 1      42        0     0        3        1355
ObjectiveC                1      244       0     0        51       10878
PHP                       5      440       0     0        71       10903
Bourne Again Shell        204    20120     0     0        2778     488723
Markdown                  1      1297      0     0        220      65732
Assembly                  2      1417      0     0        186      68020
lex                       8      2542      0     0        298      58626
CSS                       1      89        0     0        18       2258
vimscript                 1      42        0     0        3        1355
C/C++Header               2      125       0     0        19       3859
C++                       7      2202      0     0        284      53118
----------------------------------------------------------------------------
Total                     62870  26272203  0     0        3365900  974903395
----------------------------------------------------------------------------
```