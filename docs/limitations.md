# gopack Limitations

gopack does one thing well: it turns a Python application and its pip-installed
dependencies into a single executable that runs without a system Python. This
document is an honest account of what it does not do yet, and the trade-offs
that come with the approach. Knowing these up front saves surprises. Several of
them are tracked as future work in the [roadmap](roadmap.md).

## Bundle size

Bundles are tens to low hundreds of megabytes, roughly 82 to 164 MB for the
example projects. Almost all of it is the CPython runtime and the application's
pip-installed dependencies; gopack ships the standard library whole rather than
tracing which modules the application imports. It uses the stripped CPython
build, which omits the interpreter's debug symbols and was the bulk of the old
size (bundles were 256 to 340 MB before). Trimming further, by dropping stdlib
packages an application provably does not use, is possible but has a smaller
payoff and the risk of removing something the app needs. Size does not change how
a bundle is built or run.

## First-run extraction

On its first launch a bundle extracts its whole payload, the interpreter,
dependencies, and application, to a content-addressed cache. That is tens to
low hundreds of megabytes written to disk, so the first run takes longer than
later ones. Every later run reuses the extracted cache and starts quickly. The
benchmark reports this as the cold versus warm startup difference. Environments
that wipe the cache between runs pay the extraction cost each time.

## Cache growth and a writable cache

Each distinct bundle version extracts to its own directory under the cache
(`$GOPACK_CACHE`, else `$XDG_CACHE_HOME/gopack`, else `~/.cache/gopack`). Old
versions are not garbage collected automatically, so the cache grows as bundles
are updated. `gopack cache info` shows what is stored and `gopack cache clear`
reclaims it, so this is a manual step rather than an `rm` on a path you have to
work out. The launcher also needs a writable cache location: on a host with a
read-only home directory, or a locked-down container, set `GOPACK_CACHE` to a
writable path.

## One target platform per build

`gopack build` produces a bundle for the operating system and architecture of
the build machine. Building a Linux bundle on a Mac, or an arm64 bundle on an
amd64 host, is not supported yet. The design has a clear path to it (use the
published gopack binary for the target platform as the runner), but today a
bundle must be built on the platform it targets, or in a matching CI job.

## Build-time network and the runtime source

Building fetches a relocatable CPython from the python-build-standalone project
on GitHub the first time, then caches it. So the first build of a given runtime
needs network access and is limited to the Python versions that project
publishes. Later builds use the cached runtime and make no network request, so
they neither wait on GitHub nor count against its rate limit, and a build works
fully offline once the runtime is cached. The downloaded runtime is verified
against the digest the release publishes before it is used. A first build on a
fresh machine still benefits from a token if you are building many different
runtimes at once: set `GITHUB_TOKEN` (or `GOPACK_GITHUB_TOKEN`) to lift the
anonymous rate limit.

## What gets bundled

gopack bundles exactly what pip installs into the target, plus the application
tree as it is on disk. That is a deliberate strength, but it sets the boundary:

- Dependencies that are not pip-installable, or that expect OS-level packages,
  services, or system configuration, are not handled beyond the native library
  detection below.
- Data files your application reads must live inside the application directory so
  they are copied into the bundle. Files referenced by absolute paths outside the
  project are not gathered.
- The application must run from a single entry command. gopack runs one entry
  script; it is not a multi-command launcher.

## Native library detection

C extensions sometimes load native libraries. gopack inspects the installed
shared objects with the platform's linker tool (`ldd`, `otool`, or `dumpbin`),
and embeds external libraries it finds. Modern manylinux, macOS, and Windows
wheels vendor their own native libraries, so this usually finds nothing to add,
which is correct. The detection covers libraries recorded in a shared object's
load commands. A library a program opens at runtime with `dlopen` by an absolute
system path, or through an unusual mechanism, is not something the static scan
can see; on the target that library would need to be present, or added by hand.

## No code signing

gopack does not sign or notarize the executables it produces. An unsigned binary
will trigger Gatekeeper warnings on macOS and SmartScreen prompts on Windows.
Signing and notarization are left to your existing release tooling.

## Platform coverage of this build

The bundler is developed and its bundles are exercised on Linux. The macOS and
Windows launchers cross-compile from the same design, and the manifest,
extraction, and native library logic are platform-aware, but the end-to-end run
of a bundle on those platforms is validated through CI rather than in the
day-to-day development environment.
