from __future__ import annotations

from quant.config import QuantConfig


def test_quant_config_reads_defaults(monkeypatch) -> None:
    monkeypatch.delenv("VALKEY_URL", raising=False)
    monkeypatch.delenv("POSTGRES_URL", raising=False)
    monkeypatch.delenv("QUANT_WINDOW_SIZE", raising=False)
    monkeypatch.delenv("QUANT_SIGNAL_TTL_SECONDS", raising=False)
    monkeypatch.delenv("QUANT_PUBLISH_SIGNALS", raising=False)
    monkeypatch.delenv("QUANT_CACHE_PREFIX", raising=False)
    monkeypatch.delenv("QUANT_SIGNAL_CHANNEL_PREFIX", raising=False)

    config = QuantConfig.from_env()

    assert config.valkey_url.startswith("redis://")
    assert config.postgres_url == ""
    assert config.window_size == 250
    assert config.signal_ttl_seconds == 3600
    assert config.publish_signals is True
    assert config.cache_prefix == "signal"
    assert config.signal_channel_prefix == "ch:signal:processed"
