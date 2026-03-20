from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from datetime import UTC, datetime, timedelta
from html import unescape
from io import StringIO
from typing import Iterable, Sequence
from urllib.request import urlopen

import cloudscraper
import numpy as np
import pandas as pd


FRED_SERIES_CATALOG: dict[str, tuple[str, str, int]] = {
    "FEDFUNDS": ("rate", "Effective Federal Funds Rate", 3),
    "DFF": ("rate", "Daily Fed Funds Rate", 3),
    "CPIAUCSL": ("cpi", "CPI All Urban Consumers", 3),
    "GDP": ("growth", "Gross Domestic Product", 2),
    "PCEPI": ("inflation", "PCE Price Index", 2),
    "PAYEMS": ("employment", "Total Nonfarm Payrolls", 2),
    "PPIACO": ("inflation", "PPI All Commodities", 2),
}

PROXIMITY_WINDOWS: dict[str, tuple[float, float] | None] = {
    "imminent": (-4.0, 0.0),
    "pre_near": (-24.0, -4.0),
    "pre_far": (-72.0, -24.0),
    "post_near": (0.0, 4.0),
    "post_mid": (4.0, 24.0),
    "post_far": (24.0, 72.0),
    "neutral": None,
}


@dataclass(slots=True)
class MacroPoint:
    series_id: str
    series_name: str
    timestamp: datetime
    value: float
    source: str = "fred"
    event_type: str = "macro"
    importance: int = 1
    metadata: dict[str, object] = field(default_factory=dict)


@dataclass(slots=True)
class EventSurprise:
    label: str = "no_surprise"
    value: float = 0.0


def fetch_fred_series(series_id: str) -> list[MacroPoint]:
    normalized = series_id.strip().upper()
    if not normalized:
        return []

    catalog = FRED_SERIES_CATALOG.get(normalized)
    if catalog is None:
        catalog = ("macro", normalized.replace("_", " ").title(), 1)

    event_type, series_name, importance = catalog
    url = f"https://fred.stlouisfed.org/graph/fredgraph.csv?id={normalized}"
    with urlopen(url, timeout=30) as response:
        csv_text = response.read().decode("utf-8")

    frame = pd.read_csv(StringIO(csv_text))
    if frame.empty:
        return []

    date_column = frame.columns[0]
    value_column = normalized if normalized in frame.columns else frame.columns[-1]
    frame = frame[[date_column, value_column]].dropna()
    frame[value_column] = pd.to_numeric(frame[value_column], errors="coerce")
    frame = frame.dropna()

    points: list[MacroPoint] = []
    for _, row in frame.iterrows():
        timestamp = pd.Timestamp(row[date_column]).tz_localize(UTC)
        points.append(
            MacroPoint(
                series_id=normalized,
                series_name=series_name,
                timestamp=timestamp.to_pydatetime(),
                value=float(row[value_column]),
                event_type=event_type,
                importance=importance,
            )
        )
    return points


def fetch_forex_factory_calendar(week: str | None = None) -> list[MacroPoint]:
    url = "https://www.forexfactory.com/calendar"
    if week:
        url = f"{url}?week={week.strip()}"

    scraper = cloudscraper.create_scraper(browser={"browser": "chrome", "platform": "windows", "mobile": False})
    response = scraper.get(url, timeout=30)
    response.raise_for_status()

    days = extract_forex_factory_days(response.text)
    return parse_forex_factory_days(days)


def extract_forex_factory_days(html: str) -> list[dict]:
    needle = "days: ["
    start = html.find(needle)
    if start == -1:
        raise ValueError("could not locate ForexFactory days payload")

    end = _find_matching_bracket(html, start + len("days: ") - 1, open_bracket="[", close_bracket="]")
    blob = html[start + len("days: "):end]
    return json.loads(blob)


def parse_forex_factory_days(days: Sequence[dict]) -> list[MacroPoint]:
    points: list[MacroPoint] = []
    for day in days:
        day_date = str(day.get("date", "")).strip()
        day_dateline = int(day.get("dateline") or 0)
        for event in day.get("events", []):
            name = str(event.get("name", "")).strip()
            currency = str(event.get("currency", "")).strip().upper()
            if not name or not currency:
                continue

            timestamp = datetime.fromtimestamp(int(event.get("dateline") or day_dateline), tz=UTC)
            actual = _parse_numeric_value(event.get("actual"))
            forecast = _parse_numeric_value(event.get("forecast"))
            previous = _parse_numeric_value(event.get("previous"))
            revision = _parse_numeric_value(event.get("revision"))
            value = actual
            if value is None:
                value = forecast
            if value is None:
                value = previous
            if value is None:
                value = 0.0

            metadata = {
                "event_id": event.get("id"),
                "ebase_id": event.get("ebaseId"),
                "country": event.get("country"),
                "currency": currency,
                "impact_name": event.get("impactName"),
                "impact_title": event.get("impactTitle"),
                "time_label": event.get("timeLabel"),
                "time_masked": bool(event.get("timeMasked")),
                "actual_raw": event.get("actual"),
                "forecast_raw": event.get("forecast"),
                "previous_raw": event.get("previous"),
                "revision_raw": event.get("revision"),
                "actual_value": actual,
                "forecast_value": forecast,
                "previous_value": previous,
                "revision_value": revision,
                "date": event.get("date") or day_date,
                "url": event.get("url"),
                "solo_url": event.get("soloUrl"),
                "day_dateline": day_dateline,
            }
            points.append(
                MacroPoint(
                    series_id=f"FF:{currency}:{_slugify(name)}",
                    series_name=name,
                    timestamp=timestamp,
                    value=float(value),
                    source="forexfactory",
                    event_type="calendar",
                    importance=_importance_from_impact(str(event.get("impactName", "")).strip().lower()),
                    metadata=metadata,
                )
            )
    return points


