# Getting started with gopack

gopack packs a Python application into a single executable. That executable
carries its own CPython runtime and all of the application's dependencies, so it
runs on a machine that has no Python installed. On first run it extracts what it
needs to a cache and launches the app.

## Install

The quickest way is the Python launcher, which downloads the native binary for
your platform on first use:

```
pip install gopack-client
gopack version
```

Or build from source with Go 1.22 or newer:

```
git clone https://github.com/Go-Python-Toolchain/gopack
cd gopack
go build -o gopack .
./gopack version
```

## Bundle your first app

Suppose you have an application directory with an entry script and a requirements
file:

```
myapp/
  main.py
  requirements.txt
```

Bundle it:

```
gopack build ./myapp -r ./myapp/requirements.txt -o myapp
```

gopack downloads a CPython runtime, installs the requirements, and writes a single
executable named `myapp`. Run it like any other program, even where Python is not
installed:

```
./myapp
```

The entry script defaults to `main.py`. Use `--entry` to point at a different one,
and `--python` to target a specific Python version.

## How it works

The gopack binary is its own launcher. A bundle is a copy of gopack with a
compressed payload appended to it, so there is no separate launcher program: the
same tool that builds a bundle is the one that runs it. The payload holds the
CPython runtime, your application, and its installed dependencies. gopack runs as
the command line tool when it has no payload, and as the launcher when it does.
The first time you run a bundle it extracts the payload to a cache keyed by
content, then runs the bundled interpreter on your entry script. Later runs reuse
the cache.

C extensions sometimes load native libraries that are not part of the standard
system. gopack detects those and embeds them, so the bundle stays self-contained.

## Where to go next

- The [tutorial](tutorial.md) bundles a small app step by step.
- The [example project](../examples/basic) is ready to bundle.
