"""Shared configuration for the iris pipeline."""

from pathlib import Path

# The trained model is written next to the project. Inside a gopack bundle this
# resolves to the extraction cache, which is writable, so train can persist an
# artifact that evaluate and predict load back.
PROJECT_ROOT = Path(__file__).resolve().parent.parent
ARTIFACT_PATH = PROJECT_ROOT / "artifacts" / "model.joblib"

RANDOM_STATE = 42
TEST_SIZE = 0.3
N_ESTIMATORS = 200