def classify_rate_regime(fed_rate: float) -> str:
    if fed_rate >= 5.0:
        return "very_high"
    if fed_rate >= 3.0:
        return "high"
    if fed_rate >= 1.0:
        return "medium"
    return "low_zirp"


def classify_rate_direction(rate_series: Sequence[float]) -> str:
    values = [float(value) for value in rate_series if value == value]
    if len(values) < 2:
        return "paused"
    delta = values[-1] - values[0]
    if delta > 0:
        return "hiking"
    if delta < 0:
        return "cutting"
    return "paused"


def classify_cpi_trend(cpi_series: Sequence[float]) -> str:
    values = [float(value) for value in cpi_series if value == value]
    if len(values) < 3:
        return "stable"
    x = np.arange(len(values[-3:]), dtype="float64")
    slope = float(np.polyfit(x, np.asarray(values[-3:], dtype="float64"), 1)[0])
    if slope > 0.2:
        return "accelerating"
    if slope > -0.2:
        return "stable"
    return "cooling"


def classify_surprise(actual: float | None, forecast: float | None, std_history: float) -> str:
    if forecast is None:
        return "no_surprise"
    deviation = (actual - forecast) / (std_history + 1e-9) if actual is not None else 0.0
    if deviation > 1.5:
        return "massive_beat"
    if deviation > 0.5:
        return "beat"
    if deviation > -0.5:
        return "inline"
    if deviation > -1.5:
        return "miss"
    return "massive_miss"


def classify_pre_drift(change_24h: float) -> str:
    if change_24h > 0.03:
        return "strong_bullish_drift"
    if change_24h > 0.01:
        return "mild_bullish_drift"
    if change_24h > -0.01:
        return "sideways"
    if change_24h > -0.03:
        return "mild_bearish_drift"
    return "strong_bearish_drift"


def classify_vol_context(atr_ratio: float) -> str:
    if atr_ratio > 2.0:
        return "extreme_vol"
    if atr_ratio > 1.5:
        return "high_vol"
    if atr_ratio > 0.8:
        return "normal_vol"
    return "low_vol_compression"


def build_macro_environment(labels: dict[str, object]) -> str:
    parts = [
        str(labels.get("rate_direction", "")).strip(),
        f"{labels.get('rate_regime', '')}_rates".strip("_"),
        f"cpi_{labels.get('cpi_trend', '')}".strip("_"),
        str(labels.get("proximity_label", "neutral")).strip(),
        f"{labels.get('last_surprise_label', 'no_surprise')}_surprise".strip("_"),
    ]
    return "|".join(part for part in parts if part)


def label_candle_events(
    candles: Sequence[dict],
    macro_points: Sequence[MacroPoint],
) -> list[dict]:
    normalized_candles = _normalize_candles(candles)
    if not normalized_candles:
        return []

    series_lookup: dict[str, list[MacroPoint]] = {}
    all_points: list[MacroPoint] = []
    for point in sorted(macro_points, key=lambda item: item.timestamp):
        series_lookup.setdefault(point.series_id, []).append(point)
        all_points.append(point)

    labels: list[dict] = []
    for candle in normalized_candles:
        candle_time = candle["timestamp"]
        rate_points = _series_values_before(series_lookup, ("FEDFUNDS", "DFF"), candle_time)
        cpi_points = _series_values_before(series_lookup, ("CPIAUCSL",), candle_time)
        nearest_point = _nearest_point(all_points, candle_time)
        hours_to_event = _hours_delta(nearest_point.timestamp, candle_time) if nearest_point else 0.0

        rate_value = rate_points[-1].value if rate_points else 0.0
        atr_ratio = candle.get("atr_ratio")
        if atr_ratio is None:
            atr_ratio = 1.0
        surprise = _event_surprise(nearest_point)

        label = {
            "timestamp": candle_time,
            "symbol": candle["symbol"],
            "timeframe": candle["timeframe"],
            "rate_direction": classify_rate_direction([point.value for point in rate_points[-3:]]),
            "rate_regime": classify_rate_regime(rate_value),
            "cpi_trend": classify_cpi_trend([point.value for point in cpi_points[-3:]]),
            "proximity_label": _classify_proximity(hours_to_event),
            "last_surprise_label": surprise.label,
            "last_surprise_value": surprise.value,
            "hours_to_event": hours_to_event,
            "hours_from_event": -hours_to_event,
            "vol_context": classify_vol_context(float(atr_ratio)),
            "nearest_event_series_id": nearest_point.series_id if nearest_point else "",
            "nearest_event_time": nearest_point.timestamp if nearest_point else None,
        }
        label["macro_environment"] = build_macro_environment(label)
        labels.append(label)

    return labels


