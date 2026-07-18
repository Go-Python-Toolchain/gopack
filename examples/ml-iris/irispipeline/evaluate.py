"""Load the persisted model and score it on the held-out test split."""

from __future__ import annotations

import joblib
from sklearn.metrics import accuracy_score, classification_report

from . import data
from .config import ARTIFACT_PATH


def load_model() -> dict:
    if not ARTIFACT_PATH.exists():
        raise FileNotFoundError(
            f"no model at {ARTIFACT_PATH}; run 'train' first"
        )
    return joblib.load(ARTIFACT_PATH)


def evaluate() -> dict:
    bundle = load_model()
    model = bundle["model"]
    target_names = bundle["target_names"]

    dataset = data.load()
    predictions = model.predict(dataset.x_test)
    accuracy = accuracy_score(dataset.y_test, predictions)
    report = classification_report(
        dataset.y_test, predictions, target_names=target_names
    )
    return {
        "test_samples": len(dataset.x_test),
        "test_accuracy": accuracy,
        "report": report,
    }
