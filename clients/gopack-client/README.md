# gopack-client

A small installer and launcher for [gopack](https://github.com/Go-Python-Toolchain/gopack), a deployment bundler for Python written in Go.

Installing this package gives you a `gopack` command. The first time you run it, it downloads the native binary that matches your platform from the project's GitHub releases, verifies its checksum, and caches it. Every later run reuses the cached binary.

## Install

```
pip install gopack-client
```

## Use

```
gopack build ./myapp -r requirements.txt -o myapp
./myapp
```

`gopack` packs your application, a CPython runtime, and its dependencies into a
single executable that runs on a machine with no Python installed. See the [main
project](https://github.com/Go-Python-Toolchain/gopack) for full documentation.

## Supported platforms

Linux and macOS on x86_64 and arm64, and Windows on x86_64.

## License

Apache License 2.0.
