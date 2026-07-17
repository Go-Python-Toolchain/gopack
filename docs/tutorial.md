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

## Notes on size

Bundles are large today because the CPython runtime is the full install, which
carries the whole standard library. A future option to use a stripped runtime and
prune unused files will bring the size down. It does not change how you build or
run a bundle.

## Where to go next

- The [example project](../examples/basic) is ready to bundle.
- The getting started guide explains the design in more depth.
