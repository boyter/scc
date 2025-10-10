# Sloc Cloc and Code (scc)

![SCC illustration](./scc.jpg)

A tool similar to cloc, sloccount and tokei. For counting the lines of code, blank lines, comment lines, and physical lines of source code in many programming languages.

Goal is to be the fastest code counter possible, but also perform COCOMO calculation like sloccount, estimate code complexity similar to cyclomatic complexity calculators and produce unique lines of code or DRYness metrics. In short one tool to rule them all.

Also it has a very short name which is easy to type `scc`.

If you don't like sloc cloc and code feel free to use the name `Succinct Code Counter`.

[![Go](https://github.com/boyter/scc/actions/workflows/go.yml/badge.svg)](https://github.com/boyter/scc/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/boyter/scc)](https://goreportcard.com/report/github.com/boyter/scc)
[![Coverage Status](https://coveralls.io/repos/github/boyter/scc/badge.svg?branch=master)](https://coveralls.io/github/boyter/scc?branch=master)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/)](https://github.com/boyter/scc/)
![Scc count downloads](https://img.shields.io/github/downloads/boyter/scc/total?label=downloads%20%28GH%29)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

Licensed under MIT licence.

## Table of Contents

- [Install](#install)
- [Background](#background)
- [Pitch](#pitch)
- [Usage](#usage)
- [Complexity Estimates](#complexity-estimates)
- [Unique Lines of Code (ULOC)](#unique-lines-of-code-uloc)
- [COCOMO](#cocomo)
- [Output Formats](#output-formats)
- [Performance](#performance)
- [Development](#development)
- [Adding/Modifying Languages](#addingmodifying-languages)
- [Issues](#issues)
- [Badges (beta)](#badges-beta)
- [Language Support](LANGUAGES.md)
- [Citation](#citation)

### scc for Teams & Enterprise

While scc will always be a free and tool for individual developers, companies and businesses, we are exploring an enhanced version designed for teams and businesses. scc Enterprise will build on the core scc engine to provide historical analysis, team-level dashboards, and policy enforcement to help engineering leaders track code health, manage technical debt, and forecast project costs.

We are currently gathering interest for a private beta. If you want to visualize your codebase's evolution, integrate quality gates into your CI/CD pipeline, and get a big-picture view across all your projects, 
sign up for the early access list [here](https://docs.google.com/forms/d/e/1FAIpQLScIBKy3y2m0rKu89L67qwe26Xyn9Scu0gW-HQX9lC0qEAx9nQ/viewform)

### Install

#### Go Install

You can install `scc` by using the standard go toolchain.

To install the latest stable version of scc:

`go install github.com/boyter/scc/v3@latest`

To install a development version:

`go install github.com/boyter/scc/v3@master`

Note that `scc` needs go version >= 1.25.

#### Snap

A [snap install](https://snapcraft.io/scc) exists thanks to [Ricardo](https://feliciano.tech/).

`$ sudo snap install scc`

*NB* Snap installed applications cannot run outside of `/home` <https://askubuntu.com/questions/930437/permission-denied-error-when-running-apps-installed-as-snap-packages-ubuntu-17> so you may encounter issues if you use snap and attempt to run outside this directory.

#### Homebrew

Or if you have [Homebrew](https://brew.sh/) installed

`$ brew install scc`

#### Fedora

Fedora Linux users can use a [COPR repository](https://copr.fedorainfracloud.org/coprs/lihaohong/scc/):

`$ sudo dnf copr enable lihaohong/scc && sudo dnf install scc`

#### MacPorts

On macOS, you can also install via [MacPorts](https://www.macports.org)

`$ sudo port install scc`

#### Scoop

Or if you are using [Scoop](https://scoop.sh/) on Windows

`$ scoop install scc`

#### Chocolatey

Or if you are using [Chocolatey](https://chocolatey.org/) on Windows

`$ choco install scc`

#### WinGet

Or if you are using [WinGet](https://github.com/microsoft/winget-cli) on Windows

`winget install --id benboyter.scc --source winget`

#### FreeBSD

On FreeBSD, scc is available as a package

`$ pkg install scc`

Or, if you prefer to build from source, you can use the ports tree

`$ cd /usr/ports/devel/scc && make install clean`

### Run in Docker

Go to the directory you want to run scc from.

Run the command below to run the latest release of scc on your current working directory:

```bash
docker run --rm -it -v "$PWD:/pwd"  ghcr.io/boyter/scc:master scc /pwd
```

#### Manual

Binaries for Windows, GNU/Linux and macOS for both i386 and x86_64 machines are available from the [releases](https://github.com/boyter/scc/releases) page.

#### GitLab

<https://about.gitlab.com/blog/2023/02/15/code-counting-in-gitlab/>

#### Other

If you would like to assist with getting `scc` added into apt/chocolatey/etc... please submit a PR or at least raise an issue with instructions.

### Background

Read all about how it came to be along with performance benchmarks,

- <https://boyter.org/posts/sloc-cloc-code/>
- <https://boyter.org/posts/why-count-lines-of-code/>
- <https://boyter.org/posts/sloc-cloc-code-revisited/>
- <https://boyter.org/posts/sloc-cloc-code-performance/>
- <https://boyter.org/posts/sloc-cloc-code-performance-update/>

Some reviews of `scc`

- <https://nickmchardy.com/2018/10/counting-lines-of-code-in-koi-cms.html>
- <https://www.feliciano.tech/blog/determine-source-code-size-and-complexity-with-scc/>
- <https://metaredux.com/posts/2019/12/13/counting-lines.html>

Setting up `scc` in GitLab

- <https://about.gitlab.com/blog/2023/02/15/code-counting-in-gitlab/>

A talk given at the first GopherCon AU about `scc` (press S to see speaker notes)

- <https://boyter.org/static/gophercon-syd-presentation/>
- <https://www.youtube.com/watch?v=jd-sjoy3GZo>

For performance see the [Performance](https://github.com/boyter/scc#performance) section

Other similar projects,

- [SLOCCount](https://www.dwheeler.com/sloccount/) the original sloc counter
- [cloc](https://github.com/AlDanial/cloc), inspired by SLOCCount; implemented in Perl for portability
- [gocloc](https://github.com/hhatto/gocloc) a sloc counter in Go inspired by tokei
- [loc](https://github.com/cgag/loc) rust implementation similar to tokei but often faster
- [loccount](https://gitlab.com/esr/loccount) Go implementation written and maintained by ESR
- [polyglot](https://github.com/vmchale/polyglot) ATS sloc counter
- [tokei](https://github.com/XAMPPRocky/tokei) fast, accurate and written in rust
- [sloc](https://github.com/flosse/sloc) coffeescript code counter
- [stto](https://github.com/mainak55512/stto) new Go code counter with a focus on performance

Interesting reading about other code counting projects tokei, loc, polyglot and loccount

- <https://www.reddit.com/r/rust/comments/59bm3t/a_fast_cloc_replacement_in_rust/>
- <https://www.reddit.com/r/rust/comments/82k9iy/loc_count_lines_of_code_quickly/>
- <http://blog.vmchale.com/article/polyglot-comparisons>
- <http://esr.ibiblio.org/?p=8270>

Further reading about processing files on the disk performance

- <https://blog.burntsushi.net/ripgrep/>

Using `scc` to process 40 TB of files from GitHub/Bitbucket/GitLab

- <https://boyter.org/posts/an-informal-survey-of-10-million-github-bitbucket-gitlab-projects/>

### Pitch

Why use `scc`?

- It is very fast and gets faster the more CPU you throw at it
- Accurate
- Works very well across multiple platforms without slowdown (Windows, Linux, macOS)
- Large language support
- Can ignore duplicate files
- Has complexity estimations
- You need to tell the difference between Coq and Verilog in the same directory
- cloc yaml output support so potentially a drop in replacement for some users
- Can identify or ignore minified files
- Able to identify many #! files ADVANCED! <https://github.com/boyter/scc/issues/115>
- Can ignore large files by lines or bytes
- Can calculate the ULOC or unique lines of code by file, language or project
- Supports multiple output formats for integration, CSV, SQL, JSON, HTML and more

Why not use `scc`?

- You don't like Go for some reason
- It cannot count D source with different nested multi-line comments correctly <https://github.com/boyter/scc/issues/27>

### Differences

There are some important differences between `scc` and other tools that are out there. Here are a few important ones for you to consider.

Blank lines inside comments are counted as comments. While the line is technically blank the decision was made that once in a comment everything there should be considered a comment until that comment is ended. As such the following,

```c
/* blank lines follow


*/
```

Would be counted as 4 lines of comments. This is noticeable when comparing scc's output to other tools on large
repositories.

`scc` is able to count verbatim strings correctly. For example in C# the following,

```C#
private const string BasePath = @"a:\";
// The below is returned to the user as a version
private const string Version = "1.0.0";
```

Because of the prefixed @ this string ends at the trailing " by ignoring the escape character \ and as such should be
counted as 2 code lines and 1 comment. Some tools are unable to
deal with this and instead count up to the "1.0.0" as a string which can cause the middle comment to be counted as
code rather than a comment.

`scc` will also tell you the number of bytes it has processed (for most output formats) allowing you to estimate the
cost of running some static analysis tools.

### Usage

Command line usage of `scc` is designed to be as simple as possible.
Full details can be found in `scc --help` or `scc -h`. Note that the below reflects the state of master not a release, as such
features listed below may be missing from your installation.

```text
Sloc, Cloc and Code. Count lines of code in a directory with complexity estimation.
Version 3.5.0 (beta)
Ben Boyter <ben@boyter.org> + Contributors

Usage:
  scc [flags] [files or directories]

Flags:
      --avg-wage int                       average wage value used for basic COCOMO calculation (default 56286)
      --binary                             disable binary file detection
      --by-file                            display output for every file
  -m, --character                          calculate max and mean characters per line
      --ci                                 enable CI output settings where stdout is ASCII
      --cocomo-project-type string         change COCOMO model type [organic, semi-detached, embedded, "custom,1,1,1,1"] (default "organic")
      --count-as string                    count extension as language [e.g. jsp:htm,chead:"C Header" maps extension jsp to html and chead to C Header]
      --count-ignore                       set to allow .gitignore and .ignore files to be counted
      --currency-symbol string             set currency symbol (default "$")
      --debug                              enable debug output
      --directory-walker-job-workers int   controls the maximum number of workers which will walk the directory tree (default 8)
  -a, --dryness                            calculate the DRYness of the project (implies --uloc)
      --eaf float                          the effort adjustment factor derived from the cost drivers (1.0 if rated nominal) (default 1)
      --exclude-dir strings                directories to exclude (default [.git,.hg,.svn])
  -x, --exclude-ext strings                ignore file extensions (overrides include-ext) [comma separated list: e.g. go,java,js]
  -n, --exclude-file strings               ignore files with matching names (default [package-lock.json,Cargo.lock,yarn.lock,pubspec.lock,Podfile.lock,pnpm-lock.yaml])
      --file-gc-count int                  number of files to parse before turning the GC on (default 10000)
      --file-list-queue-size int           the size of the queue of files found and ready to be read into memory (default 8)
      --file-process-job-workers int       number of goroutine workers that process files collecting stats (default 8)
      --file-summary-job-queue-size int    the size of the queue used to hold processed file statistics before formatting (default 8)
  -f, --format string                      set output format [tabular, wide, json, json2, csv, csv-stream, cloc-yaml, html, html-table, sql, sql-insert, openmetrics] (default "tabular")
      --format-multi string                have multiple format output overriding --format [e.g. tabular:stdout,csv:file.csv,json:file.json]
      --gen                                identify generated files
      --generated-markers strings          string markers in head of generated files (default [do not edit,<auto-generated />])
  -h, --help                               help for scc
  -i, --include-ext strings                limit to file extensions [comma separated list: e.g. go,java,js]
      --include-symlinks                   if set will count symlink files
  -l, --languages                          print supported languages and extensions
      --large-byte-count int               number of bytes a file can contain before being removed from output (default 1000000)
      --large-line-count int               number of lines a file can contain before being removed from output (default 40000)
      --min                                identify minified files
  -z, --min-gen                            identify minified or generated files
      --min-gen-line-length int            number of bytes per average line for file to be considered minified or generated (default 255)
      --no-cocomo                          remove COCOMO calculation output
  -c, --no-complexity                      skip calculation of code complexity
  -d, --no-duplicates                      remove duplicate files from stats and output
      --no-gen                             ignore generated files in output (implies --gen)
      --no-gitignore                       disables .gitignore file logic
      --no-gitmodule                       disables .gitmodules file logic
      --no-hborder                         remove horizontal borders between sections
      --no-ignore                          disables .ignore file logic
      --no-large                           ignore files over certain byte and line size set by large-line-count and large-byte-count
      --no-min                             ignore minified files in output (implies --min)
      --no-min-gen                         ignore minified or generated files in output (implies --min-gen)
      --no-scc-ignore                      disables .sccignore file logic
      --no-size                            remove size calculation output
  -M, --not-match stringArray              ignore files and directories matching regular expression
  -o, --output string                      output filename (default stdout)
      --overhead float                     set the overhead multiplier for corporate overhead (facilities, equipment, accounting, etc.) (default 2.4)
  -p, --percent                            include percentage values in output
      --remap-all string                   inspect every file and remap by checking for a string and remapping the language [e.g. "-*- C++ -*-":"C Header"]
      --remap-unknown string               inspect files of unknown type and remap by checking for a string and remapping the language [e.g. "-*- C++ -*-":"C Header"]
      --size-unit string                   set size unit [si, binary, mixed, xkcd-kb, xkcd-kelly, xkcd-imaginary, xkcd-intel, xkcd-drive, xkcd-bakers] (default "si")
      --sloccount-format                   print a more SLOCCount like COCOMO calculation
  -s, --sort string                        column to sort by [files, name, lines, blanks, code, comments, complexity] (default "files")
      --sql-project string                 use supplied name as the project identifier for the current run. Only valid with the --format sql or sql-insert option
  -t, --trace                              enable trace output (not recommended when processing multiple files)
  -u, --uloc                               calculate the number of unique lines of code (ULOC) for the project
  -v, --verbose                            verbose output
      --version                            version for scc
  -w, --wide                               wider output with additional statistics (implies --complexity)
```

Output should look something like the below for the redis project

```text
$ scc redis 
───────────────────────────────────────────────────────────────────────────────
Language                 Files     Lines   Blanks  Comments     Code Complexity
───────────────────────────────────────────────────────────────────────────────
C                          437   267,353   31,103    45,998  190,252     48,269
JSON                       406    25,392        4         0   25,388          0
C Header                   288    48,831    5,648    11,302   31,881      3,097
TCL                        215    66,943    7,330     4,651   54,962      3,816
Shell                       75     1,626      239       343    1,044        185
Python                      34     4,802      694       498    3,610        621
Markdown                    26     4,647    1,226         0    3,421          0
Autoconf                    22    11,732    1,124     1,420    9,188      1,016
Lua                         20       525       69        71      385         89
Makefile                    20     1,956      368       170    1,418         85
YAML                        20     2,696      147        53    2,496          0
MSBuild                     11     1,995        2         0    1,993        160
Plain Text                  10     1,773      313         0    1,460          0
Ruby                         9       817       73       105      639        123
C++                          8       546       85        43      418         43
HTML                         5     9,658    2,928        12    6,718          0
License                      3        90       17         0       73          0
CMake                        2       298       49         5      244         12
CSS                          2       107       16         0       91          0
Systemd                      2        80        6         0       74          0
BASH                         1       143       16         5      122         38
Batch                        1        28        2         0       26          3
C++ Header                   1         9        1         3        5          0
Extensible Styleshe…         1        10        0         0       10          0
JavaScript                   1        31        1         0       30          5
Module-Definition            1    11,375    2,116         0    9,259        167
SVG                          1         1        0         0        1          0
Smarty Template              1        44        1         0       43          5
m4                           1       951      218        64      669          0
───────────────────────────────────────────────────────────────────────────────
Total                    1,624   464,459   53,796    64,743  345,920     57,734
───────────────────────────────────────────────────────────────────────────────
Estimated Cost to Develop (organic) $12,517,562
Estimated Schedule Effort (organic) 35.93 months
Estimated People Required (organic) 30.95
───────────────────────────────────────────────────────────────────────────────
Processed 16601962 bytes, 16.602 megabytes (SI)
───────────────────────────────────────────────────────────────────────────────
```

Note that you don't have to specify the directory you want to run against. Running `scc` will assume you want to run against the current directory.

You can also run against multiple files or directories `scc directory1 directory2 file1 file2` with the results aggregated in the output.

Since `scc` writes to standard output, there are many ways to easily share the results. For example, using [netcat](https://manpages.org/nc)
and [one of many pastebins](https://paste.c-net.org/) gives a public URL:

```bash
$ scc | nc paste.c-net.org 9999
https://paste.c-net.org/Example
```

### Ignore Files

`scc` mostly supports .ignore files inside directories that it scans. This is similar to how ripgrep, ag and tokei work. .ignore files are 100% the same as .gitignore files with the same syntax, and as such `scc` will ignore files and directories listed in them. You can add .ignore files to ignore things like vendored dependency checked in files and such. The idea is allowing you to add a file or folder to git and have ignored in the count.

It also supports its own ignore file `.sccignore` if you want `scc` to ignore things while having ripgrep, ag, tokei and others support them.

### Interesting Use Cases

Used inside Intel Nemu Hypervisor to track code changes between revisions <https://github.com/intel/nemu/blob/topic/virt-x86/tools/cloc-change.sh#L9>
Appears to also be used inside both <http://codescoop.com/> <https://pinpoint.com/> <https://github.com/chaoss/grimoirelab-graal>

It also is used to count code and guess language types in <https://searchcode.com/> which makes it one of the most frequently run code counters in the world.

You can also hook scc into your gitlab pipeline <https://gitlab.com/guided-explorations/ci-cd-plugin-extensions/ci-cd-plugin-extension-scc>

Also used by CodeQL <https://github.com/boyter/scc/pull/317> and Scaleway <https://twitter.com/Scaleway/status/1488087029476995074?s=20&t=N2-z6O-ISDdDzULg4o4uVQ>

- <https://docs.linuxfoundation.org/lfx/insights/v3-beta-version-current/getting-started/landing-page/cocomo-cost-estimation-simplified>
- <https://openems.io/>

### Features

`scc` uses a small state machine in order to determine what state the code is when it reaches a newline `\n`. As such it is aware of and able to count

- Single Line Comments
- Multi Line Comments
- Strings
- Multi Line Strings
- Blank lines

Because of this it is able to accurately determine if a comment is in a string or is actually a comment.

It also attempts to count the complexity of code. This is done by checking for branching operations in the code. For example, each of the following `for if switch while else || && != ==` if encountered in Java would increment that files complexity by one.

### Complexity Estimates

Let's take a minute to discuss the complexity estimate itself.

The complexity estimate is really just a number that is only comparable to files in the same language. It should not be used to compare languages directly without weighting them. The reason for this is that its calculated by looking for branch and loop statements in the code and incrementing a counter for that file.

Because some languages don't have loops and instead use recursion they can have a lower complexity count. Does this mean they are less complex? Probably not, but the tool cannot see this because it does not build an AST of the code as it only scans through it.

Generally though the complexity there is to help estimate between projects written in the same language, or for finding the most complex file in a project `scc --by-file -s complexity` which can be useful when you are estimating on how hard something is to maintain, or when looking for those files that should probably be refactored.

As for how it works.

It's my own definition, but tries to be an approximation of cyclomatic complexity <https://en.wikipedia.org/wiki/Cyclomatic_complexity> although done only on a file level.

The reason it's an approximation is that it's calculated almost for free from a CPU point of view (since its a cheap lookup when counting), whereas a real cyclomatic complexity count would need to parse the code. It gives a reasonable guess in practice though even if it fails to identify recursive methods. The goal was never for it to be exact.

In short when scc is looking through what it has identified as code if it notices what are usually branch conditions it will increment a counter.

The conditions it looks for are compiled into the code and you can get an idea for them by looking at the JSON inside the repository. See <https://github.com/boyter/scc/blob/master/languages.json#L3869> for an example of what it's looking at for a file that's Java.

The increment happens for each of the matching conditions and produces the number you see.

### Unique Lines of Code (ULOC)

ULOC stands for Unique Lines of Code and represents the unique lines across languages, files and the project itself. This idea was taken from
<https://cmcenroe.me/2018/12/14/uloc.html> where the calculation is presented using standard Unix tools `sort -u *.h *.c | wc -l`. This metric is
there to assist with the estimation of complexity within the project. Quoting the source

> In my opinion, the number this produces should be a better estimate of the complexity of a project. Compared to SLOC, not only are blank lines discounted, but so are close-brace lines and other repetitive code such as common includes. On the other hand, ULOC counts comments, which require just as much maintenance as the code around them does, while avoiding inflating the result with license headers which appear in every file, for example.

You can obtain the ULOC by supplying the `-u` or `--uloc` argument to `scc`.

It has a corresponding metric `DRYness %` which is the percentage of ULOC to CLOC or `DRYness = ULOC / SLOC`. The
higher the number the more DRY (don't repeat yourself) the project can be considered. In general a higher value
here is a better as it indicates less duplicated code. The DRYness metric was taken from a comment by minimax <https://lobste.rs/s/has9r7/uloc_unique_lines_code>

To obtain the DRYness metric you can use the `-a` or `--dryness` argument to `scc`, which will implicitly set `--uloc`.

Note that there is a performance penalty when calculating the ULOC metrics which can double the runtime.

Running the uloc and DRYness calculations against C code a clone of redis produces an output as follows.

```bash
$ scc -a -i c redis 
───────────────────────────────────────────────────────────────────────────────
Language                 Files     Lines   Blanks  Comments     Code Complexity
───────────────────────────────────────────────────────────────────────────────
C                          437   267,353   31,103    45,998  190,252     48,269
(ULOC)                            149892
───────────────────────────────────────────────────────────────────────────────
Total                      437   267,353   31,103    45,998  190,252     48,269
───────────────────────────────────────────────────────────────────────────────
Unique Lines of Code (ULOC)       149892
DRYness %                           0.56
───────────────────────────────────────────────────────────────────────────────
Estimated Cost to Develop (organic) $6,681,762
Estimated Schedule Effort (organic) 28.31 months
Estimated People Required (organic) 20.97
───────────────────────────────────────────────────────────────────────────────
Processed 9390815 bytes, 9.391 megabytes (SI)
───────────────────────────────────────────────────────────────────────────────
```

Further reading about the ULOC calculation can be found at <https://boyter.org/posts/sloc-cloc-code-new-metic-uloc/>

### COCOMO

The COCOMO statistics displayed at the bottom of any command line run can be configured as needed.

```text
Estimated Cost to Develop (organic) $664,081
Estimated Schedule Effort (organic) 11.772217 months
Estimated People Required (organic) 5.011633
```

To change the COCOMO parameters, you can either use one of the default COCOMO models.

```text
scc --cocomo-project-type organic
scc --cocomo-project-type semi-detached
scc --cocomo-project-type embedded
```

You can also supply your own parameters if you are familiar with COCOMO as follows,

```text
scc --cocomo-project-type "custom,1,1,1,1"
```

See below for details about how the model choices, and the parameters they use.

Organic – A software project is said to be an organic type if the team size required is adequately small, the
problem is well understood and has been solved in the past and also the team members have a nominal experience
regarding the problem.

`scc --cocomo-project-type "organic,2.4,1.05,2.5,0.38"`

Semi-detached – A software project is said to be a Semi-detached type if the vital characteristics such as team-size,
experience, knowledge of the various programming environment lie in between that of organic and Embedded.
The projects classified as Semi-Detached are comparatively less familiar and difficult to develop compared to
the organic ones and require more experience and better guidance and creativity. Eg: Compilers or
different Embedded Systems can be considered of Semi-Detached type.

`scc --cocomo-project-type "semi-detached,3.0,1.12,2.5,0.35"`

Embedded – A software project with requiring the highest level of complexity, creativity, and experience
requirement fall under this category. Such software requires a larger team size than the other two models
and also the developers need to be sufficiently experienced and creative to develop such complex models.

`scc --cocomo-project-type "embedded,3.6,1.20,2.5,0.32"`

### Large File Detection

You can have `scc` exclude large files from the output.

The option to do so is `--no-large` which by default will exclude files over 1,000,000 bytes or 40,000 lines.

You can control the size of either value using `--large-byte-count` or `--large-line-count`.

For example to exclude files over 1,000 lines and 50kb you could use the following,

`scc --no-large --large-byte-count 50000 --large-line-count 1000`

### Minified/Generated File Detection

You can have `scc` identify and optionally remove files identified as being minified or generated from the output.

You can do so by enabling the `-z` flag like so `scc -z` which will identify any file with an average line byte size >= 255 (by default) as being minified.

Minified files appear like so in the output.

```text
$ scc --no-cocomo -z ./examples/minified/jquery-3.1.1.min.js
───────────────────────────────────────────────────────────────────────────────
Language                 Files     Lines   Blanks  Comments     Code Complexity
───────────────────────────────────────────────────────────────────────────────
JavaScript (min)             1         4        0         1        3         17
───────────────────────────────────────────────────────────────────────────────
Total                        1         4        0         1        3         17
───────────────────────────────────────────────────────────────────────────────
Processed 86709 bytes, 0.087 megabytes (SI)
───────────────────────────────────────────────────────────────────────────────
```

Minified files are indicated with the text `(min)` after the language name.

Generated files are indicated with the text `(gen)` after the language name.

You can control the average line byte size using `--min-gen-line-length` such as `scc -z --min-gen-line-length 1`. Please note you need `-z` as modifying this value does not imply minified detection.

You can exclude minified files from the count totally using the flag `--no-min-gen`. Files which match the minified check will be excluded from the output.

### Remapping

Some files may not have an extension. They will be checked to see if they are a #! file. If they are then the language will be remapped to the
correct language. Otherwise, it will not process.

However, you may have the situation where you want to remap such files based on a string inside it. To do so you can use `--remap-unknown`

```bash
 scc --remap-unknown "-*- C++ -*-":"C Header"
```

The above will inspect any file with no extension looking for the string `-*- C++ -*-` and if found remap the file to be counted using the C Header rules.
You can have multiple remap rules if required,

```bash
 scc --remap-unknown "-*- C++ -*-":"C Header","other":"Java"
```

There is also the `--remap-all` parameter which will remap all files.

Note that in all cases if the remap rule does not apply normal #! rules will apply.

### Output Formats

By default `scc` will output to the console. However, you can produce output in other formats if you require.

The different options are `tabular, wide, json, csv, csv-stream, cloc-yaml, html, html-table, sql, sql-insert, openmetrics`.

Note that you can write `scc` output to disk using the `-o, --output` option. This allows you to specify a file to
write your output to. For example `scc -f html -o output.html` will run `scc` against the current directory, and output
the results in html to the file `output.html`.

You can also write to multiple output files, or multiple types to stdout if you want using the `--format-multi` option. This is
most useful when working in CI/CD systems where you want HTML reports as an artifact while also displaying the counts in stdout.

```bash
scc --format-multi "tabular:stdout,html:output.html,csv:output.csv"
```

The above will run against the current directory, outputting to standard output the default output, as well as writing
to output.html and output.csv with the appropriate formats.

#### Tabular

This is the default output format when scc is run.

#### Wide

Wide produces some additional information which is the complexity/lines metric. This can be useful when trying to
identify the most complex file inside a project based on the complexity estimate.

#### JSON

JSON produces JSON output. Mostly designed to allow `scc` to feed into other programs.

Note that this format will give you the byte size of every file `scc` reads allowing you to get a breakdown of the
number of bytes processed.

#### CSV

CSV as an option is good for importing into a spreadsheet for analysis.

Note that this format will give you the byte size of every file `scc` reads allowing you to get a breakdown of the
number of bytes processed. Also note that CSV respects `--by-file` and as such will return a summary by default.

#### CSV-Stream

csv-stream is an option useful for processing very large repositories where you are likely to run into memory issues. It's output format is 100% the same as CSV.

Note that you should not use this with the `format-multi` option as it will always print to standard output, and because of how it works will negate the memory saving it normally gains.
savings that this option provides. Note that there is no sort applied with this option.

#### cloc-yaml

Is a drop in replacement for cloc using its yaml output option. This is quite often used for passing into other
build systems and can help with replacing cloc if required.

```text
$ scc -f cloc-yml processor
# https://github.com/boyter/scc/
header:
  url: https://github.com/boyter/scc/
  version: 2.11.0
  elapsed_seconds: 0.008
  n_files: 21
  n_lines: 6562
  files_per_second: 2625
  lines_per_second: 820250
Go:
  name: Go
  code: 5186
  comment: 273
  blank: 1103
  nFiles: 21
SUM:
  code: 5186
  comment: 273
  blank: 1103
  nFiles: 21

$ cloc --yaml processor
      21 text files.
      21 unique files.
       0 files ignored.

---
# http://cloc.sourceforge.net
header :
  cloc_url           : http://cloc.sourceforge.net
  cloc_version       : 1.60
  elapsed_seconds    : 0.196972846984863
  n_files            : 21
  n_lines            : 6562
  files_per_second   : 106.613679608407
  lines_per_second   : 33314.2364566841
Go:
  nFiles: 21
  blank: 1137
  comment: 606
  code: 4819
SUM:
  blank: 1137
  code: 4819
  comment: 606
  nFiles: 21
```

#### HTML and HTML-TABLE

The HTML output options produce a minimal html report using a table that is either standalone `html` or as just a table `html-table`
which can be injected into your own HTML pages. The only difference between the two is that the `html` option includes
html head and body tags with minimal styling.

The markup is designed to allow your own custom styles to be applied. An example report
[is here to view](SCC-OUTPUT-REPORT.html).

Note that the HTML options follow the command line options, so you can use `scc --by-file -f html` to produce a report with every
file and not just the summary.

Note that this format if it has the `--by-file` option will give you the byte size of every file `scc` reads allowing you to get a breakdown of the
number of bytes processed.

#### SQL and SQL-Insert

The SQL output format "mostly" compatible with cloc's SQL output format <https://github.com/AlDanial/cloc#sql->

While all queries on the cloc documentation should work as expected, you will not be able to append output from `scc` and `cloc` into the same database. This is because the table format is slightly different
to account for scc including complexity counts and bytes.

The difference between `sql` and `sql-insert` is that `sql` will include table creation while the latter will only have the insert commands.

Usage is 100% the same as any other `scc` command but sql output will always contain per file details. You can compute totals yourself using SQL, however COCOMO calculations will appear against the metadata table as the columns `estimated_cost` `estimated_schedule_months` and `estimated_people`.

The below will run scc against the current directory, name the output as the project scc and then pipe the output to sqlite to put into the database code.db

```bash
scc --format sql --sql-project scc . | sqlite3 code.db
```

Assuming you then wanted to append another project

```bash
scc --format sql-insert --sql-project redis . | sqlite3 code.db
```

You could then run SQL against the database,

```bash
sqlite3 code.db 'select project,file,max(nCode) as nL from t
                         group by project order by nL desc;'
```

See the cloc documentation for more examples.

#### OpenMetrics

[OpenMetrics](https://openmetrics.io/) is a metric reporting format specification extending the Prometheus exposition text format.

The produced output is natively supported by [Prometheus](https://prometheus.io/) and [GitLab CI](https://docs.gitlab.com/ee/ci/testing/metrics_reports.html)

Note that OpenMetrics respects `--by-file` and as such will return a summary by default.

The output includes a metadata header containing definitions of the returned metrics:

```text
# TYPE scc_files count
# HELP scc_files Number of sourcecode files.
# TYPE scc_lines count
# UNIT scc_lines lines
# HELP scc_lines Number of lines.
# TYPE scc_code count
# HELP scc_code Number of lines of actual code.
# TYPE scc_comments count
# HELP scc_comments Number of comments.
# TYPE scc_blanks count
# HELP scc_blanks Number of blank lines.
# TYPE scc_complexity count
# HELP scc_complexity Code complexity.
# TYPE scc_bytes count
# UNIT scc_bytes bytes
# HELP scc_bytes Size in bytes.
```

The header is followed by the metric data in either language summary form:

```text
scc_files{language="Go"} 1
scc_lines{language="Go"} 1000
scc_code{language="Go"} 1000
scc_comments{language="Go"} 1000
scc_blanks{language="Go"} 1000
scc_complexity{language="Go"} 1000
scc_bytes{language="Go"} 1000
```

or, if `--by-file` is present, in per file form:

```text
scc_lines{language="Go",file="./bbbb.go"} 1000
scc_code{language="Go",file="./bbbb.go"} 1000
scc_comments{language="Go",file="./bbbb.go"} 1000
scc_blanks{language="Go",file="./bbbb.go"} 1000
scc_complexity{language="Go",file="./bbbb.go"} 1000
scc_bytes{language="Go",file="./bbbb.go"} 1000
```

### Performance

Generally `scc` will the fastest code counter compared to any I am aware of and have compared against. The below comparisons are taken from the fastest alternative counters. See `Other similar projects` above to see all of the other code counters compared against. It is designed to scale to as many CPU's cores as you can provide.

However, if you want greater performance and you have RAM to spare you can disable the garbage collector like the following on Linux `GOGC=-1 scc .` which should speed things up considerably. For some repositories turning off the code complexity calculation via `-c` can reduce runtime as well.

Benchmarks are run on fresh 48 Core CPU Optimised Digital Ocean Virtual Machine 2024/09/30 all done using [hyperfine](https://github.com/sharkdp/hyperfine).

See <https://github.com/boyter/scc/blob/master/benchmark.sh> to see how the benchmarks are run.

#### Valkey <https://github.com/valkey-io/valkey>

```shell
Benchmark 1: scc valkey
  Time (mean ± σ):      28.0 ms ±   1.6 ms    [User: 166.1 ms, System: 55.0 ms]
  Range (min … max):    24.7 ms …  31.5 ms    114 runs
 
Benchmark 2: scc -c valkey
  Time (mean ± σ):      25.8 ms ±   1.7 ms    [User: 123.7 ms, System: 53.2 ms]
  Range (min … max):    23.3 ms …  29.3 ms    114 runs
 
Benchmark 3: tokei valkey
  Time (mean ± σ):      63.0 ms ±   3.8 ms    [User: 433.8 ms, System: 244.3 ms]
  Range (min … max):    46.7 ms …  67.6 ms    44 runs
 
Benchmark 4: polyglot valkey
  Time (mean ± σ):      27.4 ms ±   0.8 ms    [User: 46.5 ms, System: 79.0 ms]
  Range (min … max):    25.7 ms …  29.5 ms    108 runs
 
Summary
  scc -c valkey ran
    1.06 ± 0.08 times faster than polyglot valkey
    1.08 ± 0.09 times faster than scc valkey
    2.44 ± 0.22 times faster than tokei valkey
```

#### CPython <https://github.com/python/cpython>

```shell
Benchmark 1: scc cpython
  Time (mean ± σ):      81.9 ms ±   4.2 ms    [User: 789.6 ms, System: 164.6 ms]
  Range (min … max):    74.0 ms …  89.6 ms    36 runs
 
Benchmark 2: scc -c cpython
  Time (mean ± σ):      75.4 ms ±   4.6 ms    [User: 621.9 ms, System: 152.6 ms]
  Range (min … max):    68.4 ms …  84.5 ms    37 runs
 
Benchmark 3: tokei cpython
  Time (mean ± σ):     162.1 ms ±   3.4 ms    [User: 1824.0 ms, System: 420.4 ms]
  Range (min … max):   156.7 ms … 168.9 ms    18 runs
 
Benchmark 4: polyglot cpython
  Time (mean ± σ):      94.2 ms ±   3.0 ms    [User: 210.3 ms, System: 260.3 ms]
  Range (min … max):    88.3 ms …  99.4 ms    30 runs
 
Summary
  scc -c cpython ran
    1.09 ± 0.09 times faster than scc cpython
    1.25 ± 0.09 times faster than polyglot cpython
    2.15 ± 0.14 times faster than tokei cpython
```

#### Linux Kernel <https://github.com/torvalds/linux>

```shell
Benchmark 1: scc linux
  Time (mean ± σ):      1.070 s ±  0.036 s    [User: 15.253 s, System: 1.962 s]
  Range (min … max):    1.011 s …  1.133 s    10 runs
 
Benchmark 2: scc -c linux
  Time (mean ± σ):      1.007 s ±  0.039 s    [User: 9.822 s, System: 1.937 s]
  Range (min … max):    0.915 s …  1.043 s    10 runs
 
Benchmark 3: tokei linux
  Time (mean ± σ):      1.094 s ±  0.019 s    [User: 19.416 s, System: 11.085 s]
  Range (min … max):    1.067 s …  1.135 s    10 runs
 
Benchmark 4: polyglot linux
  Time (mean ± σ):      1.387 s ±  0.028 s    [User: 3.775 s, System: 3.212 s]
  Range (min … max):    1.359 s …  1.433 s    10 runs
 
Summary
  scc -c linux ran
    1.06 ± 0.05 times faster than scc linux
    1.09 ± 0.05 times faster than tokei linux
    1.38 ± 0.06 times faster than polyglot linux
```

#### Sourcegraph <https://github.com/SINTEF/sourcegraph.git>

Sourcegraph has gone dark since I last ran these benchmarks hence using a clone taken before this occured.
The reason for this is to track what appears to be a performance regression in tokei.

```shell
Benchmark 1: scc sourcegraph
  Time (mean ± σ):     125.1 ms ±   8.0 ms    [User: 638.1 ms, System: 218.0 ms]
  Range (min … max):   116.7 ms … 141.3 ms    24 runs
 
Benchmark 2: scc -c sourcegraph
  Time (mean ± σ):     119.8 ms ±   8.3 ms    [User: 554.8 ms, System: 208.6 ms]
  Range (min … max):   111.9 ms … 138.4 ms    22 runs
 
Benchmark 3: tokei sourcegraph
  Time (mean ± σ):     23.888 s ±  1.416 s    [User: 73.858 s, System: 630.906 s]
  Range (min … max):   22.292 s … 27.010 s    10 runs
 
Benchmark 4: polyglot sourcegraph
  Time (mean ± σ):     113.3 ms ±   4.1 ms    [User: 237.7 ms, System: 791.8 ms]
  Range (min … max):   107.9 ms … 124.3 ms    26 runs
 
Summary
  polyglot sourcegraph ran
    1.06 ± 0.08 times faster than scc -c sourcegraph
    1.10 ± 0.08 times faster than scc sourcegraph
  210.86 ± 14.66 times faster than tokei sourcegraph

```

If you enable duplicate detection expect performance to fall by about 20% in `scc`.

Performance is tracked for some releases and presented below.

[![scc perfromance on Linux kernel](./performance-over-time.png)]
The decrease in performance from the 3.3.0 release was due to accurate .gitignore, .ignore and .gitmodule support.
Current work is focussed on resolving this.

<https://jsfiddle.net/mw21h9va/>

### CI/CD Support

Some CI/CD systems which will remain nameless do not work very well with the box-lines used by `scc`. To support those systems better there is an option `--ci` which will change the default output to ASCII only.

```text
$ scc --ci main.go
-------------------------------------------------------------------------------
Language                 Files     Lines   Blanks  Comments     Code Complexity
-------------------------------------------------------------------------------
Go                           1       272        7         6      259          4
-------------------------------------------------------------------------------
Total                        1       272        7         6      259          4
-------------------------------------------------------------------------------
Estimated Cost to Develop $6,539
Estimated Schedule Effort 2.268839 months
Estimated People Required 0.341437
-------------------------------------------------------------------------------
Processed 5674 bytes, 0.006 megabytes (SI)
-------------------------------------------------------------------------------
```

The `--format-multi` option is especially useful in CI/CD where you want to get multiple output formats useful for storage or reporting.

### Development

If you want to hack away feel free! PR's are accepted. Some things to keep in mind. If you want to change a language definition you need to update `languages.json` and then run `go generate` which will convert it into the `processor/constants.go` file.

For all other changes ensure you run all tests before submitting. You can do so using `go test ./...`. However, for maximum coverage please run `test-all.sh` which will run `gofmt`, unit tests, race detector and then all of the integration tests. All of those must pass to ensure a stable release.

### API Support

The core part of `scc` which is the counting engine is exposed publicly to be integrated into other Go applications. See <https://github.com/pinpt/ripsrc> for an example of how to do this.

It also powers all of the code calculations displayed in <https://searchcode.com/> such as <https://searchcode.com/file/169350674/main.go/> making it one of the more used code counters in the world.

However as a quick start consider the following,

Note that you must pass in the number of bytes in the content in order to ensure it is counted!

```go
package main

import (
  "fmt"
  "io/ioutil"

  "github.com/boyter/scc/v3/processor"
)

type statsProcessor struct{}

func (p *statsProcessor) ProcessLine(job *processor.FileJob, currentLine int64, lineType processor.LineType) bool {
  switch lineType {
  case processor.LINE_BLANK:
    fmt.Println(currentLine, "lineType", "BLANK")
  case processor.LINE_CODE:
    fmt.Println(currentLine, "lineType", "CODE")
  case processor.LINE_COMMENT:
    fmt.Println(currentLine, "lineType", "COMMENT")
  }
  return true
}

func main() {
  bts, _ := ioutil.ReadFile("somefile.go")  
  t := &statsProcessor{}
  filejob := &processor.FileJob{
    Filename: "test.go",
    Language: "Go",
    Content:  bts,
    Callback: t,
    Bytes:    int64(len(bts)),
  }  
  processor.ProcessConstants() // Required to load the language information and need only be done once
  processor.CountStats(filejob)
}
```

### Adding/Modifying Languages

To add or modify a language you will need to edit the `languages.json` file in the root of the project, and then run `go generate` to build it into the application. You can then `go install` or `go build` as normal to produce the binary with your modifications.

### Issues

Its possible that you may see the counts vary between runs. This usually means one of two things. Either something is changing or locking the files under scc, or that you are hitting ulimit restrictions. To change the ulimit see the following links.

- <https://superuser.com/questions/261023/how-to-change-default-ulimit-values-in-mac-os-x-10-6#306555>
- <https://unix.stackexchange.com/questions/108174/how-to-persistently-control-maximum-system-resource-consumption-on-mac/221988#221988>
- <https://access.redhat.com/solutions/61334>
- <https://serverfault.com/questions/356962/where-are-the-default-ulimit-values-set-linux-centos>
- <https://www.tecmint.com/increase-set-open-file-limits-in-linux/>

To help identify this issue run scc like so `scc -v .` and look for the message `too many open files` in the output. If it is there you can rectify it by setting your ulimit to a higher value.

### Low Memory

If you are running `scc` in a low memory environment < 512 MB of RAM you may need to set `--file-gc-count` to a lower value such as `0` to force the garbage collector to be on at all times.

A sign that this is required will be `scc` crashing with panic errors.

### Tests

scc is pretty well tested with many unit, integration and benchmarks to ensure that it is fast and complete.

### Package

Packaging as of version v3.1.0 is done through <https://goreleaser.com/>

### Containers

Note if you plan to run `scc` in Alpine containers you will need to build with CGO_ENABLED=0.

See the below Dockerfile as an example on how to achieve this based on this issue <https://github.com/boyter/scc/issues/208>

```Dockerfile
FROM golang as scc-get

ENV GOOS=linux \
GOARCH=amd64 \
CGO_ENABLED=0

ARG VERSION
RUN git clone --branch $VERSION --depth 1 https://github.com/boyter/scc
WORKDIR /go/scc
RUN go build -ldflags="-s -w"

FROM alpine
COPY --from=scc-get /go/scc/scc /bin/
ENTRYPOINT ["scc"]
```

### Badges (beta)

You can use `scc` to provide badges on your github/bitbucket/gitlab/sr.ht open repositories. For example, [![Scc Count Badge](https://sloc.xyz/github/boyter/scc/)](https://github.com/boyter/scc/)
 The format to do so is,

<https://sloc.xyz/PROVIDER/USER/REPO>

An example of the badge for `scc` is included below, and is used on this page.

```Markdown
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/)](https://github.com/boyter/scc/)
```

By default the badge will show the repo's lines count. You can also specify for it to show a different category, by using the `?category=` query string.

Valid values include `code, blanks, lines, comments, cocomo, effort` and examples of the appearance are included below.

[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=code)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=blanks)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=lines)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=comments)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=cocomo)](https://github.com/boyter/scc/)
[![Scc Count Badge](https://sloc.xyz/github/boyter/scc/?category=effort)](https://github.com/boyter/scc/)

For `cocomo` you can also set the `avg-wage` value similar to `scc` itself. For example,

<https://sloc.xyz/github/boyter/scc/?category=cocomo&avg-wage=1>
<https://sloc.xyz/github/boyter/scc/?category=cocomo&avg-wage=100000>

Note that the avg-wage value must be a positive integer otherwise it will revert back to the default value of 56286.

You can also configure the look and feel of the bad using the following parameters,

- ?lower=true will lower the title text, so "Total lines" would be "total lines"
he below can control the colours of shadows, fonts and badges
- ?font-color=fff
- ?font-shadow-color=010101
- ?top-shadow-accent-color=bbb
- ?title-bg-color=555
- ?badge-bg-color=4c1

An example of using some of these parameters to produce an admittedly ugly result

[![Scc Count Badge](https://sloc.xyz/github/boyter/scc?font-color=ff0000&badge-bg-color=0000ff&lower=true)](https://github.com/boyter/scc/)

*NB* it may not work for VERY large repositories (has been tested on Apache hadoop/spark without issue).

You can find the source code for badges in the repository at <https://github.com/boyter/scc/blob/master/cmd/badges/main.go>

#### A example for each supported provider

- Github - <https://sloc.xyz/github/boyter/scc/>
- sr.ht - <https://sloc.xyz/sr.ht/~nektro/magnolia-desktop/>
- Bitbucket - <https://sloc.xyz/bitbucket/boyter/decodingcaptchas>
- Gitlab - <https://sloc.xyz/gitlab/esr/loccount>

### Languages

List of supported languages. The master version of `scc` supports 322 languages at last count. Note that this is always assumed that you built from master, and it might trail behind what is actually supported. To see what your version of `scc` supports run `scc --languages`

[Click here to view all languages supported by master](LANGUAGES.md)

### Citation

Please use the following bibtex entry to cite scc in a publication:

<pre>
@software{scc,
  author       = {Ben Boyter},
  title        = {scc: v3.5.0},
  month        = ...,
  year         = ...,
  publisher    = {...},
  version      = {v3.5.0},
  doi          = {...},
  url          = {...}
}
</pre>

You may need to check the release page https://github.com/boyter/scc/releases to find the correct year and month for the release you are using.

### Release Checklist

- Update version
- Push code with release number
- Tag off
- Release via goreleaser
- Update dockerfile
