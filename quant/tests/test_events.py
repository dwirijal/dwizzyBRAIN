from __future__ import annotations

from datetime import UTC, datetime
from io import BytesIO

from quant.events import (
    MacroPoint,
    extract_forex_factory_days,
    build_macro_environment,
    classify_cpi_trend,
    classify_surprise,
    classify_rate_direction,
    classify_rate_regime,
    classify_vol_context,
    fetch_fred_series,
    parse_forex_factory_days,
    label_candle_events,
)


def test_fetch_fred_series_parses_public_csv(monkeypatch) -> None:
    sample_csv = b"DATE,FEDFUNDS\n2026-01-01,5.25\n2026-02-01,5.50\n"

    class _Response(BytesIO):
        def __enter__(self):
            return self

        def __exit__(self, exc_type, exc, tb):
            return False

    monkeypatch.setattr("quant.events.urlopen", lambda url, timeout=30: _Response(sample_csv))

    points = fetch_fred_series("FEDFUNDS")

    assert len(points) == 2
    assert points[0].series_id == "FEDFUNDS"
    assert points[0].series_name == "Effective Federal Funds Rate"
    assert points[0].value == 5.25


def test_label_candle_events_builds_macro_environment() -> None:
    candle_time = datetime(2026, 3, 18, 12, tzinfo=UTC)
    macro_points = [
        MacroPoint("FEDFUNDS", "Effective Federal Funds Rate", datetime(2026, 1, 1, tzinfo=UTC), 5.0),
        MacroPoint("FEDFUNDS", "Effective Federal Funds Rate", datetime(2026, 2, 1, tzinfo=UTC), 5.25),
        MacroPoint("FEDFUNDS", "Effective Federal Funds Rate", datetime(2026, 3, 18, 15, tzinfo=UTC), 5.5),
        MacroPoint("CPIAUCSL", "CPI All Urban Consumers", datetime(2026, 1, 1, tzinfo=UTC), 320.0),
        MacroPoint("CPIAUCSL", "CPI All Urban Consumers", datetime(2026, 2, 1, tzinfo=UTC), 318.0),
        MacroPoint("CPIAUCSL", "CPI All Urban Consumers", datetime(2026, 3, 1, tzinfo=UTC), 317.0),
    ]

    labels = label_candle_events(
        [
            {
                "symbol": "BTCUSDT",
                "timeframe": "4h",
                "timestamp": candle_time.isoformat().replace("+00:00", "Z"),
                "atr_ratio": 1.7,
            }
        ],
        macro_points,
    )

    assert len(labels) == 1
    label = labels[0]
    assert label["rate_regime"] == "very_high"
    assert label["rate_direction"] == "hiking"
    assert label["cpi_trend"] == "cooling"
    assert label["proximity_label"] == "imminent"
    assert label["vol_context"] == "high_vol"
    assert label["macro_environment"] == "hiking|very_high_rates|cpi_cooling|imminent|no_surprise_surprise"


def test_macro_label_helpers_cover_expected_ranges() -> None:
    assert classify_rate_regime(5.5) == "very_high"
    assert classify_rate_regime(3.5) == "high"
    assert classify_rate_direction([1.0, 1.5, 2.0]) == "hiking"
    assert classify_rate_direction([2.0, 1.5, 1.0]) == "cutting"
    assert classify_cpi_trend([10.0, 9.5, 9.0]) == "cooling"
    assert classify_vol_context(2.1) == "extreme_vol"
    assert build_macro_environment(
        {
            "rate_direction": "paused",
            "rate_regime": "high",
            "cpi_trend": "stable",
            "proximity_label": "neutral",
            "last_surprise_label": "inline",
        }
    ) == "paused|high_rates|cpi_stable|neutral|inline_surprise"


def test_extract_forex_factory_days_parses_structured_js_blob() -> None:
    html = """
    <script>
    window.calendarComponentStates[1] = {
      days: [{"date":"Tue <span>Apr 8<\\/span>","dateline":1744045200,"add":"","events":[{"id":141677,"ebaseId":341,"name":"Consumer Credit m\\/m","currency":"USD","country":"US","impactName":"low","impactTitle":"Low Impact Expected","timeLabel":"2:00am","timeMasked":false,"actual":"-0.8B","forecast":"14.9B","previous":"18.1B","revision":"8.9B","date":"Apr 8, 2025","url":"\\/calendar?day=apr8.2025#detail=141677","soloUrl":"\\/calendar\\/341-us-consumer-credit-m-m"}]}]
    };
    </script>
    """

    days = extract_forex_factory_days(html)

    assert len(days) == 1
    assert days[0]["events"][0]["name"] == "Consumer Credit m/m"


def test_parse_forex_factory_days_builds_macro_points() -> None:
    days = [
        {
            "date": "Tue <span>Apr 8</span>",
            "dateline": 1744045200,
            "add": "",
            "events": [
                {
                    "id": 141677,
                    "ebaseId": 341,
                    "name": "Consumer Credit m/m",
                    "currency": "USD",
                    "country": "US",
                    "impactName": "low",
                    "impactTitle": "Low Impact Expected",
                    "timeLabel": "2:00am",
                    "timeMasked": False,
                    "actual": "-0.8B",
                    "forecast": "14.9B",
                    "previous": "18.1B",
                    "revision": "8.9B",
                    "date": "Apr 8, 2025",
                    "url": "/calendar?day=apr8.2025#detail=141677",
                    "soloUrl": "/calendar/341-us-consumer-credit-m-m",
                }
            ],
        }
    ]

    points = parse_forex_factory_days(days)

    assert len(points) == 1
    point = points[0]
    assert point.series_id == "FF:USD:consumer-credit-m-m"
    assert point.importance == 1
    assert point.value == -800000000.0
    assert point.metadata["actual_value"] == -800000000.0
    assert point.metadata["forecast_value"] == 14900000000.0


def test_label_candle_events_uses_forex_factory_surprise_context() -> None:
    candle_time = datetime(2026, 3, 18, 12, tzinfo=UTC)
    macro_points = [
        MacroPoint(
            "FF:USD:cpi-m-m",
            "CPI m/m",
            datetime(2026, 3, 18, 15, tzinfo=UTC),
            110.0,
            source="forexfactory",
            event_type="calendar",
            importance=3,
            metadata={"actual_value": 110.0, "forecast_value": 100.0},
        )
    ]

    labels = label_candle_events(
        [
            {
                "symbol": "BTCUSDT",
                "timeframe": "4h",
                "timestamp": candle_time.isoformat().replace("+00:00", "Z"),
                "atr_ratio": 1.7,
            }
        ],
        macro_points,
    )

    assert len(labels) == 1
    label = labels[0]
    assert label["proximity_label"] == "imminent"
    assert label["last_surprise_label"] == "massive_beat"
    assert label["last_surprise_value"] == 10.0
    assert "imminent" in label["macro_environment"]
    assert classify_surprise(110.0, 100.0, 5.0) == "massive_beat"
