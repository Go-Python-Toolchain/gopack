#!/usr/bin/env bash
# Prepare the benchmark workspace: build gopack, warm the CPython runtime cache,
# and create a Python environment with the example dependencies and PyInstaller
# for the comparison builds. Everything lands under examples/benchmark/work,
# which is git ignored.
#
# Re-running is safe: existing pieces are left in place.

source "$(dirname "${BASH_SOURCE[0]}")/config.sh"

# --- gopack binary --------------------------------------------------------
if [ -z "$GOPACK_BIN" ] || [ ! -x "$GOPACK_BIN" ]; then
  if command -v go >/dev/null 2>&1; then
    echo "building gopack"
    ( cd "$REPO_ROOT" && go build -o gopack . )
    GOPACK_BIN="$REPO_ROOT/gopack"
  else
    echo "gopack binary not found and Go is not installed; cannot continue" >&2
    exit 1
  fi
fi
echo "gopack: $("$GOPACK_BIN" version 2>/dev/null)"

# --- PyInstaller environment ---------------------------------------------
venv="$TOOLS_DIR/pyi-venv"
if [ ! -x "$venv/bin/python" ]; then
  echo "creating PyInstaller venv with example dependencies (this pulls NumPy, SciPy, scikit-learn, Django, FastAPI, ...)"
  python3 -m venv "$venv"
  "$venv/bin/pip" install --quiet --upgrade pip
  for entry in "${EXAMPLES[@]}"; do
    IFS='|' read -r name subdir script args <<<"$entry"
    req="$EXAMPLES_DIR/$subdir/requirements.txt"
    [ -f "$req" ] && "$venv/bin/pip" install --quiet -r "$req"
  done
  "$venv/bin/pip" install --quiet "pyinstaller==$PYINSTALLER_VERSION"
else
  echo "PyInstaller venv already present"
fi
"$venv/bin/python" -c "import PyInstaller; print('PyInstaller', PyInstaller.__version__)" 2>/dev/null || \
  echo "warning: PyInstaller not importable in $venv" >&2

# --- Warm the CPython runtime cache --------------------------------------
# The first gopack build downloads a relocatable CPython (a couple of minutes).
# Do it once here against the basic example so the measured builds are repeats.
if [ ! -d "$GOPACK_CACHE/runtimes" ]; then
  echo "warming the CPython runtime cache (one-time download)"
  tmp="$WORK_DIR/warm-basic"
  "$GOPACK_BIN" build "$EXAMPLES_DIR/basic" -r "$EXAMPLES_DIR/basic/requirements.txt" \
    --entry main.py -o "$tmp" >/dev/null 2>&1 || true
  rm -f "$tmp"
fi

echo ""
echo "Setup complete."
echo "  gopack:      $GOPACK_BIN"
echo "  PyInstaller: $venv"
echo "  cache:       $GOPACK_CACHE"
