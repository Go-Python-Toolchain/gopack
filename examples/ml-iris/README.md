# ml-iris: a scikit-learn pipeline in one executable

A small but real machine learning project, laid out the way one usually is:

```
ml-iris/
  run.py                 command line entry: train / evaluate / predict / demo
  irispipeline/
    config.py            paths and hyperparameters
    data.py              load and split the dataset
    pipeline.py          the scikit-learn Pipeline (scaler + random forest)
    train.py             fit and persist the model with joblib
    evaluate.py          load the model and score it
    predict.py           load the model and classify a flower
  requirements.txt       scikit-learn (pulls in NumPy and SciPy)
```

It depends on NumPy, SciPy, and scikit-learn, all of which carry compiled
extensions, so it is exactly the kind of program that is painful to hand to
someone without the scientific stack. gopack bundles the interpreter and those
wheels into one file.

## Bundle it

```
gopack build ./ml-iris -r ./ml-iris/requirements.txt --entry run.py -o iris
```

## Run it, with no Python installed

```
./iris demo
```

```
iris pipeline
python 3.12.13
scikit-learn 1.5.1
numpy 1.26.4
trained on 105 samples
train accuracy 1.000
saved model to .../artifacts/model.joblib
evaluated on 45 samples
test accuracy 0.889
              precision    recall  f1-score   support

      setosa       1.00      1.00      1.00        15
  versicolor       0.78      0.93      0.85        15
   virginica       0.92      0.73      0.81        15

    accuracy                           0.89        45
   macro avg       0.90      0.89      0.89        45
weighted avg       0.90      0.89      0.89        45
sample predictions:
[5.1, 3.5, 1.4, 0.2] -> setosa (1.00)
[6.0, 2.7, 5.1, 1.6] -> versicolor (0.72)
[6.9, 3.1, 5.4, 2.1] -> virginica (0.99)
```

The subcommands work individually too. `train` fits the model and saves the
artifact, `evaluate` scores the saved model, and `predict` classifies a single
flower from four measurements:

```
./iris train
./iris evaluate
./iris predict 6.1 2.8 4.7 1.2
```

The trained model is written to `artifacts/model.joblib` inside the extraction
cache, which is writable, so `train` and the later commands share it.
