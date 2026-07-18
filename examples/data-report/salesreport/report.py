"""Assemble the printable report."""

import numpy as np
import pandas as pd

from . import generate, transform


def build(rows: int = 10_000) -> tuple[str, bool]:
    df = transform.with_revenue(generate.sales(rows))
    summary = transform.by_region(df)

    total = df["revenue"].sum()
    reconciled = bool(np.isclose(summary["revenue"].sum(), total))

    lines = [
        "regional sales report",
        f"python {__import__('sys').version.split()[0]}",
        f"pandas {pd.__version__}",
        f"numpy {np.__version__}",
        f"rows {len(df)}",
        "",
        summary.to_string(float_format=lambda v: f"{v:,.2f}"),
        "",
        f"total revenue {total:,.2f}",
    ]
    return "\n".join(lines), reconciled
