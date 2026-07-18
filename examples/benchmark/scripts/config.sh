# Shared configuration for the gopack benchmark harness.
#
# Every script sources this file. It defines the example projects to bundle, the
# directory layout, and the pinned competitor version, so a run on any machine
# measures the same thing. Machine details are captured at run time by
# machine.sh; nothing here is machine specific.

set -euo pipefail

# Root of the benchmark harness (the directory holding this scripts folder).
BENCH_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# The gopack repository root (examples/benchmark -> repo root).
REPO_ROOT="$(cd "$BENCH_ROOT/../.." && pwd)"
EXAMPLES_DIR="$REPO_ROOT/examples"

# Heavy, uncommitted artifacts: the built bundles, the installed competitor, the
# runtime caches, and the raw logs. This tree is git ignored.
WORK_DIR="${GOPACK_BENCH_WORK:-$BENCH_ROOT/work}"
BUNDLES_DIR="$WORK_DIR/bundles"
TOOLS_DIR="$WORK_DIR/tools"
RAW_DIR="${GOPACK_BENCH_RAW:-$WORK_DIR/raw}"
# gopack's build-time cache (downloaded CPython runtimes) and each bundle's
# run-time extraction cache both live here.
export GOPACK_CACHE="${GOPACK_CACHE:-$WORK_DIR/.gopack-cache}"

# Building several bundles in a row resolves the CPython runtime release from the
# GitHub API each time, which can trip the 60-per-hour anonymous rate limit. If a
# token is available, pass it through so the run stays reliable. gopack reads
# GITHUB_TOKEN / GH_TOKEN; users without one can still run, just more slowly if
# they hit the cap.
if [ -z "${GITHUB_TOKEN:-}" ] && [ -z "${GH_TOKEN:-}" ] && command -v gh >/dev/null 2>&1; then
  if tok="$(gh auth token 2>/dev/null)" && [ -n "$tok" ]; then
    export GITHUB_TOKEN="$tok"
  fi
fi

# The example projects to benchmark. Each entry is
#   name|subdir|entry|args
# where subdir is under examples/, entry is the script gopack runs, and args is
# the deterministic command used to time startup and to verify the bundle runs.
# Every project except basic is a real, multi-file application.
EXAMPLES=(
  "basic|basic|main.py|"
  "data-report|data-report|run.py|"
  "ml-iris|ml-iris|run.py|demo"
  "fastapi|fastapi-service|run.py|check"
  "django|django-notes|manage.py|check"
)

# Repetitions when timing startup. The reported figure is the median of the warm
# runs; the first run is reported separately as the cold (first launch) time.
STARTUP_RUNS="${GOPACK_BENCH_RUNS:-5}"

# Pinned competitor. PyInstaller is the closest peer: it also produces a single
# self-extracting executable. The comparison note in the docs explains the
# differences that matter (import analysis versus bundling what pip installed,
# and re-extract-every-run versus a content-addressed cache).
PYINSTALLER_VERSION="6.9.0"

# Tool entry points.
find_gopack() {
  if command -v gopack >/dev/null 2>&1 && [ -z "${GOPACK_PREFER_LOCAL:-}" ]; then
    command -v gopack; return
  fi
  for c in "$REPO_ROOT/gopack" "$REPO_ROOT/gopack.exe"; do
    [ -x "$c" ] && { echo "$c"; return; }
  done
  command -v gopack 2>/dev/null || echo ""
}
GOPACK_BIN="${GOPACK_BIN:-$(find_gopack)}"

mkdir -p "$WORK_DIR" "$BUNDLES_DIR/gopack" "$BUNDLES_DIR/pyinstaller" "$TOOLS_DIR" "$RAW_DIR"
