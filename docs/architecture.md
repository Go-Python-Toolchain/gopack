# gopack Architecture

This document explains how gopack turns a Python application into a single
executable, and records the main design decisions.

## One binary, two roles: no launcher stub

A self-extracting executable needs a small program at the front that, when run,
finds the appended payload, extracts it, and starts the real application. A
common way to build this is to keep a separate minimal launcher program and
append the payload to it. gopack does not do that.

Instead, gopack is its own launcher. When gopack starts it checks whether it has
a payload appended to itself. If it does, it acts as the launcher: it extracts
the payload and runs the bundled application. If it does not, it acts as the
command line tool. So a finished bundle is simply a copy of the gopack binary
with a payload appended, and the same tool that builds a bundle is the one that
runs it.

This choice removes a whole category of moving parts:

- There is no separate launcher program to build, version, ship, or keep in sync.
- Building a bundle needs no Go toolchain and no prebuilt helper binaries. gopack
  reads a copy of itself and appends the payload.
- The launcher and the tool can never drift apart, because they are the same
  binary.

For a bundle that targets a different operating system than the build machine,
the same idea applies: gopack uses the gopack binary for the target platform,
which is published with every release, as the base to append the payload to.
There is still no stub; the base is always a real gopack binary.

## The payload

The payload is a zip appended to the gopack binary, followed by a fixed trailer
that records where the payload begins and how long it is. The zip contains:

- `gopack.json`, the run manifest, which names the entry command and any
  environment the app needs.
- `app`, the application source.
- `site-packages`, the dependencies installed with pip.
- `python`, a relocatable CPython runtime.
- `libs`, any external native libraries that were detected and embedded.

## Running a bundle

On first run the launcher reads its own file, finds the payload through the
trailer, and extracts it to a cache directory keyed by the payload's content
hash. It then runs the bundled interpreter on the entry script, setting the
environment from the manifest so that the app finds its dependencies and any
embedded native libraries. Later runs reuse the extracted cache, so startup is
fast.

## Native libraries

C extensions sometimes load native libraries. gopack scans the installed shared
objects with the platform's linker inspection tool, separates standard system
libraries and libraries already inside the bundle from the rest, and embeds the
external ones. Modern wheels usually vendor their own native libraries, so the
scan often finds nothing external to add, which is the correct result.

## The CPython runtime

gopack downloads a relocatable CPython build from the python-build-standalone
project for the requested version and platform, and caches it. The install is
relocatable, so it runs correctly from the extraction cache without a system
Python.

It prefers the stripped variant of that build (`install_only_stripped`), which
omits the interpreter's debug symbols. Those symbols were most of a bundle's
size, so preferring the stripped build roughly halves it, and a platform that
does not publish a stripped build falls back to the full one.
