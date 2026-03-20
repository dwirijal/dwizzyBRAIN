SHELL := /bin/bash

GO_COVER_MIN ?= 35.0
QUANT_COVER_MIN ?= 60
API_CMD_COVER_MIN ?= 9.0
API_AUTH_COVER_MIN ?= 20.0
API_DEFI_COVER_MIN ?= 20.0
API_HANDLER_COVER_MIN ?= 25.0
API_MARKET_COVER_MIN ?= 20.0
API_NEWS_COVER_MIN ?= 15.0

.PHONY: fmt-check lint test test-go test-quant coverage coverage-go coverage-go-critical coverage-quant build quality

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt found unformatted files" && gofmt -l . && exit 1)

lint:
	golangci-lint run ./...

test:
	$(MAKE) test-go
	$(MAKE) test-quant

test-go:
	go test ./...

test-quant:
	@if [ -x quant/.venv/bin/pytest ]; then \
		quant/.venv/bin/pytest -q quant/tests; \
	else \
		cd quant && pytest -q; \
	fi

coverage:
	$(MAKE) coverage-go
	$(MAKE) coverage-quant

coverage-go:
	@go test ./... -coverprofile=/tmp/dwizzybrain-go.cover.out
	@go tool cover -func=/tmp/dwizzybrain-go.cover.out | awk -v min="$(GO_COVER_MIN)" '/^total:/{ \
		gsub("%","",$$3); cov=$$3+0; \
		printf("Go coverage: %.1f%% (min %.1f%%)\n", cov, min); \
		if (cov < min) { exit 1 } \
	}'
	@$(MAKE) coverage-go-critical

coverage-go-critical:
	@set -e; \
	check_pkg() { \
		pkg="$$1"; min="$$2"; \
		cov="$$(go test $$pkg -cover -count=1 | sed -n 's/.*coverage: \([0-9.]*\)%.*/\1/p' | tail -n1)"; \
		if [ -z "$$cov" ]; then \
			echo "failed to parse coverage for $$pkg"; \
			exit 1; \
		fi; \
		awk -v pkg="$$pkg" -v cov="$$cov" -v min="$$min" 'BEGIN { \
			printf("%s coverage: %.1f%% (min %.1f%%)\n", pkg, cov, min); \
			if ((cov + 0) < (min + 0)) exit 1; \
		}'; \
	}; \
	check_pkg ./api/cmd/api "$(API_CMD_COVER_MIN)"; \
	check_pkg ./api/auth "$(API_AUTH_COVER_MIN)"; \
	check_pkg ./api/defi "$(API_DEFI_COVER_MIN)"; \
	check_pkg ./api/handler "$(API_HANDLER_COVER_MIN)"; \
	check_pkg ./api/market "$(API_MARKET_COVER_MIN)"; \
	check_pkg ./api/news "$(API_NEWS_COVER_MIN)";

coverage-quant:
	@if [ -x quant/.venv/bin/pytest ]; then \
		quant/.venv/bin/pytest -q --cov=quant/src/quant --cov-report=term-missing --cov-fail-under=$(QUANT_COVER_MIN) quant/tests; \
	else \
		cd quant && pytest -q --cov=src/quant --cov-report=term-missing --cov-fail-under=$(QUANT_COVER_MIN) tests; \
	fi

build:
	go build ./...

quality: fmt-check lint test coverage build
