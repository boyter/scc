[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)


A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform license analysis and bring in COCOMO calculation like sloccount and to estimate code complexity. In short one tool to rule them all.

https://github.com/Aaronepower/tokei
https://github.com/AlDanial/cloc
https://www.dwheeler.com/sloccount/

```
-------------------------------------------------------------------------------
 Language            Files        Lines         Code     Comments       Blanks
-------------------------------------------------------------------------------
 Assembly             1482       420953       366611         3805        50537
 Autoconf                7          182          136           17           29
 C                   26187     17906413     13036234      2299473      2570706
 C Header            20520      5164917      3696579       956359       511979
 C++                     7         2202         1838           77          287
 C++ Header              2          125           59           47           19
 CSS                     1           89           44           27           18
 Device Tree          2592       645105       507023        61775        76307
 Happy                   9         5667         5667            0            0
 HEX                     1           86           86            0            0
 HTML                    5         6161         5492            0          669
 JSON                  214       108649       108649            0            0
 LD Script              20          607          477           72           58
 Makefile             2466        58107        38732        10300         9075
 Markdown                1         1297         1297            0            0
 Module-Definition       1            8            8            0            0
 Objective C++           1          244          189            0           55
 Perl                   46        30627        23018         3190         4419
 Python                 80        18918        14394         2105         2419
 ReStructuredText      850       168686       168686            0            0
 Shell                 239        23243        15817         4210         3216
 SVG                    57        39430        37967         1371           92
 TeX                     1         1015          904            3          108
 Plain Text           4127       446032       446032            0            0
 Unreal Script           5          694          422          169          103
 Vim Script              1           42           27           12            3
-------------------------------------------------------------------------------
 Total               58922     25049499     18476388      3343012      3230099
-------------------------------------------------------------------------------
```