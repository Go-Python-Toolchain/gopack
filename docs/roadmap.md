# gopack Roadmap

This is where gopack is headed. It is a direction, not a dated release plan.
Items are grouped by the problem they solve, and each one is written so its
"done" is obvious. The current constraints these address are described in
[limitations](limitations.md).

## Smaller bundles

The biggest gap between gopack today and the blueprint's target is size: bundles
are hundreds of megabytes when the goal is tens.

- Use the stripped CPython variant (`install_only_stripped`) instead of the full
  install.
- Prune byte-code caches (`__pycache__`), test directories, and other files that
  are not needed at run time from the staged tree.
- Optionally drop parts of the standard library the application does not import.

Done when the example bundles land in the tens-of-megabytes range without
changing how they are built or run.

## Cross-platform builds

Today a bundle targets the platform it is built on. The design already supports
using a real gopack binary for the target platform as the runner, published with
every release.

- Add `--target os/arch` to `gopack build`.
- Fetch the matching gopack binary from the gopack release and use it as the
  runner, then append the payload as usual.

Done when a Linux host can build a working macOS or Windows bundle, and an amd64
host can build an arm64 bundle.

## Faster, more reliable builds

- Skip the GitHub release lookup when the requested runtime is already in the
  cache, so repeat builds do not touch the network or the API rate limit at all.
- Support a fully offline build against a pre-populated runtime cache, with a
  clear error when the runtime is missing rather than a network failure.

Done when a second build of the same project makes no network requests.

## Reproducible output

- Normalize file ordering and timestamps in the payload zip so that building the
  same inputs twice yields byte-identical bundles.

Done when two builds of an unchanged project produce identical checksums.

## Trimming what goes in

- A `--exclude` option to drop files and directories from the application tree,
  for build artifacts and local scratch that should not ship.
- Optional detection of installed dependencies the application never imports, so
  they can be left out.

Done when a project can shrink its bundle by excluding files it names, and unused
dependencies can be reported.

## Signing and distribution

- Hooks to sign and notarize the produced executable as part of the build, for
  macOS Gatekeeper and Windows SmartScreen.

Done when a signed, notarized bundle runs without a security prompt on a stock
machine.

## Cache lifecycle

- A `gopack cache` command to list extracted bundles and their sizes and to prune
  old versions, so the run-time cache does not grow without bound.

Done when a user can see and reclaim the space bundles use.

## Platform validation

- Exercise end-to-end bundle runs on macOS and Windows in CI, matching the
  coverage the Linux path has today.

Done when the acceptance run passes on all three platforms in CI.
