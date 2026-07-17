import sys

import six

print("hello from a gopack bundle")
print("python", sys.version.split()[0])
print("six", six.__version__)
print("args", sys.argv[1:])
