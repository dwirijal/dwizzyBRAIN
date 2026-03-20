from __future__ import annotations

import os
from dataclasses import dataclass


def _parse_bool(value: str | None, default: bool) -> bool:
    normalized = (value or "").strip().lower()
    if not normalized:
        return default
    if normalized in {"1", "true", "yes", "on"}:
        return True
    if normalized in {"0", "false", "no", "off"}:
        return False
    return default


def _parse_int(value: str | None, default: int) -> int:
    normalized = (value or "").strip()
    if not normalized:
        return default
    try:
        parsed = int(normalized)
    except ValueError:
        return default
    return parsed if parsed > 0 else default


@dataclass(slots=True)
class QuantConfig:
    valkey_url: str
    postgres_url: str
    window_size: int = 250
    signal_ttl_seconds: int = 3600
    publish_signals: bool = True
    cache_prefix: str = "signal"
    signal_channel_prefix: str = "ch:signal:processed"

    @classmethod
    def from_env(cls) -> "QuantConfig":
        return cls(
            valkey_url=_read_secret("VALKEY_URL", default="redis://localhost:6379/0"),
            postgres_url=_read_secret("POSTGRES_URL", default=""),
            window_size=_parse_int(os.getenv("QUANT_WINDOW_SIZE"), 250),
            signal_ttl_seconds=_parse_int(os.getenv("QUANT_SIGNAL_TTL_SECONDS"), 3600),
            publish_signals=_parse_bool(os.getenv("QUANT_PUBLISH_SIGNALS"), True),
            cache_prefix=(os.getenv("QUANT_CACHE_PREFIX") or "signal").strip() or "signal",
            signal_channel_prefix=(os.getenv("QUANT_SIGNAL_CHANNEL_PREFIX") or "ch:signal:processed").strip()
            or "ch:signal:processed",
        )


def _read_secret(name: str, default: str) -> str:
    value = (os.getenv(name) or "").strip()
    if value:
        return value

    file_name = (os.getenv(f"{name}_FILE") or "").strip()
    if not file_name:
        return default

    try:
        with open(file_name, "r", encoding="utf-8") as handle:
            return handle.read().strip() or default
    except OSError:
        return default
