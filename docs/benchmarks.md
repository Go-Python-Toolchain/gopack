# Benchmarks

This document records how gopack performs on real applications, and how it
compares to PyInstaller, the closest single-file peer. The numbers cover the
three things that matter when you ship a Python app as one executable: how long
it takes to build, how big the result is, and how fast it starts. Every figure
is reproducible with the harness in
[`examples/benchmark`](../examples/benchmark), which bundles each example both
ways and measures them on your own machine.

## What was measured

- Subjects: the example projects in this repository, all real multi-file
  applications except the first.
  - `basic`: a minimal app (one dependency).
  - `data-report`: a pandas and NumPy data pipeline.
  - `ml-iris`: a scikit-learn training and prediction pipeline (NumPy, SciPy).
  - `fastapi`: a FastAPI service with Jinja templates and static files.
  - `django`: a full Django project with the admin site, migrations, and
    management commands.
- Tools: gopack (this build) and PyInstaller 6.9.0, both producing a single
  self-extracting executable.
- Every bundle was run with the environment reduced to `PATH=/nonexistent`, so
  there was no system Python to fall back on. A bundle that did not run was
  recorded as a failure.

## Environment

- CPU: 12th Gen Intel Core i9-12900H (20 threads)
- Memory: 31.0 GB
- OS/arch: linux/amd64 (kernel 6.17)
- gopack runtime: stripped CPython 3.12.13 from python-build-standalone
- PyInstaller: 6.9.0 on CPython 3.12

Absolute numbers depend on the machine. The shape of the comparison, and the
architectural difference it exposes, hold across hardware.

## How to reproduce

```
cd examples/benchmark
scripts/setup.sh      # build gopack, warm the runtime cache, install PyInstaller and deps
scripts/run.sh        # build every example both ways, measure, write work/raw/results.md
```

## An earlier version of this page

A previous version of this document reported gopack bundles at 256 to 340 MB and
called size the tool's main weakness. That was accurate then, and the work since
has changed it: gopack now ships a stripped CPython runtime, so the numbers below
are between a half and two thirds smaller. The old figures are kept in the size
narrative for comparison rather than quietly dropped.

## Every bundle runs

All five examples were bundled with gopack and all five ran correctly with no
Python installed, from the scikit-learn pipeline to the full Django project.

PyInstaller bundled all five too, but its `data-report` bundle failed to run: it
raised a NumPy import error that PyInstaller's onefile packaging of NumPy is
prone to, the kind that needs an extra hook or flag to resolve. gopack's
`data-report` bundle ran without any such help. That row is marked below and left
out of the head-to-head averages, but it is itself part of the comparison: what
runs out of the box is a result, not a footnote.

## Build time and bundle size

Build time is a repeat build, with the CPython runtime already cached (the first
gopack build downloads it once). Lower is better on both.

| Example | gopack build | gopack size | PyInstaller build | PyInstaller size |
| --- | ---: | ---: | ---: | ---: |
| basic | 8.3 s | 81.9 MB | 9.7 s | 7.1 MB |
| data-report | 23.4 s | 126.8 MB | 59.8 s | 64.2 MB (did not run) |
| ml-iris | 30.6 s | 164.2 MB | 110.8 s | 102.8 MB |
| fastapi | 22.8 s | 100.5 MB | 35.6 s | 23.7 MB |
| django | 13.1 s | 93.0 MB | 55.8 s | 36.9 MB |

Two things changed from the earlier measurements, and one did not.

**Size** is much closer than it was. gopack still ships the whole standard
library and everything pip installed, so it is larger than PyInstaller, which
traces the import graph and packs only what it finds. But the stripped runtime
took gopack from a flat 250-to-340 MB band down to 82 to 164 MB, and the gap now
narrows sharply as an app grows real dependencies: on the scikit-learn pipeline
it is 164 MB against 103, a factor of 1.6 rather than the 3-plus it was. For a
trivial app PyInstaller is still dramatically smaller (7 MB against 82), because
there is almost nothing for gopack to leave out.

**Build time** now favors gopack on every non-trivial app, often by a wide
margin: 23 seconds against 60 on `data-report`, 31 against 111 on `ml-iris`, 13
against 56 on `django`. PyInstaller's static import analysis is thorough and
slow; gopack stages what pip installed and compresses it, which is steadier and,
on a real dependency set, quicker.

**The shape of the trade** is unchanged: gopack includes everything and needs no
per-framework configuration; PyInstaller includes only what it can prove is
imported and needs help to find the rest. That is the subject of "the cost of
configuration" below.

## Startup latency

The first launch (cold) and the median of subsequent launches (warm), each
running the example's deterministic command. Lower is better.

