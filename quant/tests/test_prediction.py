from __future__ import annotations

import math
from datetime import UTC, datetime, timedelta

from quant.prediction import build_outcome_training_data, load_model, predict_outcome, save_model, train_outcome_model


def _rows(count: int = 80) -> list[dict]:
    base = datetime(2026, 3, 18, tzinfo=UTC)
    rows: list[dict] = []
    for index in range(count):
        close = 100.0 + index
        rows.append(
            {
                "time": base + timedelta(days=index),
                "symbol": "SPY",
                "timeframe": "1d",
                "close": close,
                "close_1d_later": close * 1.01,
                "dist_from_ema9": index * 0.1,
                "dist_from_ema21": index * 0.2,
                "dist_from_ema50": index * 0.3,
                "dist_from_ema200": index * 0.4,
                "adx": 25.0 + (index % 5),
                "supertrend_dir": 1,
                "rsi_14": 50.0 + (index % 20),
                "rsi_2": 50.0,
                "macd_hist": 0.1 * index,
                "macd_hist_slope": 0.01 * index,
                "stoch_k": 50.0,
                "cci_20": 0.0,
                "bb_position": 0.5,
                "bb_width": 0.2,
                "atr_ratio": 1.0,
                "hist_vol_20": 10.0,
                "kc_position": 0.5,
                "volume_ratio": 1.0,
                "volume_trend": 1.0,
                "cmf_20": 0.0,
                "obv_slope": 0.0,
                "candle_body_pct": 0.4,
                "upper_wick_pct": 0.2,
                "lower_wick_pct": 0.2,
                "dist_from_vwap": 0.1,
                "rsi_slope": 0.0,
                "hours_to_event": 24.0,
                "last_surprise_value": 0.0,
                "rate_regime": "medium",
                "vol_context": "normal",
            }
        )
    return rows


def test_build_outcome_training_data_shapes() -> None:
    X, y = build_outcome_training_data(_rows())

    assert X.shape[0] == y.shape[0]
    assert X.shape[1] == 30
    assert y.shape[0] > 0


def test_train_outcome_model_roundtrip(tmp_path) -> None:
    model, metrics = train_outcome_model(_rows())

    assert metrics.rows_used > 0
    assert 0.0 <= metrics.directional_accuracy <= 1.0
    assert not math.isnan(metrics.mae_1d)

    sample = _rows()[0]
    prediction = predict_outcome(model, sample)
    assert isinstance(prediction, float)

    path = tmp_path / "outcome.pkl"
    save_model(model, path, metrics)
    loaded_model, loaded_metrics = load_model(path)

    assert loaded_metrics is not None
    assert loaded_metrics.rows_used == metrics.rows_used
    assert isinstance(predict_outcome(loaded_model, sample), float)
