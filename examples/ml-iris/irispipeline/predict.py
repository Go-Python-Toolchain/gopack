"""Load the persisted model and classify a single flower."""

from __future__ import annotations

import numpy as np

from .evaluate import load_model

# A representative sample from each iris class, used when the caller does not
# pass their own measurements.
SAMPLES = [
    [5.1, 3.5, 1.4, 0.2],
    [6.0, 2.7, 5.1, 1.6],
    [6.9, 3.1, 5.4, 2.1],
]


def predict(features: list[float]) -> dict:
    bundle = load_model()
    model = bundle["model"]
    target_names = bundle["target_names"]

    row = np.asarray(features, dtype=float).reshape(1, -1)
    label = int(model.predict(row)[0])
    proba = model.predict_proba(row)[0]
    return {
        "features": features,
        "label": target_names[label],
        "confidence": float(proba[label]),
    }
