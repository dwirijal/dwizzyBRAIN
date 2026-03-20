# Quant Service

`quant/` is the standalone Python intelligence worker for `dwizzyBRAIN`.
It subscribes to raw OHLCV candles, computes signals, backfills history, labels macro events, and builds pattern embeddings.

## Status

- Realtime quant loop: done
- Indicator catalog: done
- Backfill pipeline: done
- Macro event labelling: done
- Pattern engine: done
- Consumer API / bot integration: done
- Optional extras like CryptoPanic: partial / optional

## Local Setup

Use the project venv, not the system Python.

```bash
cd quant
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"
```

If the venv already exists, just activate it.

## Run

Healthcheck:

```bash
python -m quant.main --healthcheck
```

Worker mode:

```bash
python -m quant.main
```

Backfill helpers:

```bash
python -m quant.backfill.fetch_ohlcv --help
python -m quant.backfill.compute_bulk --help
python -m quant.backfill.fetch_yfinance --help
python -m quant.backfill.fetch_fred --help
python -m quant.backfill.label_events --help
python -m quant.backfill.build_vectors --help
python -m quant.backfill.search_patterns --help
python -m quant.backfill.train_outcome_model --help
```

## Verification

Preferred verification commands:

```bash
python -m pytest -q tests
python -m compileall src
```

The repository-level `pytest` may fail if the quant dependencies are not installed in the active interpreter.

## API Surface

The API layer in `dwizzyBRAIN` currently exposes:

- `GET /v1/quant/pattern`
- `GET /v1/quant/signals`
- `GET /v1/quant/signals/latest`
- `GET /v1/quant/signals/summary`

The signal endpoints read from the live `signals` table and are keyed by `symbol` + `timeframe`, with optional `exchange`.

## Environment

Required runtime variables:

- `VALKEY_URL`
- `POSTGRES_URL`
- `QUANT_WINDOW_SIZE`
- `QUANT_SIGNAL_TTL_SECONDS`
- `QUANT_PUBLISH_SIGNALS`
- `QUANT_CACHE_PREFIX`
- `QUANT_SIGNAL_CHANNEL_PREFIX`

Optional event/backfill variables:

- `FRED_API_KEY`
- `FOREX_FACTORY_BASE_URL`

## Notes

- `quant/` is intentionally separate from `engine/`.
- `quant` talks to the rest of `dwizzyBRAIN` only through Valkey, Postgres, and the public API surface.
- Use `*_FILE` secrets in production if the environment provides them.
