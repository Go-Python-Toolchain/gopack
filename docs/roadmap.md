# gopack Roadmap

This is where gopack is headed. It is a direction, not a dated release plan.
Items are grouped by the problem they solve, and each one is written so its
"done" is obvious. The current constraints these address are described in
[limitations](limitations.md).

## Smaller bundles

Bundles were hundreds of megabytes because the runtime dominated them and most
of the runtime was the interpreter's debug symbols. gopack now acquires the
stripped CPython variant (`install_only_stripped`), which omits those symbols,
and falls back to the full build for a platform that does not publish a stripped
one. That alone cut the example bundles from 256 to 339 MB down to 82 to 164 MB,
and none of them changed how they build or run.

What is left here is smaller. Pruning byte-code caches, test directories, and
unused stdlib was the rest of the original plan; measured against the stripped
runtime it is worth only a couple of megabytes, since the stripped build already
excludes the test suites, and removing byte-code caches would trade a little size
for a slower first run. So the remaining size work is:

- Optionally drop parts of the standard library, such as `tkinter`, that an
  application provably does not import. This is the only pruning with enough
  payoff to be worth the risk of removing something an app needs.

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

**Done.** A build now looks in the runtime cache before contacting GitHub, so a
second build of any project makes no network request and does not touch the API
rate limit, and a build works fully offline once a runtime has been acquired. A
freshly downloaded runtime is checked against the digest the release publishes
and refused on a mismatch, so a corrupted or tampered interpreter never reaches a
bundle.

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
