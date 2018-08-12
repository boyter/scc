Sloc Cloc and Code (scc)
------------------------

A tool similar to cloc, sloccount and tokei. For counting physical the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform COCOMO calculation like sloccount and to estimate code complexity similar to cyclomatic complexity calculators. In short one tool to rule them all and the one I wish I had before I wrote it.

Also it has a very short name which is easy to type `scc`.

It is faster than loc/sloccount/cloc/gocloc with more accuracy. It is however marginally slower than tokei (on linux) but with duplicate detection and complexity count. If you disable garbage collection `GOGC=-1 scc .` it is indeed faster than everything with the exception of tokei (if compiled with 1.27+) by a huge margin.

[![Build Status](https://travis-ci.org/boyter/scc.svg?branch=master)](https://travis-ci.org/boyter/scc)
[![Go Report Card](https://goreportcard.com/badge/github.com/boyter/scc)](https://goreportcard.com/report/github.com/boyter/scc)

Dual-licensed under MIT or the [UNLICENSE](http://unlicense.org).

Read all about how it came to be along with performance benchmarks https://boyter.org/posts/sloc-cloc-code/ or why use a code counting tool https://boyter.org/posts/why-count-lines-of-code/

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
   1.6.0

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

Benchmarks run on fresh 32 CPU Optimised Digital Ocean Virtual Machine 2018/08/12 all done using [hyperfine](https://github.com/sharkdp/hyperfine) with 3 warm-up runs and 10 times runs.

```
scc v1.6.0
tokei v7.0.3 (compiled with Rust 1.27)
loc v0.4.1 (compiled with Rust 1.27)
polyglot v0.4.60
sloccount v2.26
cloc v1.6.0
```

#### Redis commit 39c70e728b5af0c50989ffbc05e568099f3e081b https://github.com/antirez/redis/

| Tool | Command | Time |
| ---- | ------- | ---- |
| scc (default) | `scc redis` | 20.1 ms ± 3.0 ms |
| scc (performance mode) | `GOGC=-1 scc -c -co redis` | 17.3 ms ± 2.2 ms |
| tokei | `tokei redis` | 20.9 ms ± 3.6 ms |
| loc | `loc redis` | 70.9 ms ± 31.7 ms |
| polyglot | `polyglot redis` | 24.5 ms  ± 3.6 ms |
| sloccount | `sloccount redis` | 1.002 s ± 0.012 s |
| cloc | `cloc redis` | 1.883 s ± 0.022 s |

#### Django commit d3449faaa915a08c275b35de01e66a7ef6bdb2dc https://github.com/django/django

| Tool | Command | Time |
| ---- | ------- | ---- |
| scc (default) | `scc django` | 53.5 ms ± 3.2 ms |
| scc (performance mode) | `GOGC=-1 scc -c -co django` | 49.5 ms ± 3.3 ms |
| tokei | `tokei django` | 73.5 ms ± 5.0 ms |
| loc | `loc django` | 328.2 ms ± 64.0 ms |
| polyglot | `polyglot django` | 90.4 ms ±  3.1 ms |
| sloccount | `sloccount django` | 2.644 s ± 0.026 s |
| cloc | `cloc django` | 14.711 s ± 0.228 s |

#### Linux Kernel commit ec0c96714e7ddeda4eccaa077f5646a0fd6e371f https://github.com/torvalds/linux

| Tool | Command | Time |
| ---- | ------- | ---- |
| scc (default) | `scc linux` | 1.212 s ± 0.055 s |
| scc (performance mode) | `GOGC=-1 scc -c -co linux` | 658.0 ms ±  30.3 ms |
| tokei | `tokei linux` | 559.7 ms ±  27.7 ms |
| loc | `loc linux` | 1.649 s ±  0.184 s |
| polyglot | `polyglot linux` | 1.034 s ±  0.031 s |
| sloccount | `sloccount linux` | 61.602 s ±  5.231 s |
| cloc | `cloc linux` | 178.112 s ±  12.129 s |


To run scc as quickly as possible use the command `GOGC=-1 scc -c -co .` which should run in a time comparable to tokei for most repositories. If you enable duplicate detection expect performance to fall by about 50%

### API Support

The core part of `scc` which is the counting engine is exposed publicly to be integrated into other Go applications. See https://github.com/pinpt/ripsrc for an example of how to do this.

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

### Low Memory

If you are running `scc` in a low memory envrionment < 512 MB of RAM you may need to set `--filegccount` or `--fgc` to a lower value such as `0` to force the garbage collector to be on at all times.

A sign that this is required will be `scc` crashing with panic errors.

### Tests

scc is pretty well tested with many unit, integration and benchmarks to ensure that it is fast and complete.

### Package

Run go build for windows and linux then the following in linux, keep in mind need to update the version

```
GOOS=darwin GOARCH=amd64 go build && zip -r9 scc-1.0.0-x86_64-apple-darwin.zip scc
GOOS=windows GOARCH=amd64 go build && zip -r9 scc-1.0.0-x86_64-pc-windows.zip scc.exe
GOOS=linux GOARCH=amd64 go build && zip -r9 scc-1.0.0-x86_64-unknown-linux.zip scc
```

### Languages

List of supported languages. Note that this is always assumed that you built from master, and it might trail behind what is actually supported. To see what your version of `scc` supports run `scc --languages`

```
ABAP (abap)
ActionScript (as)
Ada (ada,adb,ads,pad)
Agda (agda)
Alex (x)
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
Meson ()
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
Rakefile (rake)
Razor (cshtml)
Report Definition Language (rdl)
ReStructuredText (rst)
Robot Framework (robot)
Ruby (rb)
Ruby HTML (rhtml)
Rust (rs)
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
Zsh (zsh)
```