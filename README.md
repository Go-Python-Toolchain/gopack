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
- [Architecture](docs/architecture.md): how a bundle is built and why gopack is its own launcher.
- [Protocol](docs/protocol.md): the exact on-disk format of a bundle and the launch sequence.
- [Benchmarks](docs/benchmarks.md): build time, bundle size, and startup, next to PyInstaller.
- [Validation](docs/validation.md): how bundling is checked, including NumPy, Pandas, and FastAPI.
- [Limitations](docs/limitations.md): what gopack does not do yet, and the trade-offs.
- [Roadmap](docs/roadmap.md): where gopack is headed.

## Examples

Real, multi-file applications you can bundle and run, each with its own README:

- [basic](examples/basic): a minimal quickstart app.
- [ml-iris](examples/ml-iris): a scikit-learn training and prediction pipeline.
- [fastapi-service](examples/fastapi-service): a FastAPI service with Jinja templates and static files.
- [django-notes](examples/django-notes): a full Django project with the admin site, migrations, and management commands.
- [data-report](examples/data-report): a pandas and NumPy data pipeline.
- [benchmark](examples/benchmark): the harness that measures all of the above and compares against PyInstaller.

## Design

- A self-extracting executable: gopack is its own launcher, so a bundle is a copy of gopack with a compressed payload appended. There is no separate launcher program.
- A relocatable CPython runtime, so the target needs no system Python.
- Dependencies installed with pip, so packages are exactly what pip would give you.
- Automatic detection of external native libraries pulled in by C extensions.

## License

Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
