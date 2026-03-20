from __future__ import annotations

import pickle
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

import numpy as np

from quant.patterns import build_fingerprint


@dataclass(slots=True)
class OutcomeModelMetrics:
    rows_used: int
    train_rows: int
    test_rows: int
    mae_1d: float
    directional_accuracy: float


def build_outcome_training_data(rows: Iterable[dict]) -> tuple[np.ndarray, np.ndarray]:
    features: list[list[float]] = []
    targets: list[float] = []
    for row in rows:
        close = _safe_float(row.get("close"))
        future_close = _safe_float(row.get("close_1d_later"), default=np.nan)
        if np.isnan(close) or np.isnan(future_close) or close == 0:
            continue
        features.append(build_fingerprint(row))
        targets.append(((future_close - close) / close) * 100.0)

    if not features:
        return np.empty((0, 30), dtype=float), np.empty((0,), dtype=float)

    return np.asarray(features, dtype=float), np.asarray(targets, dtype=float)


def train_outcome_model(
    rows: Iterable[dict],
    *,
    test_ratio: float = 0.2,
    random_state: int = 42,
):
    try:
        from sklearn.ensemble import RandomForestRegressor
        from sklearn.metrics import mean_absolute_error
        from sklearn.model_selection import train_test_split
    except ImportError as exc:  # pragma: no cover - runtime dependency check
        raise SystemExit("scikit-learn is required for outcome prediction training") from exc

    X, y = build_outcome_training_data(rows)
    if len(X) < 20:
        raise ValueError("need at least 20 usable rows to train outcome model")

    X_train, X_test, y_train, y_test = train_test_split(
        X,
        y,
        test_size=test_ratio,
        random_state=random_state,
        shuffle=True,
    )

    model = RandomForestRegressor(
        n_estimators=300,
        random_state=random_state,
        n_jobs=-1,
        min_samples_leaf=2,
    )
    model.fit(X_train, y_train)
    predictions = model.predict(X_test)

    mae = float(mean_absolute_error(y_test, predictions))
    directional_accuracy = float(np.mean(np.sign(predictions) == np.sign(y_test)))
    metrics = OutcomeModelMetrics(
        rows_used=int(len(X)),
        train_rows=int(len(X_train)),
        test_rows=int(len(X_test)),
        mae_1d=mae,
        directional_accuracy=directional_accuracy,
    )
    return model, metrics


def predict_outcome(model, row: dict) -> float:
    vector = np.asarray([build_fingerprint(row)], dtype=float)
    prediction = model.predict(vector)
    if isinstance(prediction, np.ndarray):
        return float(prediction[0])
    return float(prediction)


def save_model(model, path: str | Path, metrics: OutcomeModelMetrics | None = None) -> None:
    payload = {"model": model, "metrics": metrics}
    with Path(path).open("wb") as handle:
        pickle.dump(payload, handle)


def load_model(path: str | Path):
    with Path(path).open("rb") as handle:
        payload = pickle.load(handle)
    return payload["model"], payload.get("metrics")


def _safe_float(value: object, default: float = float("nan")) -> float:
    try:
        result = float(value)
    except (TypeError, ValueError):
        return default
    if result != result:
        return default
    return result
