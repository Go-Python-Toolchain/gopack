"""gopack-client installs and launches the native gopack binary.

gopack is a deployment bundler that packs a Python app, a CPython runtime, and
its dependencies into a single self-contained executable. This package is a thin
launcher: on first use it downloads the binary that matches your platform from
the project's GitHub releases, caches it, and runs it. Every later call reuses
the cached binary.
"""

from ._binary import ensure_binary, resolve_version

__all__ = ["ensure_binary", "resolve_version"]
