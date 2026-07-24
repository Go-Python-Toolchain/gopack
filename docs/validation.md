# gopack validation

This document records the acceptance test for the bundler.

## Bundling scientific packages

The milestone acceptance test bundles three packages that rely on compiled
extensions, NumPy, Pandas, and FastAPI, into standalone executables and runs each
one in an empty environment with no Python on the PATH. Every bundle must run and
produce a correct result.

Measured on Linux x86_64 with CPython 3.12.13:

| Package | Version | Shared objects | External libraries | Bundle size | Result |
| :-- | :-- | --: | --: | --: | :-- |
| NumPy | 2.5.1 | 22 | 0 | 270 MB | computed a sum correctly |
| Pandas | 3.0.3 | 67 | 0 | 291 MB | computed a data frame sum correctly |
| FastAPI | 0.139.2 | 1 | 0 | 255 MB | imported and constructed an app |

Each bundle was run with the environment reduced to `PATH=/nonexistent`, so there
was no system Python to fall back on. Every one ran its bundled interpreter and
produced the expected output.

## Native libraries

The auto-detection reported zero external libraries for all three packages. That
is the correct result: modern manylinux wheels vendor their own native libraries,
for example into a `numpy.libs` directory inside the package, so the shared
objects they load are already inside the bundle rather than out on the system.
gopack preserves the installed layout, so those vendored libraries travel with
the bundle and load through their relative run paths.

For extensions that link a library that is not vendored and not a standard system
library, the scan reports it as external and, unless told not to, embeds it into
the bundle and points the loader at it. That path is covered by the unit tests in
the nativelibs package.

## The acceptance test

The test lives in internal/assemble as TestBundleScientificPackages. It is guarded
by the GOPACK_NETWORK_TESTS environment variable so the normal test run stays
offline; it downloads CPython and several large packages and writes a few hundred
megabytes per bundle.

## Runtime acquisition

Two properties of how gopack acquires its runtime are checked without a network,
using fake cache directories and a local server. A build finds an
already-extracted runtime in the cache and uses it, preferring the newest version
and the stripped variant, and ignores a half-finished extraction that lacks its
completion marker. A downloaded runtime whose bytes do not match the digest the
release publishes is refused before anything is extracted. A guarded network test
(`GOPACK_NETWORK_TESTS=1`) confirms the end to end behavior: the first build
downloads and verifies a runtime, and a second build with that runtime cached
makes zero HTTP requests, which is what lets a repeat build skip the API rate
limit and an offline build work at all.

## Reproducible builds

Building the same application twice, into two different temporary directories,
produces byte-identical bundles. The check is `TestReproducibleBundleReal`,
guarded by `GOPACK_NETWORK_TESTS=1` because it installs a real dependency: the
one step that was not deterministic was byte-compilation, so it has to run
against real installed packages to be meaningful. The two builds are compared
byte for byte, and the test names the first differing offset if they diverge.
