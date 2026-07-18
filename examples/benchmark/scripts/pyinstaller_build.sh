#!/usr/bin/env bash
# Build each example into a PyInstaller onefile executable, record the build time
# and size, and verify it runs with no system Python.
#
# PyInstaller is the closest peer to gopack: it also emits a single self-
# extracting executable. The difference this step makes visible is configuration.
# gopack bundles exactly what pip installed with no per-framework flags. With
# PyInstaller each framework needs help: data files must be listed by hand, and
# dynamically imported modules must be collected explicitly. The flags below are
# recorded per example so the cost of that configuration is part of the result.
#
# Requires a Python environment with the example dependencies and PyInstaller
# installed. setup.sh builds one under work/tools/pyi-venv; override with
# PYI_PYTHON.

source "$(dirname "${BASH_SOURCE[0]}")/config.sh"
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

PYI_PYTHON="${PYI_PYTHON:-$TOOLS_DIR/pyi-venv/bin/python}"
if [ ! -x "$PYI_PYTHON" ]; then
  # Fall back to the shared smoke venv if that is where PyInstaller was installed.
  if [ -x "$TOOLS_DIR/smoke-venv/bin/python" ]; then
    PYI_PYTHON="$TOOLS_DIR/smoke-venv/bin/python"
  fi
fi
if ! "$PYI_PYTHON" -c "import PyInstaller" 2>/dev/null; then
  echo "PyInstaller not available at $PYI_PYTHON; run setup.sh (or skip the PyInstaller comparison)" >&2
  exit 1
fi

build_csv="$RAW_DIR/pyinstaller_build.csv"
size_csv="$RAW_DIR/pyinstaller_size.csv"
flags_log="$RAW_DIR/pyinstaller_flags.txt"
verify_log="$RAW_DIR/pyinstaller_verify.txt"
: >"$build_csv"; echo "label,wall_seconds,max_rss_kb,exit_code" >"$build_csv"
: >"$size_csv"; echo "name,size_mb,status" >"$size_csv"
: >"$flags_log"; : >"$verify_log"

# set_extra_flags NAME APPDIR: populate the global FLAGS array with the
# PyInstaller flags a given example needs, beyond --onefile. An array is used
# rather than a string so that --add-data paths (which contain a space in this
# repository's path) survive as single arguments.
set_extra_flags() {
  local name="$1" appdir="$2"
  FLAGS=()
  case "$name" in
    fastapi)
      FLAGS=(--collect-all fastapi --collect-submodules starlette
             --collect-submodules uvicorn --collect-submodules app
             --add-data "$appdir/app/templates:app/templates"
             --add-data "$appdir/app/static:app/static")
      ;;
    django)
      FLAGS=(--collect-all django --collect-submodules notes
             --collect-submodules config
             --add-data "$appdir/notes/templates:notes/templates"
             --add-data "$appdir/notes/static:notes/static"
             --add-data "$appdir/notes/migrations:notes/migrations")
      ;;
    ml-iris)
      # scikit-learn pulls SciPy and NumPy, whose compiled internals PyInstaller
      # does not find on its own; the recommended remedy is to collect them all.
      FLAGS=(--collect-submodules irispipeline
             --collect-all sklearn --collect-all scipy --collect-all numpy)
      ;;
    data-report)
      FLAGS=(--collect-submodules salesreport)
      ;;
  esac
}

pyi_work="$WORK_DIR/pyi-work"
pyi_spec="$WORK_DIR/pyi-spec"
pyi_dist="$WORK_DIR/pyi-dist"

for entry in "${EXAMPLES[@]}"; do
  IFS='|' read -r name subdir script args <<<"$entry"
  appdir="$EXAMPLES_DIR/$subdir"

  section "pyinstaller build: $name"
  set_extra_flags "$name" "$appdir"
  echo "$name: --onefile ${FLAGS[*]}" >>"$flags_log"
  echo "  flags: ${FLAGS[*]:-(none)}" >&2

  # --clean starts each build from a fresh cache. A build cache shared across
  # examples can otherwise leave a bundle missing libpython and failing at run
  # time, so this keeps every build independent and reproducible.
  ( cd "$appdir" && \
    time_run "$build_csv" "build,$name" -- \
      "$PYI_PYTHON" -m PyInstaller --onefile --noconfirm --clean --log-level WARN \
        --name "$name" \
        --distpath "$pyi_dist" --workpath "$pyi_work" --specpath "$pyi_spec" \
        "${FLAGS[@]}" "$script" )

  produced="$pyi_dist/$name"
  out="$BUNDLES_DIR/pyinstaller/$name"
  if [ ! -f "$produced" ]; then
    echo "  BUILD FAILED for $name" | tee -a "$verify_log" >&2
    printf '%s,NA,build-failed\n' "$name" >>"$size_csv"
    continue
  fi
  cp "$produced" "$out"
  size="$(file_size_mb "$out")"
  echo "  built $name (${size} MB)" >&2

  # Verify it runs with no system Python.
  # shellcheck disable=SC2086
  if env -i PATH=/nonexistent HOME="$HOME" "$out" $args >>"$verify_log" 2>&1; then
    echo "  verify OK: $name" | tee -a "$verify_log" >&2
    printf '%s,%s,ok\n' "$name" "$size" >>"$size_csv"
  else
    rc=$?
    echo "  verify FAILED ($rc): $name" | tee -a "$verify_log" >&2
    printf '%s,%s,runtime-failed\n' "$name" "$size" >>"$size_csv"
  fi
done

echo "" >&2
echo "pyinstaller results in $build_csv, sizes in $size_csv, flags in $flags_log" >&2
