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

## Known follow-up

Bundles are currently around 260 to 290 MB because the CPython runtime is the full
install, which carries the entire standard library. Using the stripped runtime
variant and pruning byte-code caches and test directories would bring bundles much
closer to the tens of megabytes range, without changing how they are built or run.
