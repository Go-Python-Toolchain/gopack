"""Model definition: a scaler plus a random forest, as a scikit-learn Pipeline."""

from sklearn.ensemble import RandomForestClassifier
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import StandardScaler

from .config import N_ESTIMATORS, RANDOM_STATE


def build() -> Pipeline:
    return Pipeline(
        steps=[
            ("scaler", StandardScaler()),
            (
                "forest",
                RandomForestClassifier(
                    n_estimators=N_ESTIMATORS,
                    random_state=RANDOM_STATE,
                ),
            ),
        ]
    )
