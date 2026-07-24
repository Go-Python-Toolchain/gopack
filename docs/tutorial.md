# gopack tutorial

This walkthrough bundles a small application that uses a third party dependency,
then runs the result on a machine as if no Python were installed. It assumes
gopack is installed. If it is not, see [getting started](getting-started.md).

## 1. Write a small app

Create a directory with an entry script and a requirements file.

`myapp/main.py`:

```python
import sys
import six

print("hello from a gopack bundle")
print("python", sys.version.split()[0])
print("six", six.__version__)
print("args", sys.argv[1:])
```

`myapp/requirements.txt`:

```
six
```

## 2. Bundle it

```
gopack build ./myapp -r ./myapp/requirements.txt -o myapp
```

gopack reports each step as it acquires the runtime, stages the app and its
dependencies, and assembles the executable:

```
acquiring CPython 3.12 for linux/amd64
staging application and dependencies
assembling
wrote myapp (256.2 MB)
```

## 3. Run it anywhere

Run the single file. It works even in an environment with no Python on the path:

```
./myapp alpha beta
```

```
hello from a gopack bundle
python 3.12.13
six 1.17.0
args ['alpha', 'beta']
```

The bundle used its own interpreter and its own copy of the dependency. Nothing
was read from a system Python, because there does not need to be one.

## 4. Ship it

Copy the file to the target machine and run it:

```
scp ./myapp user@server:
ssh user@server ./myapp
```

There is nothing else to install on the server.

## 5. Leave things out

A dependency often ships its own test suite, type stubs, or documentation that a
running app does not need. `--exclude` drops them from the bundle:

```
gopack build ./myapp -r ./myapp/requirements.txt --exclude tests --exclude '*.pyi' -o myapp
```

A pattern without a slash matches a name anywhere in the tree, so `tests` removes
every directory called tests. A pattern with a slash targets one path, such as
`site-packages/scipy/misc`. The build prints how much each exclusion removed.

## 6. Manage the cache

gopack keeps two things between builds and runs: the CPython runtimes it
downloads, and the bundles it extracts on first launch. See and reclaim that
space with:

```
gopack cache info
gopack cache clear          # remove extracted bundles, keep the downloaded runtimes
gopack cache clear --all    # remove the runtimes too
```

The extracted bundles are regenerated the next time a bundle runs, so clearing
them is safe; the runtimes are downloads, which is why they are kept unless you
ask for `--all`.

## Notes on size

Bundles are tens to low hundreds of megabytes: most of a bundle is the CPython
runtime and the dependencies pip installs. gopack ships the stripped runtime, so
the size is mostly your dependencies. `--exclude` trims what a dependency carries
but does not need. None of this changes how you build or run a bundle.

## Where to go next

- The [example project](../examples/basic) is ready to bundle.
- The getting started guide explains the design in more depth.
