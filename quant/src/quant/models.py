from __future__ import annotations

from dataclasses import asdict, dataclass
from datetime import UTC, datetime


def _coerce_timestamp(value: str | datetime) -> datetime:
    if isinstance(value, datetime):
        return value.astimezone(UTC)

    normalized = value.replace("Z", "+00:00")
    return datetime.fromisoformat(normalized).astimezone(UTC)


@dataclass(slots=True)
class OHLCVPayload:
    symbol: str
    exchange: str
    timeframe: str
    timestamp: datetime
    open: float
    high: float
    low: float
    close: float
    volume: float
    funding_rate: float | None = None

    @classmethod
    def from_dict(cls, data: dict) -> "OHLCVPayload":
        return cls(
            symbol=data["symbol"],
            exchange=data["exchange"],
            timeframe=data["timeframe"],
            timestamp=_coerce_timestamp(data["timestamp"]),
            open=float(data["open"]),
            high=float(data["high"]),
            low=float(data["low"]),
            close=float(data["close"]),
            volume=float(data["volume"]),
            funding_rate=_optional_float(data.get("funding_rate") or data.get("fundingRate") or data.get("funding")),
        )

    def to_dict(self) -> dict:
        payload = asdict(self)
        payload["timestamp"] = self.timestamp.astimezone(UTC).isoformat().replace("+00:00", "Z")
        return payload


@dataclass(slots=True)
class QuantSignal:
    symbol: str
    exchange: str
    timeframe: str
    timestamp: datetime
    close: float
    quant_score: float
    rsi_14: float
    macd: float
    macd_signal: float
    macd_histogram: float
    bb_upper: float
    bb_middle: float
    bb_lower: float
    ema_fast: float
    ema_slow: float
    bb_position: float
    atr_14: float
    volume_ratio: float
    funding_rate: float | None = None
    funding_sentiment: str | None = None
    anomaly: bool = False
    anomaly_type: str | None = None

    def to_dict(self) -> dict:
        payload = asdict(self)
        payload["timestamp"] = self.timestamp.astimezone(UTC).isoformat().replace("+00:00", "Z")
        return payload


def _optional_float(value: object) -> float | None:
    if value in (None, ""):
        return None
    try:
        return float(value)
    except (TypeError, ValueError):
        return None
