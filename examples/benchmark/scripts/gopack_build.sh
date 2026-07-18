#!/usr/bin/env bash
# Build every example into a gopack bundle, record the build time and bundle
# size, and verify each bundle runs correctly with no system Python on the PATH.
#
# The one-time CPython download is not part of the reported build time: run.sh
# (or setup.sh) warms the runtime cache first, so these numbers reflect a repeat
# build, which is the normal case.

source "$(dirname "${BASH_SOURCE[0]}")/config.sh"
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

build_csv="$RAW_DIR/gopack_build.csv"
size_csv="$RAW_DIR/gopack_size.csv"
: >"$build_csv"; echo "label,wall_seconds,max_rss_kb,exit_code" >"$build_csv"
: >"$size_csv"; echo "name,size_mb" >"$size_csv"
verify_log="$RAW_DIR/gopack_verify.txt"; : >"$verify_log"

if [ -z "$GOPACK_BIN" ] || [ ! -x "$GOPACK_BIN" ]; then
  echo "gopack binary not found; build it: (cd $REPO_ROOT && go build -o gopack .)" >&2
  exit 1
fi

for entry in "${EXAMPLES[@]}"; do
  IFS='|' read -r name subdir script args <<<"$entry"
  appdir="$EXAMPLES_DIR/$subdir"
  out="$BUNDLES_DIR/gopack/$name"
  req="$appdir/requirements.txt"

  section "gopack build: $name"
  reqflag=()
  [ -f "$req" ] && reqflag=(-r "$req")

  # Time the build itself.
  time_run "$build_csv" "build,$name" -- \
    "$GOPACK_BIN" build "$appdir" "${reqflag[@]}" --entry "$script" -o "$out"

  if [ ! -f "$out" ]; then
    echo "  BUILD FAILED for $name" | tee -a "$verify_log" >&2
    printf '%s,NA\n' "$name" >>"$size_csv"
    continue
  fi
  size="$(file_size_mb "$out")"
  printf '%s,%s\n' "$name" "$size" >>"$size_csv"
  echo "  built $name (${size} MB)" >&2

  # Verify it runs with no system Python.
  echo "  verifying $name in a stripped environment (PATH=/nonexistent)" >&2
  # shellcheck disable=SC2086
  if run_stripped -- "$out" $args >>"$verify_log" 2>&1; then
    echo "  verify OK: $name" | tee -a "$verify_log" >&2
  else
    rc=$?
    echo "  verify FAILED ($rc): $name" | tee -a "$verify_log" >&2
  fi
done

echo "" >&2
echo "gopack build results in $build_csv, sizes in $size_csv" >&2
