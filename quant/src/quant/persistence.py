from __future__ import annotations

import json
from dataclasses import dataclass

import redis


@dataclass(slots=True)
class SignalStore:
    client: redis.Redis
    ttl_seconds: int = 3600
    prefix: str = "signal"

    def key_for(self, symbol: str, timeframe: str) -> str:
        return f"{self.prefix}:{symbol}:{timeframe}"

    def save(self, signal: object) -> None:
        payload = signal.to_dict()  # type: ignore[attr-defined]
        key = self.key_for(payload["symbol"], payload["timeframe"])
        self.client.set(key, json.dumps(payload), ex=self.ttl_seconds)

    def load(self, symbol: str, timeframe: str) -> dict | None:
        raw = self.client.get(self.key_for(symbol, timeframe))
        if not raw:
            return None
        return json.loads(raw)
