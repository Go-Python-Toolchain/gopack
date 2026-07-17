# Basic gopack example

A tiny application that uses a third party dependency. Bundle it into one
executable and run it with no system Python.

## Bundle

```
gopack build . -r requirements.txt -o myapp
```

gopack acquires a CPython runtime, installs `six`, and writes a single executable
named `myapp`.

## Run

```
./myapp alpha beta
```

```
hello from a gopack bundle
python 3.12.13
six 1.17.0
args ['alpha', 'beta']
```

The executable carries its own interpreter and dependency, so it runs even where
Python is not installed.
