#!/usr/bin/env bash
# One command to reproduce the whole gopack benchmark: capture the machine,
# build every example with gopack and with PyInstaller, measure startup latency,
# and write results.md. Run setup.sh first to download the runtime and install
# PyInstaller.
#
#   scripts/setup.sh     # once: build gopack, warm the runtime cache, make the PyInstaller env
#   scripts/run.sh       # build, measure, report

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$here/config.sh"

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 not found; the measurement wrapper needs it." >&2
  exit 1
fi
if [ -z "$GOPACK_BIN" ] || [ ! -x "$GOPACK_BIN" ]; then
  echo "gopack binary not found; run setup.sh or build it: (cd $REPO_ROOT && go build -o gopack .)" >&2
  exit 1
fi

bash "$here/machine.sh"
bash "$here/gopack_build.sh"

# The PyInstaller comparison is optional: skip it cleanly if PyInstaller is not
# installed, so a gopack-only run still produces results.
if "$here/../work/tools/pyi-venv/bin/python" -c "import PyInstaller" 2>/dev/null \
   || "$TOOLS_DIR/smoke-venv/bin/python" -c "import PyInstaller" 2>/dev/null; then
  bash "$here/pyinstaller_build.sh" || echo "pyinstaller stage reported errors; continuing" >&2
else
  echo "PyInstaller not installed; skipping the comparison build (run setup.sh to include it)" >&2
fi

bash "$here/startup.sh"
python3 "$here/aggregate.py"

echo ""
echo "Done. Raw logs and results.md are in: $RAW_DIR"
