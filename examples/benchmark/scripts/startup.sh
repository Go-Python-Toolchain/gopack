#!/usr/bin/env bash
# Measure startup latency of the built bundles: the first (cold) launch and the
# median of subsequent (warm) launches, running the example's deterministic
# command each time.
#
# gopack extracts its payload once to a content-addressed cache, so its warm
# launches skip extraction. To measure a true cold launch the extraction cache
# is cleared first (the downloaded runtimes are kept, since those are a
# build-time cost, not a launch cost). PyInstaller onefile re-extracts to a temp
# directory on every launch, so its cold and warm figures are close by design;
# that difference is the point of the comparison.

source "$(dirname "${BASH_SOURCE[0]}")/config.sh"
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

csv="$RAW_DIR/startup.csv"
: >"$csv"; echo "label,wall_seconds,max_rss_kb,exit_code" >"$csv"

clear_gopack_extraction() {
  # Remove extracted bundles but keep the downloaded runtimes.
  find "$GOPACK_CACHE" -mindepth 1 -maxdepth 1 ! -name runtimes -exec rm -rf {} + 2>/dev/null || true
}

for entry in "${EXAMPLES[@]}"; do
  IFS='|' read -r name subdir script args <<<"$entry"
  # shellcheck disable=SC2206
  argv=($args)

  gp="$BUNDLES_DIR/gopack/$name"
  if [ -x "$gp" ]; then
    section "startup: gopack $name"
    clear_gopack_extraction
    time_run "$csv" "gopack,cold,$name" -- \
      env -i PATH=/nonexistent HOME="$HOME" GOPACK_CACHE="$GOPACK_CACHE" "$gp" "${argv[@]}"
    for _ in $(seq 1 "$STARTUP_RUNS"); do
      time_run "$csv" "gopack,warm,$name" -- \
        env -i PATH=/nonexistent HOME="$HOME" GOPACK_CACHE="$GOPACK_CACHE" "$gp" "${argv[@]}"
    done
  fi

  pi="$BUNDLES_DIR/pyinstaller/$name"
  if [ -x "$pi" ]; then
    section "startup: pyinstaller $name"
    time_run "$csv" "pyinstaller,cold,$name" -- \
      env -i PATH=/nonexistent HOME="$HOME" "$pi" "${argv[@]}"
    for _ in $(seq 1 "$STARTUP_RUNS"); do
      time_run "$csv" "pyinstaller,warm,$name" -- \
        env -i PATH=/nonexistent HOME="$HOME" "$pi" "${argv[@]}"
    done
  fi
done

echo "" >&2
echo "startup samples written to $csv" >&2
