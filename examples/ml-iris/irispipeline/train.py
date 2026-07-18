"""Train the pipeline and persist the fitted model."""

from __future__ import annotations

import joblib

from . import data, pipeline
from .config import ARTIFACT_PATH


def train() -> dict:
    dataset = data.load()
    model = pipeline.build()
    model.fit(dataset.x_train, dataset.y_train)

    ARTIFACT_PATH.parent.mkdir(parents=True, exist_ok=True)
    joblib.dump(
        {"model": model, "target_names": dataset.target_names},
        ARTIFACT_PATH,
    )

    train_score = model.score(dataset.x_train, dataset.y_train)
    return {
        "artifact": str(ARTIFACT_PATH),
        "train_samples": len(dataset.x_train),
        "train_accuracy": train_score,
    }
