"""Command line entry for the iris pipeline.

Usage:
  run.py demo                 train, evaluate, and predict in one go (default)
  run.py train                fit the model and save the artifact
  run.py evaluate             score the saved model on the test split
  run.py predict a b c d      classify one flower from four measurements

This is a small but real ML project: data loading, a scikit-learn Pipeline,
training with a persisted artifact, evaluation, and prediction, split across the
irispipeline package. gopack bundles NumPy, SciPy, and scikit-learn with the
interpreter, so the whole thing runs where no scientific Python is installed.
"""

import sys

from irispipeline import evaluate as evaluate_mod
from irispipeline import predict as predict_mod
from irispipeline import train as train_mod


def _print_header() -> None:
    import numpy
    import sklearn

    print("iris pipeline")
    print("python", sys.version.split()[0])
    print("scikit-learn", sklearn.__version__)
    print("numpy", numpy.__version__)


def do_train() -> int:
    info = train_mod.train()
    print(f"trained on {info['train_samples']} samples")
    print(f"train accuracy {info['train_accuracy']:.3f}")
    print(f"saved model to {info['artifact']}")
    return 0


def do_evaluate() -> int:
    info = evaluate_mod.evaluate()
    print(f"evaluated on {info['test_samples']} samples")
    print(f"test accuracy {info['test_accuracy']:.3f}")
    print(info["report"])
    return 0 if info["test_accuracy"] >= 0.85 else 1


def do_predict(args: list[str]) -> int:
    if len(args) == 4:
        samples = [[float(v) for v in args]]
    else:
        samples = predict_mod.SAMPLES
    for features in samples:
        result = predict_mod.predict(features)
        dims = ", ".join(f"{v:.1f}" for v in result["features"])
        print(f"[{dims}] -> {result['label']} ({result['confidence']:.2f})")
    return 0


def main(argv: list[str]) -> int:
    command = argv[0] if argv else "demo"
    rest = argv[1:]

    if command == "train":
        _print_header()
        return do_train()
    if command == "evaluate":
        _print_header()
        return do_evaluate()
    if command == "predict":
        return do_predict(rest)
    if command == "demo":
        _print_header()
        rc = do_train()
        rc |= do_evaluate()
        print("sample predictions:")
        rc |= do_predict([])
        return rc

    print(f"unknown command {command!r}", file=sys.stderr)
    return 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
