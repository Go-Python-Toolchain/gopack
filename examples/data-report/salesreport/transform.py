"""The analytics: derive revenue and aggregate by region."""

import pandas as pd


def with_revenue(df: pd.DataFrame) -> pd.DataFrame:
    out = df.copy()
    out["revenue"] = (out["units"] * out["price"]).round(2)
    return out


def by_region(df: pd.DataFrame) -> pd.DataFrame:
    return (
        df.groupby("region")
        .agg(
            orders=("units", "size"),
            units=("units", "sum"),
            revenue=("revenue", "sum"),
        )
        .sort_values("revenue", ascending=False)
    )
