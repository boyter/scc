# Closing the gap on llvm-project

Where scc's time went on `llvm-project`, what was done about it, and what was
measured and abandoned. Follows on from the syscall table in boyter/scc#769.

Every number here was re-measured on a quiet machine after an independent audit
found several of the first set wrong. Where a figure is inside the noise it says
so rather than being quoted.

## The comparison was not comparing the same work

The headline was scc 331ms against mezura 143ms on llvm-project, 2.3x. Both
reproduce. What they do not do is describe the same job.

| | files | bytes |
|---|---:|---:|
| scc | 159,726 | 1,851 MiB |
| mezura | 94,837 | 680 MiB |

mezura recognises 84 languages and **LLVM IR and MLIR are not among them**, so it
returns no row for either. `.ll` alone is 720 MiB across 47,714 files, 39% of
everything scc reads here and none of what mezura reads. With MLIR that is 750
MiB, **64% of the 1,171 MiB difference** — dominant, but not all of it. The rest
is Plain Text (46 MiB), JSON (26.5), Markdown (12.9), YAML (7.6) and others,
plus mezura excluding 6,589 generated files and honouring `.gitignore`. The two
tools disagree about what is worth counting, and that disagreement is most of the
gap.

(Both tools label MiB as "MB". The figures above are MiB. An earlier draft of
this document also claimed mezura does not recognise TableGen: **scc does not
recognise TableGen either** — `languages.json` has no entry, and the 1,815 `.td`
files of this tree count as zero bytes in both tools — so it explains none of the
difference.)

Normalising matters, and so does saying which clock:

| per MiB | scc before | scc after | mezura |
|---|---:|---:|---:|
| **user CPU** | 4.02 ms | 1.82 ms | 2.54 ms |
| **wall** | 0.173 ms | 0.113 ms | 0.206 ms |

Per user CPU scc went from **1.58x slower** than mezura to **1.40x faster**. Per
wall clock scc was **already faster** per byte before any of this (0.84x), because
it parallelises better; it is now 1.82x faster. The user-CPU pair is the honest
one for "how much work does this tool do per byte"; the wall pair is the honest
one for "how long do I wait". Quoting either without naming the clock is how the
first draft of this document ended up with a figure that did not reconcile.

## Where the time was

CPU profile of `scc -c --exp-per-language-counters llvm-project`, 8.74s sampled:

| cost | share |
|---|---:|
| `countLoopGeneric` | 51.0% cumulative, **41.3% flat** |
| `DetermineLanguage` | 11.1%, of which regex 10.1% |
| `ReadFile` | 9.7% |
| `walkDirectoryRecursive` | 5.4% |

41% of all CPU was the generic loop's own instructions, about 3.3 ns a byte, with
its callees adding only another 10%. 59.1% of scc's bytes here had no specialised
counter and went through it.

## What was changed

### A line comment ends at its newline

`countLoopGeneric` switched on the state of every byte and had **no case for
`SComment` or `SCommentCode`**. Nothing in a line comment can move the state, so
every byte of one ran the whole loop body to do nothing.

Instrumented: 543,345,194 iterations for 1,147,253,434 bytes, 2.11 bytes an
iteration, and 89.4% of those iterations were in `SComment`, 4.1% more in
`SCommentCode`. Nine tenths of the loop ran to arrive where it started.

`IndexByte` to the newline. Five lines. Where the case sits matters more than what
it does: as an `else if` beside the existing blank-run skip it cost 2% on corpora
with few line comments; inside the switch it is free, the states being dense
enough for a jump table.

### Counters for LLVM IR and Assembly

Sizing candidates by uncovered bytes said LLVM IR and Assembly were worth about
the same. **What costs the generic loop is not bytes but stops.** LLVM IR declares
sixteen complexity checks and nine open with a letter common in the language, so a
naive stop set covers a quarter of every `.ll` file before any work is done;
anchored onto seven rarer bytes it is far less. Assembly has the same small token
set, a much lower stop rate and files that are 36% whitespace both loops already
skip a vector at a time: 1.14x, against LLVM IR's 3.94x.

(The stop-rate percentages quoted in that commit message were measured over a
32 MB sample — an alphabetical prefix, so mostly `llvm/test` — and do not hold
over the whole corpus. The ranking they were used to justify does hold; the
numbers themselves should be re-measured before being quoted anywhere.)

### Anchor the heuristics that are anchored

Only 3 of 366 languages carry heuristics, so detection across 366 languages was
never the cost. The cost is `.h` and `.m`: 19,731 ambiguous files here.

Not the bytes searched — `toCheck` is capped at 20,000 and averages 3,604. Not the
literal pre-check — 68ms for the corpus, and where it passes the regex matches 97%
of the time, because LLVM's headers really are C++. It is that a pattern opening
`^\s*` has no literal prefix, so the engine retries at all 3,604 byte offsets
instead of the ~120 line starts where it could begin. `std::\w+` runs in 1.3 us;
`^[ \t]*(try|constexpr)` takes 37.9 us.

Rewriting `(?m)^\s*REST` to `(?m)\A(?:REST)` and running it at the line starts the
plan already walks gives 12x to 26x per pattern, hit counts unchanged.

## Result

llvm-project, `GOMAXPROCS=1`, CPU seconds, interleaved with arms rotated, min of
9 on an otherwise idle machine:

| | CPU | vs base |
|---|---:|---:|
| base | 4.76 | 1.00x |
| comment skip | 3.15 | **1.51x** |
| LLVM IR + Assembly counters | 3.03 | **1.57x** |
| anchored heuristics | 4.32 | **1.10x** |
| **all three** | **2.59** | **1.84x** |

