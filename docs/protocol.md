# gopack Bundle Protocol

This document specifies the on-disk format of a gopack bundle and the runtime
protocol the launcher follows to run it. It is the reference for anyone who wants
to inspect a bundle, build a compatible one, or understand exactly what happens
when a bundle runs. The [architecture](architecture.md) doc explains the design
choices; this one pins down the format.

## The executable layout

A finished bundle is a single file with three parts, in order:

```
+-------------------------+  offset 0
|   runner                |   a real gopack binary for the target platform
+-------------------------+  offset = len(runner)
|   payload               |   a zip archive
+-------------------------+  offset = len(runner) + len(payload)
|   trailer (40 bytes)    |   magic + payload offset + length + content key
+-------------------------+  end of file
```

The runner is an unmodified gopack binary. gopack acts as the command line tool
when it has no payload and as the launcher when it does, so appending a payload
turns a copy of gopack into a self-extracting application. Nothing in the runner
region is rewritten; the payload and trailer are simply concatenated onto the
end.

## The trailer

The trailer records where the payload is and its content key. The current format
is `GOPACK02`, 40 bytes:

| Bytes | Field | Encoding |
| --- | --- | --- |
| 0 to 7 | magic | the ASCII string `GOPACK02` |
| 8 to 15 | payload offset | unsigned 64-bit, big-endian |
| 16 to 23 | payload length | unsigned 64-bit, big-endian |
| 24 to 39 | content key | 16 ASCII hex characters, the payload's key |

The content key is the first 16 hex characters of the payload's SHA-256, computed
once when the bundle is built. It names the extraction cache directory, so the
launcher reads it from the trailer instead of hashing the whole payload on every
run.

The magic ends in a two-digit format version. The earlier `GOPACK01` format was
the same without the content key, a 24-byte trailer; a reader still accepts it
and recovers the key by hashing the payload, so bundles built before the key was
recorded keep running. A reader that recognizes neither magic should refuse the
file rather than guess.

To detect a bundle, read the final 40 bytes and check for the `GOPACK02` magic at
their start; if it is not there, check for `GOPACK01` in the final 24 bytes.
Validate that `offset + length + trailer size` does not exceed the file size. If
no magic matches or the bounds do not hold, the file has no gopack payload and
gopack runs as the command line tool.

## The payload

The payload is a standard zip archive. Streaming it as a zip means a bundle can
be produced and extracted without holding it all in memory, which matters because
bundles are hundreds of megabytes. The archive holds these top-level entries:

| Entry | Contents |
| --- | --- |
| `gopack.json` | the run manifest (see below) |
| `app/` | the application source, copied as-is |
| `site-packages/` | dependencies, installed with `pip install --target` |
| `libs/` | external native libraries that were detected and embedded (optional) |
| `python/` | a relocatable CPython runtime |

Paths inside the zip always use forward slashes. Symbolic links in the source
tree are followed and stored as regular files, so the extractor never has to
recreate a link and a bundle cannot smuggle a link that points outside itself.

## The run manifest

`gopack.json` describes how to run the application:

```json
{
  "name": "myapp",
  "entry": ["python/bin/python3", "app/main.py"],
  "env": {
    "PYTHONPATH": "${ROOT}/site-packages"
  }
}
```

| Field | Type | Meaning |
| --- | --- | --- |
| `name` | string, optional | a human label for the bundle |
| `entry` | array of strings, required | the command to run |
| `env` | object, optional | environment variables to set for the command |

Entry resolution, applied at launch:

- The first element is the program. If it is a relative path it is joined with
  the extraction root, so `python/bin/python3` becomes an absolute path inside
  the cache. On Windows the interpreter is `python/python.exe`.
- Each later element is an argument. A relative argument that names a file which
  exists under the root is resolved to that absolute path; anything else is
  passed through unchanged. This is how `app/main.py` becomes an absolute script
  path while a flag like `--verbose` is left alone.
- The user's own command line arguments are appended after the manifest
  arguments.

Environment resolution:

- Every value may contain the token `${ROOT}`, which is replaced with the
  extraction root. This is how `PYTHONPATH` points at the bundle's own
  `site-packages`, and how an embedded native library directory is put on
  `LD_LIBRARY_PATH` (Linux), `DYLD_LIBRARY_PATH` (macOS), or `PATH` (Windows).
- The resolved variables are layered on top of the process environment, so the
  bundle inherits the caller's environment and overrides only what the manifest
  names. Variables are emitted in sorted order for a stable, reproducible launch.

## The content-addressed cache

On first run the launcher extracts the payload to a cache directory named by the
payload's content, so two bundles with identical payloads share one extraction
and a changed payload lands in a new directory.

- The key is the first 16 hex characters of the SHA-256 of the payload bytes. It
  is recorded in the trailer at build time, so a warm run reads it directly; a
  bundle in the older format without a recorded key has it recomputed from the
  payload, which gives the same value.
- The cache root is `$GOPACK_CACHE` if set, else `$XDG_CACHE_HOME/gopack` if set,
  else `~/.cache/gopack`. The extraction goes to `<cache root>/<key>`.
- A marker file `.gopack-complete` is written after a successful extraction. Its
  presence is what tells a later run the directory is fully populated; without
  it the launcher extracts again. This keeps a half-finished extraction, for
  example one interrupted partway, from being trusted.

Extraction unpacks the zip into the cache directory, preserving each entry's file
mode. Every entry path is checked so it cannot escape the extraction root; an
entry whose cleaned path would resolve outside the root is rejected rather than
written.

## The launch sequence

When a bundle runs, the launcher performs these steps:

1. Find its own path.
2. Open the trailer, locate the payload, and read its content key (recomputing it
   from the payload only for a bundle in the older format that did not record it).
3. Compute the cache directory from the key.
4. If the directory does not carry the completion marker, extract the payload
   into it and write the marker.
5. Read and parse `gopack.json`. A manifest with no entry command is an error.
6. Resolve the entry command and append the user's arguments.
7. Run the command with its working directory set to the extraction root,
   standard input, output, and error connected to the launcher's, and the
   environment set from the manifest.
8. Wait for the command and exit with its exit code.

Later runs skip extraction because the marker is present, so startup is the cost
of reading the manifest and starting the interpreter. For a bundle that records
its content key this no longer includes hashing the payload, which for a large
bundle was most of a warm start.

## Building a compatible bundle

A bundle can be produced without gopack by following the format: stage the
`app`, `site-packages`, and `python` trees, write `gopack.json`, zip them with
forward-slash paths, concatenate the zip onto a gopack binary for the target
platform, and append the 40-byte trailer recording the zip's offset, length, and
content key (the first 16 hex characters of its SHA-256). gopack's own `build`
command does exactly this, using a copy of the running gopack binary as the
runner.
