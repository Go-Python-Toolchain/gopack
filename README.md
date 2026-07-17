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

Early development. The command line skeleton and build pipeline are in place. The
self-extracting launcher, the CPython runtime acquisition, the bundle generator,
and the native library auto-detection are being built in order.

## Install

While pre-release, build from source:

```
git clone https://github.com/Go-Python-Toolchain/gopack
cd gopack
go build -o gopack .
./gopack version
```

Requires Go 1.22 or newer.

## Design

- A self-extracting executable: the payload is appended to a small Go launcher.
- A relocatable CPython runtime, so the target needs no system Python.
- Dependencies installed with pip, so packages are exactly what pip would give you.
- Automatic detection of external native libraries pulled in by C extensions.

## License

Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
