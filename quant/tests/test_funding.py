from __future__ import annotations

from quant.funding import classify_funding_sentiment, extract_funding_rate


def test_extract_funding_rate_supports_common_keys() -> None:
    payload = {"fundingRate": "0.0009"}
    assert extract_funding_rate(payload) == 0.0009


def test_classify_funding_sentiment() -> None:
    assert classify_funding_sentiment(0.001) == "bearish"
    assert classify_funding_sentiment(-0.001) == "bullish"
    assert classify_funding_sentiment(0.0) == "neutral"
    assert classify_funding_sentiment(None) is None
