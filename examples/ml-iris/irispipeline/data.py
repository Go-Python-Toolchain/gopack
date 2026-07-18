"""Dataset loading and splitting."""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split

from .config import RANDOM_STATE, TEST_SIZE


@dataclass
class Dataset:
    x_train: np.ndarray
    x_test: np.ndarray
    y_train: np.ndarray
    y_test: np.ndarray
    feature_names: list[str]
    target_names: list[str]


def load() -> Dataset:
    raw = load_iris()
    x_train, x_test, y_train, y_test = train_test_split(
        raw.data,
        raw.target,
        test_size=TEST_SIZE,
        random_state=RANDOM_STATE,
        stratify=raw.target,
    )
    return Dataset(
        x_train=x_train,
        x_test=x_test,
        y_train=y_train,
        y_test=y_test,
        feature_names=list(raw.feature_names),
        target_names=list(raw.target_names),
    )
