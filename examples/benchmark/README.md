# gopack benchmark

Reproduce the numbers behind gopack's bundles on your own machine, and compare
gopack against PyInstaller, the closest single-file peer. This harness bundles
every example project in this repository, measures build time, bundle size, and
startup latency, and checks that each bundle actually runs with no Python
installed.

The published figures live in [`docs/benchmarks.md`](../../docs/benchmarks.md).
This directory is the harness that produces them.

## What it measures

For each example project, both with gopack and with PyInstaller:

1. **Build time** and **bundle size**: how long it takes to produce the single
   executable, and how big it is.
2. **Startup latency**: the first (cold) launch and the median of subsequent
   (warm) launches, running the example's deterministic command.
3. **It runs at all**: every bundle is executed with the environment stripped to
   `PATH=/nonexistent`, so there is no system Python to fall back on. A bundle
   that does not run is recorded as a failure.

The examples are real, multi-file applications: a scikit-learn pipeline, a
FastAPI service with Jinja templates and static files, a full Django project
with the admin site and management commands, a pandas and NumPy data pipeline,
and a minimal quickstart app.

## The comparison, honestly

gopack and PyInstaller both emit one self-extracting executable, but they differ
in two ways that the tables make visible:

- **Configuration.** gopack bundles exactly what pip installed, with no
  per-framework flags. PyInstaller analyzes imports statically, so each
  framework needs help: data files (templates, static, migrations) listed by
  hand and dynamically imported modules collected explicitly. The flags each
  example needs are recorded in `work/raw/pyinstaller_flags.txt`.
- **Startup.** gopack extracts its payload once to a content-addressed cache, so
  warm launches skip extraction. PyInstaller onefile re-extracts to a temp
  directory on every launch. The cold and warm columns show this.

Neither tool is strictly better; they make different trade-offs. The harness
reports what each costs so you can decide.

## Requirements

- `python3` (runs the measurement wrapper and the aggregator, standard library
  only)
- `go` (to build gopack, if a `gopack` binary is not already present)
- `git`, and a network connection (the first gopack build downloads a
  relocatable CPython; PyInstaller pulls the example dependencies)
- Optional: `GITHUB_TOKEN` (or `gh` logged in). Building several bundles in a row
  resolves the CPython release from the GitHub API each time; a token lifts the
  anonymous 60-per-hour rate limit. The harness picks up `gh auth token`
  automatically if present.

## Run it

```
# From this directory (examples/benchmark).

# 1. One time: build gopack, warm the runtime cache, install PyInstaller and the
#    example dependencies (~several minutes, downloads a few hundred MB).
scripts/setup.sh

# 2. Build every bundle, measure, and write work/raw/results.md.
scripts/run.sh
```

Or run a single stage:

```
scripts/machine.sh            # record CPU / OS / tool versions
scripts/gopack_build.sh       # bundle each example with gopack, verify each runs
scripts/pyinstaller_build.sh  # bundle each example with PyInstaller, verify each runs
scripts/startup.sh            # cold and warm startup for every built bundle
python3 scripts/aggregate.py  # rebuild results.md from the raw CSVs
```

Results, raw logs, built bundles, and the machine description land in `work/`:

- `raw/results.md` - the assembled tables
- `raw/*.csv` - one row per sample, so you can re-aggregate
- `raw/gopack_verify.txt`, `raw/pyinstaller_verify.txt` - the output of each
  bundle running in a stripped environment
- `raw/pyinstaller_flags.txt` - the flags PyInstaller needed per example
- `raw/machine.txt` - CPU, memory, OS, and tool versions
- `bundles/` - the built executables

## Tuning

- `GOPACK_BENCH_RUNS` - warm startup repetitions (default 5, median reported)
- `GOPACK_BENCH_WORK` - where bundles, tools, caches, and raw output live
  (default `./work`)
- `GOPACK_BIN`, `PYI_PYTHON` - point at a specific gopack binary or PyInstaller
  Python

## Contribute a machine

Build and startup times depend on the hardware. If you run this on a Ryzen
desktop, an Apple M-series laptop, or a low-end machine, the `machine.txt` and
`results.md` from your run are what a cross-hardware table needs. Open an issue
or a pull request against the gopack repo with both files.

## Pinned versions

The example dependencies are pinned in each project's `requirements.txt`. The
competitor is pinned here:

| Piece | Version |
| --- | --- |
| PyInstaller | 6.9.0 |
| CPython runtime | 3.12 (python-build-standalone) |
