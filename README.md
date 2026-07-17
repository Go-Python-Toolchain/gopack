# gopack

An intelligent deployment bundler for Python, written in Go.

gopack packs a Python application, a self-contained CPython runtime, and all of
its dependencies into a single executable. That executable runs on a machine
with no Python installed. On first run it extracts what it needs to a cache and
launches the app. There is no Docker image to build and no system packages to
install on the target.

gopack is part of the [Go-Python Toolchain](https://github.com/Go-Python-Toolchain).
It does not ask you to restructure your app. It bundles exactly what pip installs.

## Status

Working. gopack bundles a Python app, a CPython runtime, and its dependencies
into a single executable that runs with no system Python. It has been validated
by bundling NumPy, Pandas, and FastAPI into runnable binaries.

## Install

The easiest way is the Python launcher, which downloads the native binary for
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

## Use

```
gopack build ./myapp -r ./myapp/requirements.txt -o myapp
./myapp
```

gopack acquires a CPython runtime, installs the requirements, and writes a single
executable. The entry script defaults to `main.py`; use `--entry` and `--python`
to change the entry point or target version.

In GitHub Actions, install gopack with the bundled action:

```yaml
- uses: Go-Python-Toolchain/gopack/.github/actions/setup-gopack@v0.1.0
- run: gopack build ./app -r ./app/requirements.txt -o app
```

## Documentation

- [Getting started](docs/getting-started.md): install gopack and bundle your first app.
- [Tutorial](docs/tutorial.md): bundle a small app step by step and run it anywhere.
- [Validation](docs/validation.md): how bundling is checked, including NumPy, Pandas, and FastAPI.
- [examples/](examples/basic): a small app you can bundle right away.

## Design

- A self-extracting executable: the payload is appended to a small Go launcher.
- A relocatable CPython runtime, so the target needs no system Python.
- Dependencies installed with pip, so packages are exactly what pip would give you.
- Automatic detection of external native libraries pulled in by C extensions.

## License

Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
