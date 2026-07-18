#!/usr/bin/env python3
"""Turn the raw benchmark CSVs into the published markdown tables.

Reads the build, size, and startup CSVs from the raw directory and writes
results.md. Standard library only, so it runs anywhere the harness runs.
"""

import csv
import os
import statistics
import sys

RAW = os.environ.get("GOPACK_BENCH_RAW")
if not RAW:
    here = os.path.dirname(os.path.abspath(__file__))
    work = os.environ.get("GOPACK_BENCH_WORK") or os.path.join(here, "..", "work")
    RAW = os.path.join(work, "raw")

# Preserve the example order used by the harness.
ORDER = ["basic", "data-report", "ml-iris", "fastapi", "django"]


def load_csv(name):
    path = os.path.join(RAW, name)
    if not os.path.exists(path):
        return []
    with open(path, newline="") as fh:
        return list(csv.DictReader(fh))


def num(row, key):
    try:
        return float(row[key])
    except (TypeError, ValueError, KeyError):
        return None


def build_times(rows):
    """label like build,<name> -> seconds."""
    out = {}
    for r in rows:
        label = r.get("label", "")
        parts = label.split(",")
        if len(parts) == 2 and parts[0] == "build":
            v = num(r, "wall_seconds")
            if v is not None:
                out[parts[1]] = v
    return out


def sizes(rows):
    out = {}
    for r in rows:
        name = r.get("name")
        if not name:
            continue
        out[name] = (r.get("size_mb", "NA"), r.get("status", "ok"))
    return out


def startup_medians(rows):
    """(tool,phase,name) -> median wall ms; cold uses the single sample."""
    buckets = {}
    for r in rows:
        parts = r.get("label", "").split(",")
        if len(parts) != 3:
            continue
        tool, phase, name = parts
        v = num(r, "wall_seconds")
        if v is None:
            continue
        buckets.setdefault((tool, phase, name), []).append(v * 1000.0)
    return {k: statistics.median(v) for k, v in buckets.items()}


def order_names(present):
    named = [n for n in ORDER if n in present]
    extra = [n for n in present if n not in ORDER]
    return named + sorted(extra)


def build_size_table(gbuild, gsize, pbuild, psize):
    names = order_names(set(gbuild) | set(gsize) | set(pbuild) | set(psize))
    lines = [
        "### Build time and bundle size\n",
        "Build time is a repeat build (the CPython runtime is already cached).",
        "Lower is better on both.\n",
        "| Example | gopack build | gopack size | PyInstaller build | PyInstaller size |",
        "| --- | ---: | ---: | ---: | ---: |",
    ]
    for n in names:
        gb = f"{gbuild[n]:.1f} s" if n in gbuild else "n/a"
        gs = f"{gsize[n][0]} MB" if n in gsize else "n/a"
        pb = f"{pbuild[n]:.1f} s" if n in pbuild else "n/a"
        if n in psize:
            mb, status = psize[n]
            ps = f"{mb} MB" if status == "ok" else f"{mb} MB ({status})"
            if mb == "NA":
                ps = status
        else:
            ps = "n/a"
        lines.append(f"| {n} | {gb} | {gs} | {pb} | {ps} |")
    return "\n".join(lines) + "\n"


def startup_table(med):
    names = order_names({k[2] for k in med})
    lines = [
        "### Startup latency\n",
        "Cold is the first launch; warm is the median of subsequent launches,",
        "each running the example's deterministic command. gopack extracts once",
        "to a content-addressed cache, so its warm launches skip extraction;",
        "PyInstaller onefile re-extracts every launch. Lower is better.\n",
        "| Example | gopack cold | gopack warm | PyInstaller cold | PyInstaller warm |",
        "| --- | ---: | ---: | ---: | ---: |",
    ]
    for n in names:
        def cell(tool, phase):
            v = med.get((tool, phase, n))
            return f"{v:.0f} ms" if v is not None else "n/a"
        lines.append(
            f"| {n} | {cell('gopack','cold')} | {cell('gopack','warm')} | "
            f"{cell('pyinstaller','cold')} | {cell('pyinstaller','warm')} |"
        )
    return "\n".join(lines) + "\n"


def main():
    gbuild = build_times(load_csv("gopack_build.csv"))
    pbuild = build_times(load_csv("pyinstaller_build.csv"))
    gsize = sizes(load_csv("gopack_size.csv"))
    psize = sizes(load_csv("pyinstaller_size.csv"))
    med = startup_medians(load_csv("startup.csv"))

    parts = ["# gopack benchmark results\n"]
    machine = os.path.join(RAW, "machine.txt")
    if os.path.exists(machine):
        parts.append("```\n" + open(machine).read().rstrip() + "\n```\n")
    if gbuild or pbuild:
        parts.append(build_size_table(gbuild, gsize, pbuild, psize))
    if med:
        parts.append(startup_table(med))
    if not (gbuild or med):
        parts.append("_No samples found. Run the harness first._\n")

    report = "\n".join(parts)
    out = os.path.join(RAW, "results.md")
    with open(out, "w") as fh:
        fh.write(report)
    sys.stdout.write(report)
    sys.stderr.write(f"\nwrote {out}\n")


if __name__ == "__main__":
    main()
