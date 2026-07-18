# Shared helpers for the gopack benchmark harness.

# Absolute path so callers that cd elsewhere (the PyInstaller build runs from the
# app directory) can still find the measurement wrapper.
MEASURE_PY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/measure.py"

# time_run OUT_CSV LABEL -- CMD...
# Runs CMD once under measure.py and appends a CSV row:
#   "label",wall_seconds,max_rss_kb,exit_code
time_run() {
  local out_csv="$1"; shift
  local label="$1"; shift
  [ "$1" = "--" ] && shift
  local line
  line="$(python3 "$MEASURE_PY" "$@" 2>/dev/null || echo "NA,NA,127")"
  printf '"%s",%s\n' "$label" "$line" >>"$out_csv"
}

# file_size_mb PATH -> size in MB with one decimal.
file_size_mb() {
  local bytes
  bytes="$(stat -c %s "$1" 2>/dev/null || stat -f %z "$1" 2>/dev/null || echo 0)"
  awk -v b="$bytes" 'BEGIN { printf "%.1f", b/1024/1024 }'
}

# run_stripped -- CMD... : run a command with the environment reduced so there is
# no system Python to fall back on, proving a bundle is self-contained. HOME and
# the gopack cache are preserved so extraction has somewhere to write.
run_stripped() {
  [ "$1" = "--" ] && shift
  env -i PATH=/nonexistent HOME="$HOME" GOPACK_CACHE="$GOPACK_CACHE" "$@"
}

section() { echo "" >&2; echo "== $* ==" >&2; }
