"""Deterministic sample data, standing in for a real data source."""

import numpy as np
import pandas as pd

REGIONS = ["north", "south", "east", "west"]


def sales(rows: int = 10_000, seed: int = 42) -> pd.DataFrame:
    rng = np.random.default_rng(seed)
    return pd.DataFrame(
        {
            "region": rng.choice(REGIONS, size=rows),
            "units": rng.integers(1, 100, size=rows),
            "price": rng.uniform(5.0, 50.0, size=rows).round(2),
        }
    )
