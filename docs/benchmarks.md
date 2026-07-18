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
- Python for the runtime and for PyInstaller: CPython 3.12

Absolute numbers depend on the machine. The shape of the comparison, and the
architectural difference it exposes, hold across hardware.

## How to reproduce

```
cd examples/benchmark
scripts/setup.sh      # build gopack, warm the runtime cache, install PyInstaller and deps
scripts/run.sh        # build every example both ways, measure, write work/raw/results.md
```

## Every bundle runs

All five examples were bundled with gopack and all five ran correctly with no
Python installed, from the scikit-learn pipeline to the full Django project
answering `migrate`, `seed`, `check`, and `runserver`. The same five were
bundled with PyInstaller and, once given the flags each framework needs (see
"the cost of configuration" below), all five ran as well. Correctness first: the
tables below only compare bundles that actually work.

## Build time and bundle size

Build time is a repeat build, with the CPython runtime already cached (the first
gopack build downloads it once). Lower is better on both.

| Example | gopack build | gopack size | PyInstaller build | PyInstaller size |
| --- | ---: | ---: | ---: | ---: |
| basic | 21.8 s | 256.2 MB | 4.6 s | 7.1 MB |
| data-report | 28.6 s | 301.2 MB | 31.3 s | 64.9 MB |
| ml-iris | 36.0 s | 338.6 MB | 63.7 s | 104.7 MB |
| fastapi | 32.2 s | 274.9 MB | 18.7 s | 23.7 MB |
| django | 24.5 s | 267.3 MB | 28.9 s | 36.9 MB |

PyInstaller bundles are smaller, and for a tiny app dramatically so, because it
walks the import graph and packs only the modules it finds. gopack ships the
whole CPython install and everything pip installed, so its bundles sit in a
narrow band around 250 to 340 MB regardless of the app. That size is gopack's
main weak point today, and the path to shrinking it is in the
[roadmap](roadmap.md). Note that as an application grows, the gap narrows: for
the scikit-learn pipeline PyInstaller is already over 100 MB. Build times are in
the same order of magnitude, and gopack's is steady because most of it is
staging and compressing a similar payload each time.

## Startup latency

The first launch (cold) and the median of subsequent launches (warm), each
running the example's deterministic command. Lower is better.

| Example | gopack cold | gopack warm | PyInstaller cold | PyInstaller warm |
| --- | ---: | ---: | ---: | ---: |
| basic | 3811 ms | 166 ms | 100 ms | 100 ms |
| data-report | 6289 ms | 544 ms | 983 ms | 1004 ms |
| ml-iris | 8240 ms | 1139 ms | 2299 ms | 2239 ms |
| fastapi | 6118 ms | 530 ms | 634 ms | 624 ms |
| django | 5149 ms | 336 ms | 744 ms | 747 ms |

This table is where the two designs diverge, and neither wins outright.

gopack extracts its payload once to a content-addressed cache. The first launch
pays for writing hundreds of megabytes to disk, so it is slow, several seconds.
Every launch after that skips extraction and runs from the cache, which is why
the warm column is a different order of magnitude from the cold one.

PyInstaller onefile re-extracts its payload to a temporary directory on every
launch, so its cold and warm figures are nearly identical: there is no warm path
to speak of. For the tiny `basic` app that is very fast, faster than gopack even
warm, because there is almost nothing to extract. But the cost scales with the
bundle, so for the real applications the per-launch extraction shows: on the
scikit-learn pipeline PyInstaller pays about 2.2 seconds every single run, while
gopack pays it once and then runs in about 1.1 seconds. For a tool you launch
repeatedly, gopack's warm launches win on `data-report`, `ml-iris`, `fastapi`,
and `django`; PyInstaller wins on the trivial app and on the very first launch
of any of them.

The takeaway: PyInstaller is better when a bundle is launched rarely or is very
small; gopack is better for a real application launched more than once, which is
the common case for a service or a tool.

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

The scikit-learn pipeline is the clearest example of the cost. A first attempt
with `--collect-submodules sklearn` built successfully but failed at run time
with `ModuleNotFoundError: No module named 'scipy._cyutility'`, a compiled
internal PyInstaller had not collected. It worked only after collecting all of
scikit-learn, SciPy, and NumPy. The FastAPI and Django bundles needed their
templates, static files, and migrations listed by hand, because a static import
scan cannot see files that are loaded by path at run time. gopack needed none of
this, because it does not guess what to include: it ships what pip installed.

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
configuration and a re-extraction on every launch. gopack produces a large
bundle by including the whole runtime and everything pip installed, which needs
no configuration, runs correctly the first time, and, after a slow first launch,
starts quickly on every launch after. For shipping a real application that runs
more than once, on a machine you do not control, gopack's trade is usually the
one you want. Its size is the honest weak spot, and shrinking it is the top item
on the roadmap.