**They are not additive.** Individually they save 1.61 + 1.73 + 0.44 = 3.78s;
together they save 2.17s. **43% is double counted**, because the comment skip and
the LLVM IR counter compete for the same `.ll` bytes. The heuristic work goes the
other way and is worth more as the others land — 1.10x alone against 1.15x on top
of the other two, its fixed 0.44s being a larger share of a smaller total.

Full parallelism, and hyperfine:

| | before | after | |
|---|---:|---:|---|
| parallel wall | 0.360s | 0.240s | |
| parallel user | 7.39s | 3.42s | |
| hyperfine stock wall | 433.2 ms | 274.6 ms | 1.58x |
| hyperfine stock user | 11,049 ms | 5,571 ms | **1.98x** |
| hyperfine `-c --exp` wall | 342.7 ms | 227.3 ms | 1.51x |
| hyperfine `-c --exp` user | 7,456 ms | 3,406 ms | **2.19x** |

**It is an llvm-shaped win, not a general one.** On the Linux kernel the wall
clock does not move at all (1.00 ± 0.03), though there is a small sign-consistent
CPU win of 1-2% underneath it — lower in 11 of 11 paired single-thread reps — that
never surfaces because the run is not CPU-bound at 32 threads. lucene 1.00x.
kubernetes 1.02 ± 0.02, ruby 1.03 ± 0.04 and cpython 1.01 ± 0.02 are all inside
the noise band and should not be quoted as wins; an earlier draft claimed 1.05x
for cpython and ruby, measured with `/usr/bin/time`, whose 10 ms granularity on a
200 ms run makes 5% exactly one tick.

Output byte identical on llvm-project, linux, kubernetes, cpython, ruby and
lucene, summary and per-file, with and without the counters: 296,599 files.

## Measured and not taken

**Syscalls.** `os.Open` on a directory registers with the runtime poller, which
always fails on a directory, at 4 `fcntl` + 1 `epoll_ctl` per directory — 67,494
and 16,873 here. Raw `getdents64` removes 84,195 syscalls and makes the walk 6.1%
faster in isolation, and changes the whole run by 0.2%. Replacing the per-file
`os.Lstat` (980 ns) with `fstat` on the descriptor already open (73 ns) is worth
~145ms and a reproducible 9% of system time, and no wall time. Neither touches the
gap, which is user time in the counting loop.

There is no duplicated stat: the walker takes `IsDir()` from `getdents64`'s
`d_type` and never stats a regular file.

**Hoisting the loop invariants.** The `byteType != nil` test, the `Cognitive`
global and the bounds check on `content[index]` are all really executed per byte —
the assembly confirms none are hoisted. Removing them made it **slower**, 0.8-3.5%
on three corpora: the function already spills hard, and the changed live ranges
cost more than the deleted instructions saved. The 41% flat is switch dispatch,
spill traffic and the size of the loop body, not branch overhead.

**Parallelism.** Already 81% efficient. The largest counted file is 24.59 MB
against a perfect-balance share of 57.85 MB a core, and the last worker finishes
2.2-3.6 ms after the median. Batching the per-file channel sends is worth 3% CPU
and 15% fewer futexes and no wall time.

## The read-syscall change was reverted

`4c83231` made scc read a file in one syscall where the stat could be trusted,
halving `read` from 179,670 to 90,248 on the Linux kernel — the count mezura's
author reported in #769. It is gone.

It bought no measurable time. Interleaved, 25 reps, `GOMAXPROCS=1`, warm caches,
it moved neither user nor system time out of the noise on the kernel or on
llvm-project. And its rule was unsound: a first read coming up short while meeting
the stat's promise is not proof of the end of a file, because the first read can
land at or past a size the file has since grown beyond. The test that shipped with
it caught this, failing 8 runs in 30, and the failure was misread as unfixable and
the assertion weakened — which was wrong twice over, since an `fstat` on the open
descriptor distinguishes the cases in twenty lines, and the weakened assertion
could no longer fail at all.

The `fstat` fix works and was measured: 972 ns a file, against 925 ns for simply
reverting and 830 ns for the unsound version. **The correct fix is slower than not
optimising**, because an `fstat` costs more than the end-of-file `read` it
replaces. So the revert is both the correct and the fast answer, and the syscall
count in #769 is best answered with "it halves, it buys nothing, and the way we
halved it was wrong".

## Notes on measuring this

**A/B between two separately built binaries is unreliable at about 10% here.** A
control arm doing strictly more work than its baseline measured 10.4% *less* user
CPU. Anything worth under 10% must be measured with both arms in one binary,
selected by an environment variable, so code layout is held still. The anchored
heuristics read 1.09x that way and 1.05x across two binaries — the 4% gap is
layout alone, which is what `scripts/tune-alignment.sh` exists for.

32-thread wall clock is the wrong instrument for anything small: at 0.3s wall the
sign of an effect can flip between corpora. `GOMAXPROCS=1` with user+sys makes
wall approximately user+sys and collapses the scheduler noise. And `/usr/bin/time`
resolves 10 ms, which is 5% of a 200 ms run — it cannot measure the small corpora
at all.

**A corpus cannot find a bug no file in it contains.** 296,599 files agreeing
proved less here than three hand-built files disagreeing: the anchored-heuristics
change silently reclassified headers containing a form feed, a lone carriage
return or a vertical tab before a keyword, and no file in six real corpora carries
that shape. The property test that should have caught it passed only because its
piece list omitted those three bytes.