def _normalize_candles(candles: Sequence[dict]) -> list[dict]:
    normalized: list[dict] = []
    for candle in candles:
        timestamp = candle.get("timestamp")
        if timestamp is None:
            continue
        if isinstance(timestamp, str):
            parsed = pd.to_datetime(timestamp, utc=True)
        else:
            parsed = pd.Timestamp(timestamp, tz="UTC")
        normalized.append(
            {
                "timestamp": parsed.to_pydatetime(),
                "symbol": str(candle.get("symbol", "")).strip(),
                "timeframe": str(candle.get("timeframe", "")).strip().lower(),
                "atr_ratio": candle.get("atr_ratio"),
            }
        )
    return sorted(normalized, key=lambda row: row["timestamp"])


def _series_values_before(
    lookup: dict[str, list[MacroPoint]],
    series_ids: Sequence[str],
    cutoff: datetime,
) -> list[MacroPoint]:
    values: list[MacroPoint] = []
    for series_id in series_ids:
        series = lookup.get(series_id, [])
        values.extend([point for point in series if point.timestamp <= cutoff])
    values.sort(key=lambda point: point.timestamp)
    return values


def _nearest_point(points: Sequence[MacroPoint], cutoff: datetime) -> MacroPoint | None:
    if not points:
        return None
    return min(points, key=lambda point: abs(_hours_delta(cutoff, point.timestamp)))


def _classify_proximity(hours_to_event: float) -> str:
    for label, window in PROXIMITY_WINDOWS.items():
        if window is None:
            continue
        lower, upper = window
        if lower <= hours_to_event < upper:
            return label
    return "neutral"


def _hours_delta(start: datetime, end: datetime) -> float:
    return (end - start).total_seconds() / 3600.0


def _event_surprise(point: MacroPoint | None) -> EventSurprise:
    if point is None or point.source != "forexfactory":
        return EventSurprise()

    actual = point.metadata.get("actual_value")
    forecast = point.metadata.get("forecast_value")
    if actual is None or forecast is None:
        return EventSurprise()

    actual_value = float(actual)
    forecast_value = float(forecast)
    baseline = max(abs(forecast_value) * 0.05, 1.0)
    return EventSurprise(
        label=classify_surprise(actual_value, forecast_value, baseline),
        value=actual_value - forecast_value,
    )


def _importance_from_impact(impact_name: str) -> int:
    match impact_name:
        case "high":
            return 3
        case "medium":
            return 2
        case "low":
            return 1
        case _:
            return 0


def _parse_numeric_value(value: object) -> float | None:
    if value is None:
        return None
    text = str(value).strip()
    if not text or text.lower() in {"n/a", "na", "-"}:
        return None

    primary = text.split("|", 1)[0].strip()
    match = re.search(r"([-+]?\d[\d,]*(?:\.\d+)?)\s*([KMBT%]?)", primary)
    if not match:
        return None

    number = float(match.group(1).replace(",", ""))
    suffix = match.group(2).upper()
    scale = {
        "": 1.0,
        "%": 1.0,
        "K": 1_000.0,
        "M": 1_000_000.0,
        "B": 1_000_000_000.0,
        "T": 1_000_000_000_000.0,
    }.get(suffix, 1.0)
    return number * scale


def _slugify(value: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", value.strip().lower())
    return slug.strip("-")


def _find_matching_bracket(text: str, start: int, *, open_bracket: str, close_bracket: str) -> int:
    depth = 0
    in_string = False
    escape = False
    for index in range(start, len(text)):
        ch = text[index]
        if in_string:
            if escape:
                escape = False
            elif ch == "\\":
                escape = True
            elif ch == '"':
                in_string = False
        else:
            if ch == '"':
                in_string = True
            elif ch == open_bracket:
                depth += 1
            elif ch == close_bracket:
                depth -= 1
                if depth == 0:
                    return index + 1
    raise ValueError(f"unmatched bracket {open_bracket}{close_bracket}")
