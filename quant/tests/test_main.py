from __future__ import annotations

import json
from datetime import UTC, datetime, timedelta

import fakeredis

from quant.main import QuantWorker, build_signal, deserialize_message, parse_channel
from quant.models import OHLCVPayload
from quant.persistence import SignalStore


def _message(index: int) -> dict:
    timestamp = datetime(2026, 3, 18, tzinfo=UTC) + timedelta(minutes=index)
    payload = {
        "timestamp": timestamp.isoformat().replace("+00:00", "Z"),
        "open": 100 + index,
        "high": 101 + index,
        "low": 99 + index,
        "close": 100.5 + index,
        "volume": 1000 + index,
    }
    return {
        "type": "pmessage",
        "pattern": "ch:ohlcv:raw:*",
        "channel": "ch:ohlcv:raw:BTCUSDT:binance:1m",
        "data": json.dumps(payload),
    }


def test_parse_channel() -> None:
    assert parse_channel("ch:ohlcv:raw:BTCUSDT:binance:1m") == ("BTCUSDT", "binance", "1m")


def test_deserialize_message_infers_channel_fields() -> None:
    payload = deserialize_message(_message(0))

    assert payload.symbol == "BTCUSDT"
    assert payload.exchange == "binance"
    assert payload.timeframe == "1m"


def test_build_signal_returns_none_without_enough_history() -> None:
    payloads = [deserialize_message(_message(index)) for index in range(5)]
    assert build_signal(payloads) is None


def test_worker_emits_processed_signal() -> None:
    client = fakeredis.FakeRedis(decode_responses=True)
    worker = QuantWorker(client, signal_store=SignalStore(client, ttl_seconds=120))

    signal = None
    for index in range(40):
        signal = worker.handle_raw_message(_message(index))

    assert signal is not None
    published = worker.publish_signal(signal)
    assert published == 0

    buffered = worker._buffers[("BTCUSDT", "binance", "1m")]
    assert isinstance(buffered[-1], OHLCVPayload)
    cached = worker.signal_store.load("BTCUSDT", "1m")
    assert cached is not None
    assert cached["symbol"] == "BTCUSDT"
    assert cached["timeframe"] == "1m"
    assert cached["anomaly"] in {True, False}
