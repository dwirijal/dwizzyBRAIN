from __future__ import annotations

import argparse
import json
import logging
import os
from collections import defaultdict
from typing import Iterable

import pandas as pd
import redis

from quant.anomaly import detect_anomaly
from quant.config import QuantConfig
from quant.indicators import apply_indicators, build_frame, latest_complete_row
from quant.funding import classify_funding_sentiment, extract_funding_rate
from quant.models import OHLCVPayload, QuantSignal
from quant.persistence import SignalStore
from quant.scorer import calculate_quant_score

LOGGER = logging.getLogger(__name__)


def create_redis_client(url: str | None = None) -> redis.Redis:
    redis_url = url or os.getenv("VALKEY_URL", "redis://localhost:6379/0")
    return redis.Redis.from_url(redis_url, decode_responses=True)


def parse_channel(channel: str) -> tuple[str, str, str]:
    prefix = "ch:ohlcv:raw:"
    if not channel.startswith(prefix):
        raise ValueError(f"unsupported channel {channel!r}")

    parts = channel[len(prefix) :].split(":")
    if len(parts) != 3:
        raise ValueError(f"malformed channel {channel!r}")

    return parts[0], parts[1], parts[2]


def deserialize_message(raw_message: dict) -> OHLCVPayload:
    channel = raw_message["channel"]
    symbol, exchange, timeframe = parse_channel(channel)
    payload = json.loads(raw_message["data"])
    payload.setdefault("symbol", symbol)
    payload.setdefault("exchange", exchange)
    payload.setdefault("timeframe", timeframe)
    return OHLCVPayload.from_dict(payload)


def build_signal(payloads: Iterable[OHLCVPayload]) -> QuantSignal | None:
    payload_list = [payload.to_dict() for payload in payloads]
    frame = build_frame(payload_list)
    enriched = apply_indicators(frame)
    latest = latest_complete_row(enriched)
    if latest is None:
        return None

    funding_rate = extract_funding_rate(payload_list[-1]) if payload_list else None
    anomaly, anomaly_type = detect_anomaly(enriched)
    bb_position = float(latest["bb_position"])
    volume_ratio = float(latest["volume_ratio"])
    atr_14 = float(latest["atr_14"])
    atr_ratio_value = latest.get("atr_ratio")
    atr_ratio = 0.0 if pd.isna(atr_ratio_value) else float(atr_ratio_value)

    score = calculate_quant_score(
        rsi=float(latest["rsi"]),
        macd_histogram=float(latest["macd_histogram"]),
        close=float(latest["close"]),
        bb_upper=float(latest["bb_upper"]),
        bb_lower=float(latest["bb_lower"]),
        ema_fast=float(latest["ema_fast"]),
        ema_slow=float(latest["ema_slow"]),
        bb_position=bb_position,
        volume_ratio=volume_ratio,
        atr_ratio=atr_ratio,
        anomaly=anomaly,
        funding_rate=funding_rate,
    )

    return QuantSignal(
        symbol=str(latest["symbol"]),
        exchange=str(latest["exchange"]),
        timeframe=str(latest["timeframe"]),
        timestamp=latest["timestamp"].to_pydatetime(),
        close=float(latest["close"]),
        quant_score=score,
        rsi_14=float(latest["rsi"]),
        macd=float(latest["macd"]),
        macd_signal=float(latest["macd_signal"]),
        macd_histogram=float(latest["macd_histogram"]),
        bb_upper=float(latest["bb_upper"]),
        bb_middle=float(latest["bb_middle"]),
        bb_lower=float(latest["bb_lower"]),
        ema_fast=float(latest["ema_fast"]),
        ema_slow=float(latest["ema_slow"]),
        bb_position=bb_position,
        atr_14=atr_14,
        volume_ratio=volume_ratio,
        funding_rate=funding_rate,
        funding_sentiment=classify_funding_sentiment(funding_rate),
        anomaly=anomaly,
        anomaly_type=anomaly_type,
    )


class QuantWorker:
    def __init__(
        self,
        client: redis.Redis,
        window_size: int = 250,
        signal_store: SignalStore | None = None,
        publish_signals: bool = True,
    ) -> None:
        self.client = client
        self.window_size = window_size
        self.signal_store = signal_store
        self.publish_signals = publish_signals
        self._buffers: dict[tuple[str, str, str], list[OHLCVPayload]] = defaultdict(list)

    def subscribe(self) -> redis.client.PubSub:
        pubsub = self.client.pubsub()
        pubsub.psubscribe("ch:ohlcv:raw:*")
        return pubsub

    def handle_raw_message(self, raw_message: dict) -> QuantSignal | None:
        payload = deserialize_message(raw_message)
        key = (payload.symbol, payload.exchange, payload.timeframe)
        buffer = self._buffers[key]
        buffer.append(payload)
        if len(buffer) > self.window_size:
            del buffer[:-self.window_size]

        return build_signal(buffer)

    def publish_signal(self, signal: QuantSignal) -> int:
        if self.signal_store is not None:
            self.signal_store.save(signal)

        if not self.publish_signals:
            return 0

        channel = f"ch:signal:processed:{signal.symbol}"
        return int(self.client.publish(channel, json.dumps(signal.to_dict())))

    def run(self) -> None:
        pubsub = self.subscribe()
        LOGGER.info("quant worker subscribed to ch:ohlcv:raw:*")

        for raw_message in pubsub.listen():
            if raw_message.get("type") not in {"pmessage", "message"}:
                continue

            signal = self.handle_raw_message(raw_message)
            if signal is None:
                continue

            self.publish_signal(signal)
            LOGGER.info(
                "published quant signal",
                extra={
                    "symbol": signal.symbol,
                    "exchange": signal.exchange,
                    "timeframe": signal.timeframe,
                    "score": signal.quant_score,
                },
            )


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant")
    parser.add_argument("--healthcheck", action="store_true", help="print a health payload and exit")
    args = parser.parse_args()

    if args.healthcheck:
        print(json.dumps({"ok": True, "service": "quant"}))
        return

    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
    config = QuantConfig.from_env()
    client = create_redis_client(config.valkey_url)
    signal_store = SignalStore(client, ttl_seconds=config.signal_ttl_seconds, prefix=config.cache_prefix)
    QuantWorker(
        client,
        window_size=config.window_size,
        signal_store=signal_store,
        publish_signals=config.publish_signals,
    ).run()


if __name__ == "__main__":
    main()
