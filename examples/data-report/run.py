"""Entry point for the sales report.

Usage:
  run.py [rows]     build the report over `rows` synthetic orders (default 10000)

A small but real data pipeline: generate, transform, and report, split across
the salesreport package. It leans on pandas and NumPy, both of which carry
compiled extensions, so it is exactly the kind of tool that is awkward to hand
to someone without the scientific stack. gopack makes it one executable.
"""

import sys

from salesreport import report


def main(argv: list[str]) -> int:
    rows = int(argv[0]) if argv else 10_000
    text, reconciled = report.build(rows)
    print(text)
    # The per-region revenue must sum back to the grand total.
    return 0 if reconciled else 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
