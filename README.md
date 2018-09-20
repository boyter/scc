Sloc Cloc and Code (scc)
------------------------

A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform COCOMO calculation like sloccount and to estimate code complexity similar to cyclomatic complexity calculators. In short one tool to rule them all and the one I wish I had before I wrote it.

Also it has a very short name which is easy to type `scc`.

[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)
[![Go Report Card](https://goreportcard.com/badge/github.com/boyter/scc)](https://goreportcard.com/report/github.com/boyter/scc)

Dual-licensed under MIT or the [UNLICENSE](http://unlicense.org).

Read all about how it came to be along with performance benchmarks https://boyter.org/posts/sloc-cloc-code/ or why use a code counting tool https://boyter.org/posts/why-count-lines-of-code/ or about making it even faster https://boyter.org/posts/sloc-cloc-code-performance/

For performance see the [Performance](https://github.com/boyter/scc#performance) section

Other similar projects,

 - https://github.com/Aaronepower/tokei
 - https://github.com/AlDanial/cloc
 - https://www.dwheeler.com/sloccount/
 - https://github.com/cgag/loc
 - https://github.com/hhatto/gocloc/
 - https://github.com/vmchale/polyglot

Interesting reading about about code counting about tokei, loc and polyglot

 - https://www.reddit.com/r/rust/comments/59bm3t/a_fast_cloc_replacement_in_rust/
 - https://www.reddit.com/r/rust/comments/82k9iy/loc_count_lines_of_code_quickly/
 - http://blog.vmchale.com/article/polyglot-comparisons

Further reading about processing files on the disk performance

 - https://blog.burntsushi.net/ripgrep/

### Pitch

Why use `scc`?

 - It is very fast and gets faster the more CPU you throw at it
 - Accurate
 - Works very well across multiple platforms without slowdown (Windows, Linux, macOS)
 - Large language support
 - Can ignore duplicate files
 - Has complexity estimations

Why not use `scc`?

 - Unable to tell the difference between Coq and Verilog (currently, if enough people raise a bug it will be resolved)
 - You don't like Go for some reason
 - You are working on a Linux system with less than 4 CPU cores and really need the fastest counter possible (use loc or polyglot)

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
   1.10.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --languages                         Print out supported languages and their extensions
   --format value, -f value            Set output format [possible values: tabular, wide, json, csv] (default: "tabular")
   --output FILE, -o FILE              Set output file if not set will print to stdout FILE
   --pathblacklist value, --pbl value  Which directories should be ignored as comma separated list (default: ".git,.hg,.svn")
   --sort value, -s value              Sort languages / files based on column [possible values: files, name, lines, blanks, code, comments, complexity] (default: "files")
   --whitelist value, --wl value       Restrict file extensions to just those provided as a comma separated list E.G. go,java,js
   --files                             Set to specify you want to see the output for every file
   --verbose, -v                       Set to enable verbose output
   --duplicates, -d                    Set to check for and remove duplicate files from stats and output
   --complexity, -c                    Set to skip complexity calculations note will be overridden if wide is set
   --wide, -w                          Set to check produce more output such as complexity and code vs complexity ranking. Same as setting format to wide
   --averagewage value, --aw value     Set as integer to set the average wage used for basic COCOMO calculation (default: 56286)
   --cocomo, --co                      Set to check remove COCOMO calculation output
   --filegccount value, --fgc value    How many files to parse before turning the GC on (default: 10000)
   --binary                            Set to disable binary file detection
   --debug                             Set to enable debug output
   --trace                             Set to enable trace output, not recommended for multiple files
   --help, -h                          show help
   --version, --ver                    Print the version
```

Output should look something like the below for the redis project

```
$ scc .
-------------------------------------------------------------------------------
Language                 Files     Lines     Code  Comments   Blanks Complexity
-------------------------------------------------------------------------------
C                          248    142725   102653     23885    16187      25729
C Header                   199     27451    18485      5746     3220       1554
TCL                         97     16356    13713       893     1750       1313
Shell                       36      1103      772       215      116         86
Lua                         20       525      387        70       68         65
Autoconf                    18     10817     8467      1325     1025        950
gitignore                   11       151      135         0       16          0
Makefile                     9      1031      722       100      209         50
Markdown                     8      1886     1363         0      523          0
Ruby                         8      2423     1905       323      195        292
HTML                         5      9658     8791        12      855          0
C++                          5       311      244        16       51         31
YAML                         4       273      254         0       19          0
License                      3        66       55         0       11          0
CSS                          2       107       91         0       16          0
Python                       2       219      160        19       40         67
Batch                        1        28       26         0        2          3
m4                           1       562      393        53      116          0
C++ Header                   1         9        5         3        1          0
Smarty Template              1        44       43         0        1          5
Plain Text                   1        23       16         0        7          0
-------------------------------------------------------------------------------
Total                      680    215768   158680     32660    24428      30145
-------------------------------------------------------------------------------
Estimated Cost to Develop $5,522,600
Estimated Schedule Effort 29.370095 months
Estimated People Required 22.273729
-------------------------------------------------------------------------------
```

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

Benchmarks are run on fresh 32 CPU Optimised Digital Ocean Virtual Machine 2018/09/19 all done using [hyperfine](https://github.com/sharkdp/hyperfine) with 3 warm-up runs and 10 timed runs.

```
scc v1.10.0
tokei v8.0.0 (compiled with Rust 1.29)
loc v0.4.1 (compiled with Rust 1.29)
polyglot v0.5.13
gocloc commit b3aa5f37096bbbfa22803a532214a11dbefa0206
```

#### Redis commit 7cdf272d46ad9b658ef8f5d8485af0eeb17cae6d https://github.com/antirez/redis/

| Program | Runtime |
|---|---|
| scc | 38.1 ms ±   1.2 ms |
| scc (no complexity) | 27.2 ms ±   1.7 ms |
| tokei | 17.2 ms ±   3.0 ms |
| loc | 31.5 ms ±  23.5 ms |
| polyglot | 15.2 ms ±   1.3 ms |
| gocloc | 129.3 ms ±   2.1 ms |

#### CPython commit 471503954a91d86cf04228c38134108c67a263b0

| Program | Runtime |
|---|---|
| scc | 69.8 ms ±   2.7 ms |
| scc (no complexity) | 55.9 ms ±   2.6 ms |
| tokei | 65.9 ms ±   6.4 ms |
| loc | 104.0 ms ±  58.4 ms |
| polyglot | 84.3 ms ±   8.4 ms |
| gocloc | 787.2 ms ±   7.1 ms |

#### Linux Kernel commit 7876320f88802b22d4e2daf7eb027dd14175a0f8 https://github.com/torvalds/linux

| Program | Runtime |
|---|---|
| scc | 623.5 ms ±  22.1 ms |
| scc (no complexity) | 471.1 ms ±  18.0 ms |
| tokei | 549.7 ms ±  30.8 ms |
| loc | 556.7 ms ± 145.7 ms |
| polyglot | 953.1 ms ±  31.0 ms |
| gocloc | 12.083 s ±  0.030 s |

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
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-1.0.0-x86_64-apple-darwin.zip scc
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-1.0.0-x86_64-pc-windows.zip scc.exe
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" && zip -r9 scc-1.0.0-x86_64-unknown-linux.zip scc
```

### Languages

List of supported languages. Note that this is always assumed that you built from master, and it might trail behind what is actually supported. To see what your version of `scc` supports run `scc --languages`

```
ABAP (abap)
ActionScript (as)
Ada (ada,adb,ads,pad)
Agda (agda)
Alex (x)
AsciiDoc (adoc)
ASP (asa,asp)
ASP.NET (asax,ascx,asmx,aspx,master,sitemap,webinfo)
Assembly (s,asm)
ATS (dats)
Autoconf (in)
AutoHotKey (ahk)
AWK (awk)
BASH (bash)
Basic (bas)
Batch (bat,btm,cmd)
Boo (tex)
Brainfuck (bf)
C (c,ec,pgc)
C Header (h)
C Shell (csh)
C# (cs)
C++ (cc,cpp,cxx,c++,pcc)
C++ Header (hh,hpp,hxx,inl,ipp)
Cabal (cabal)
Cassius (cassius)
Ceylon (ceylon)
Clojure (clj)
ClojureScript (cljs)
CMake (cmake,cmakelists.txt)
COBOL (cob,cbl,ccp,cobol,cpy)
CoffeeScript (coffee)
Cogent (cogent)
ColdFusion (cfm)
ColdFusion CFScript (cfc)
Coq (v)
Crystal (cr)
CSS (css)
CSV (csv)
Cython (pyx)
D (d)
Dart (dart)
Device Tree (dts,dtsi)
Dhall (dhall)
Dockerfile (dockerfile,dockerignore)
Document Type Definition (dtd)
Elixir (ex,exs)
Elm (elm)
Emacs Dev Env (ede)
Emacs Lisp (el)
Erlang (erl,hrl)
Expect (exp)
Extensible Stylesheet Language Transformations (xslt)
F# (fs,fsi,fsx,fsscript)
F* (fst)
Fish (fish)
Forth (4th,forth,fr,frt,fth,f83,fb,fpm,e4,rx,ft)
FORTRAN Legacy (f,for,ftn,f77,pfo)
FORTRAN Modern (f03,f08,f90,f95)
Freemarker Template (ftl)
GDScript (gd)
Gherkin Specification (feature)
gitignore (.gitignore)
GLSL (vert,tesc,tese,geom,frag,comp)
Go (go)
Go Template (tmpl)
Gradle (gradle)
Groovy (groovy,grt,gtpl,gvy)
Hamlet (hamlet)
Handlebars (hbs,handlebars)
Happy (y,ly)
Haskell (hs)
Haxe (hx)
HEX (hex)
HTML (html,htm)
IDL (idl)
Idris (idr,lidr)
Intel HEX (ihex)
Isabelle (thy)
Jade (jade)
JAI (jai)
Java (java)
JavaScript (js,mjs)
JavaServer Pages (jsp)
Jenkins Buildfile (jenkinsfile)
JSON (json)
JSX (jsx)
Julia (jl)
Julius (julius)
Korn Shell (ksh)
Kotlin (kt,kts)
LaTeX (tex)
LD Script (lds)
Lean (lean,hlean)
LESS (less)
LEX (l)
License (license,licence,copying,copying3,unlicense,unlicence)
Lisp (lisp,lsp)
LOLCODE (lol,lols)
Lua (lua)
Lucius (lucius)
m4 (m4)
Macromedia eXtensible Markup Language (mxml)
Madlang (mad)
Makefile (makefile,mak,mk)
Markdown (md,markdown)
Meson (meson.build,meson_options.txt)
Modula3 (m3,mg,ig,i3)
Module-Definition (def)
MQL Header (mqh)
MQL4 (mq4)
MQL5 (mq5)
MSBuild (csproj,vbproj,fsproj,props,targets)
MUMPS (mps)
Mustache (mustache)
Nim (nim)
Nix (nix)
Objective C (m)
Objective C++ (mm)
OCaml (ml,mli)
Opalang (opa)
Org (org)
Oz (oz)
Pascal (pas)
Patch (patch)
Perl (pl,pm)
PHP (php)
PKGBUILD (pkgbuild)
Plain Text (text,txt)
Polly (polly)
Powershell (ps1,psm1)
Processing (pde)
Prolog (p,pro)
Properties File (properties)
Protocol Buffers (proto)
PSL Assertion (psl)
Puppet (pp)
PureScript (purs)
Python (py)
QCL (qcl)
QML (qml)
R (r)
Rakefile (rake,rakefile)
Razor (cshtml)
Report Definition Language (rdl)
ReStructuredText (rst)
Robot Framework (robot)
Ruby (rb)
Ruby HTML (rhtml)
Rust (rs)
SAS (sas)
Sass (sass,scss)
Scala (sc,scala)
Scheme (scm,ss)
Scons (csig,sconstruct,sconscript)
sed (sed)
Shell (sh)
SKILL (il)
Smarty Template (tpl)
Softbridge Basic (sbl)
SPDX (spdx)
Specman e (e)
Spice Netlist (ckt)
SQL (sql)
SRecode Template (srt)
Standard ML (SML) (sml)
Stata (do,ado)
SVG (svg)
Swift (swift)
SystemVerilog (sv,svh)
TCL (tcl)
TeX (tex,sty)
Thrift (thrift)
TOML (toml)
TypeScript (ts,tsx)
TypeScript Typings (d.ts)
Unreal Script (uc,uci,upkg)
Ur/Web (ur,urs)
Ur/Web Project (urp)
Vala (vala)
Varnish Configuration (vcl)
Verilog (vg,vh)
Verilog Args File (irunargs,xrunargs)
VHDL (vhd)
Vim Script (vim)
Visual Basic (vb)
Vue (vue)
Wolfram (nb,wl)
XAML (xaml)
XCode Config (xcconfig)
XML (xml)
XML Schema (xsd)
Xtend (xtend)
YAML (yaml,yml)
Zig (zig)
Zsh (zsh)
```