| Example | gopack cold | gopack warm | PyInstaller cold | PyInstaller warm |
| --- | ---: | ---: | ---: | ---: |
| basic | 2477 ms | 37 ms | 199 ms | 223 ms |
| data-report | 4925 ms | 514 ms | 1397 ms | 1411 ms* |
| ml-iris | 7054 ms | 1655 ms | 4024 ms | 4010 ms |
| fastapi | 4198 ms | 636 ms | 1201 ms | 1172 ms |
| django | 3644 ms | 328 ms | 1353 ms | 1400 ms |

\* PyInstaller's `data-report` figures are the time to reach the NumPy import
error, not a successful run.

This table is where the two designs diverge, and neither wins outright.

gopack extracts its payload once to a content-addressed cache. The first launch
pays for writing the bundle to disk, so it is the slow one, a few seconds. Every
launch after that skips extraction and, since G2, no longer rehashes the payload
either, so the warm column is a different order of magnitude from the cold one:
the `basic` app starts in 37 ms warm.

PyInstaller onefile re-extracts its payload to a temporary directory on every
launch, so its cold and warm figures are nearly identical: there is no warm path
to speak of. For a real application that cost shows on every run. On the
scikit-learn pipeline PyInstaller pays about 4 seconds each launch while gopack,
after its first, runs in about 1.7 seconds; on Django it is about 1.4 seconds
every time against gopack's 0.3. gopack's warm launches win on `ml-iris`,
`fastapi`, and `django`; PyInstaller wins on the very first launch of any bundle
and, warm, roughly ties on the trivial app where it does not really re-extract
much either way.

The takeaway is unchanged: PyInstaller is better when a bundle is launched rarely
or is very small; gopack is better for a real application launched more than once,
which is the common case for a service or a tool.

## The cost of configuration

The tables do not capture one real difference: what it took to get a working
bundle. gopack bundled all five examples with the same command shape and no
per-framework flags:

```
gopack build ./<app> -r ./<app>/requirements.txt --entry <script> -o <name>
```

It bundles exactly what pip installed, so a framework's data files and
dynamically imported modules come along automatically.

PyInstaller analyzes imports statically, so each framework needed help:

| Example | PyInstaller flags beyond `--onefile` |
| --- | --- |
| basic | none |
| data-report | `--collect-submodules salesreport` |
| ml-iris | `--collect-all sklearn --collect-all scipy --collect-all numpy` |
| fastapi | `--collect-all fastapi --collect-submodules starlette --collect-submodules uvicorn --add-data templates --add-data static` |
| django | `--collect-all django --collect-submodules notes --collect-submodules config --add-data templates --add-data static --add-data migrations` |

NumPy is the clearest example of the cost, and the reason the `data-report` row
above did not run. Its bundle was built with the app's own submodules collected
(`--collect-submodules salesreport`), which is enough for the application code
but not for NumPy: the resulting bundle raised NumPy's "you may be trying to
import from the source tree" error at launch. Making it run needs NumPy collected
explicitly as well, the same `--collect-all numpy` the scikit-learn pipeline
needed, which is exactly why `ml-iris` carries `--collect-all sklearn
--collect-all scipy --collect-all numpy`. The FastAPI and Django bundles needed
their templates, static files, and migrations listed by hand, because a static
import scan cannot see files loaded by path at run time. gopack needed none of
this, because it does not guess what to include: it ships what pip installed, so
the same command that bundled the trivial app also bundled the NumPy pipeline,
and both ran.

## Threats to validity

These measurements were performed on Linux x86_64 with an Intel i9-12900H.
Absolute timings will vary across operating systems, CPUs, storage devices
(extraction and re-extraction are disk bound, so a slow or fast drive moves the
startup numbers directly), package-index and network latency (the first build
downloads a CPython runtime), and the versions of the tools compared against.
The relative standing is more stable than the absolute numbers, but it too can
shift with tool versions. The benchmark harness is published so readers can
reproduce the measurements on their own hardware and see what holds.

## Summary

gopack and PyInstaller make opposite trades. PyInstaller produces a small bundle
by including only what it can prove is imported, at the cost of per-framework
configuration, a slower build, and a re-extraction on every launch. gopack
produces a larger bundle by including the whole runtime and everything pip
installed, which needs no configuration, runs correctly the first time, builds
faster on a real dependency set, and, after a slow first launch, starts quickly
on every launch after.

Size was gopack's clear weak spot and is now a modest one: the stripped runtime
brought the bundles down to tens to low hundreds of megabytes, close enough that
on a real application the difference is a factor rather than an order of
magnitude. For shipping a real application that runs more than once, on a machine
you do not control, gopack's trade is usually the one you want.
