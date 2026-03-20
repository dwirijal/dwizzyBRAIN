package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"dwizzyBRAIN/engine/defi"
	defistable "dwizzyBRAIN/engine/defi/stablecoins"
	defiyields "dwizzyBRAIN/engine/defi/yields"
	"dwizzyBRAIN/engine/market"
	"dwizzyBRAIN/engine/market/arbitrage"
	"dwizzyBRAIN/engine/market/coingecko"
	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/engine/market/ohlcv"
	"dwizzyBRAIN/engine/market/ticker"
	"dwizzyBRAIN/engine/market/ws"
	newsengine "dwizzyBRAIN/engine/news"
	newsai "dwizzyBRAIN/engine/news/ai"
	newsarchive "dwizzyBRAIN/engine/news/archive"
	newsimpact "dwizzyBRAIN/engine/news/impact"
	"dwizzyBRAIN/engine/storage"
	storageext "dwizzyBRAIN/engine/storage_ext"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

const defaultMappingSyncExchanges = "binance,bybit,okx,gateio,kucoin,kraken,mexc,htx"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	postgresPool, err := storage.NewPostgresPoolFromEnv(ctx)
	if err != nil {
		return err
	}
	defer postgresPool.Close()

	mappingEnabled := parseBoolEnv(os.Getenv("MAPPING_SYNC_ENABLED"), true)
	defiEnabled := parseBoolEnv(os.Getenv("DEFI_TVL_ENABLED"), false)
	defiRunOnce := parseBoolEnv(os.Getenv("DEFI_TVL_RUN_ONCE"), false)
	yieldsEnabled := parseBoolEnv(os.Getenv("DEFI_YIELD_ENABLED"), false)
	yieldsRunOnce := parseBoolEnv(os.Getenv("DEFI_YIELD_RUN_ONCE"), false)
	stableEnabled := parseBoolEnv(os.Getenv("DEFI_STABLECOIN_ENABLED"), false)
	stableRunOnce := parseBoolEnv(os.Getenv("DEFI_STABLECOIN_RUN_ONCE"), false)
	newsEnabled := parseBoolEnv(os.Getenv("NEWS_ENABLED"), false)
	newsRunOnce := parseBoolEnv(os.Getenv("NEWS_RUN_ONCE"), false)
	newsAIEnabled := parseBoolEnv(os.Getenv("NEWS_AI_ENABLED"), false)
	newsAIRunOnce := parseBoolEnv(os.Getenv("NEWS_AI_RUN_ONCE"), false)
	newsImpactEnabled := parseBoolEnv(os.Getenv("NEWS_IMPACT_ENABLED"), false)
	newsImpactRunOnce := parseBoolEnv(os.Getenv("NEWS_IMPACT_RUN_ONCE"), false)
	newsArchiveEnabled := parseBoolEnv(os.Getenv("NEWS_ARCHIVE_ENABLED"), false)
	newsArchiveRunOnce := parseBoolEnv(os.Getenv("NEWS_ARCHIVE_RUN_ONCE"), false)
	storageExtEnabled := parseBoolEnv(os.Getenv("STORAGE_EXT_ENABLED"), false)
	storageExtRunOnce := parseBoolEnv(os.Getenv("STORAGE_EXT_RUN_ONCE"), false)
	ohlcvTargets, err := parseOHLCVTargets(os.Getenv("OHLCV_SYNC_TARGETS"))
	if err != nil {
		return err
	}
	ohlcvEnabled := len(ohlcvTargets) > 0
	binanceWSEnabled := parseBoolEnv(os.Getenv("BINANCE_WS_ENABLED"), false)
	pollTargets, err := parseTickerPollTargets(os.Getenv("TICKER_POLL_TARGETS"))
	if err != nil {
		return err
	}
	tickerRuntimeEnabled := binanceWSEnabled || len(pollTargets) > 0
	coingeckoEnabled := parseBoolEnv(os.Getenv("COINGECKO_COLDLOAD_ENABLED"), false)
	coingeckoRunOnce := parseBoolEnv(os.Getenv("COINGECKO_COLDLOAD_RUN_ONCE"), false)
	spreadEnabled := parseBoolEnv(os.Getenv("SPREAD_ENABLED"), false)
	arbitrageEnabled := parseBoolEnv(os.Getenv("ARBITRAGE_ENABLED"), false)

	exchanges := parseExchanges(os.Getenv("MAPPING_SYNC_EXCHANGES"))
	if len(exchanges) == 0 {
		exchanges = parseExchanges(defaultMappingSyncExchanges)
	}

	interval, err := parseInterval(os.Getenv("MAPPING_SYNC_INTERVAL"))
	if err != nil {
		return err
	}
	coingeckoPages, err := parsePositiveIntEnv("COINGECKO_COLDLOAD_PAGES", os.Getenv("COINGECKO_COLDLOAD_PAGES"), 4)
	if err != nil {
		return err
	}
	coingeckoPerPage, err := parsePositiveIntEnv("COINGECKO_COLDLOAD_PER_PAGE", os.Getenv("COINGECKO_COLDLOAD_PER_PAGE"), 250)
	if err != nil {
		return err
	}
	defiProtocolLimit, err := parsePositiveIntEnv("DEFI_TVL_PROTOCOL_LIMIT", os.Getenv("DEFI_TVL_PROTOCOL_LIMIT"), 50)
	if err != nil {
		return err
	}
	defiProtocolHistoryLimit, err := parsePositiveIntEnv("DEFI_TVL_PROTOCOL_HISTORY_LIMIT", os.Getenv("DEFI_TVL_PROTOCOL_HISTORY_LIMIT"), 50)
	if err != nil {
		return err
	}
	defiChainHistoryLimit, err := parsePositiveIntEnv("DEFI_TVL_CHAIN_HISTORY_LIMIT", os.Getenv("DEFI_TVL_CHAIN_HISTORY_LIMIT"), 15)
	if err != nil {
		return err
	}
	defiHistoryPoints, err := parsePositiveIntEnv("DEFI_TVL_HISTORY_POINTS", os.Getenv("DEFI_TVL_HISTORY_POINTS"), 365)
	if err != nil {
		return err
	}
	yieldPoolLimit, err := parsePositiveIntEnv("DEFI_YIELD_POOL_LIMIT", os.Getenv("DEFI_YIELD_POOL_LIMIT"), 100)
	if err != nil {
		return err
	}
	yieldHistoryLimit, err := parsePositiveIntEnv("DEFI_YIELD_HISTORY_LIMIT", os.Getenv("DEFI_YIELD_HISTORY_LIMIT"), 30)
	if err != nil {
		return err
	}
	yieldHistoryPoints, err := parsePositiveIntEnv("DEFI_YIELD_HISTORY_POINTS", os.Getenv("DEFI_YIELD_HISTORY_POINTS"), 365)
	if err != nil {
		return err
	}
	stableAssetLimit, err := parsePositiveIntEnv("DEFI_STABLECOIN_POOL_LIMIT", os.Getenv("DEFI_STABLECOIN_POOL_LIMIT"), 100)
	if err != nil {
		return err
	}
	stableHistoryLimit, err := parsePositiveIntEnv("DEFI_STABLECOIN_HISTORY_LIMIT", os.Getenv("DEFI_STABLECOIN_HISTORY_LIMIT"), 30)
	if err != nil {
		return err
	}
	stableHistoryPoints, err := parsePositiveIntEnv("DEFI_STABLECOIN_HISTORY_POINTS", os.Getenv("DEFI_STABLECOIN_HISTORY_POINTS"), 365)
	if err != nil {
		return err
	}
	stableDepegPct, err := parsePositiveFloatEnv("DEFI_STABLECOIN_DEPEG_THRESHOLD_PCT", os.Getenv("DEFI_STABLECOIN_DEPEG_THRESHOLD_PCT"), 0.01)
	if err != nil {
		return err
	}
	newsSources := parseExchanges(os.Getenv("NEWS_SOURCES"))

	manager, err := market.NewCCXTManager(exchanges)
	if err != nil {
		return err
	}

	var (
		valkeyClient    *redis.Client
		sharedAgg       *ticker.Aggregator
		sharedMapStore  *mapping.PostgresStore
		sharedResolver  *mapping.SymbolResolver
		sharedPublisher *market.Publisher
		sharedIngestion *market.IngestionService
	)

	if tickerRuntimeEnabled || arbitrageEnabled || spreadEnabled || ohlcvEnabled {
		valkeyClient, err = storage.NewValkeyClientFromEnv(ctx)
		if err != nil {
			return err
		}
		defer valkeyClient.Close()
	}
	if tickerRuntimeEnabled || arbitrageEnabled || spreadEnabled || ohlcvEnabled {
		sharedMapStore = mapping.NewPostgresStore(postgresPool)
		sharedResolver = mapping.NewSymbolResolver(sharedMapStore, nil)
		sharedPublisher = market.NewPublisher(valkeyClient)
		sharedIngestion = market.NewIngestionService(sharedResolver, sharedPublisher)
		sharedAgg = ticker.NewAggregator()
	}

	var coingeckoJob *coingecko.Job
	if coingeckoEnabled || coingeckoRunOnce {
		cgFetcher, err := coingecko.NewFetcherFromEnv()
		if err != nil {
			return err
		}
		cgStore := coingecko.NewStore(postgresPool)
		cgService := coingecko.NewService(cgFetcher, cgStore, coingeckoPages, coingeckoPerPage)
		cgInterval, err := parseIntervalWithDefault("COINGECKO_COLDLOAD_INTERVAL", os.Getenv("COINGECKO_COLDLOAD_INTERVAL"), 24*time.Hour)
		if err != nil {
			return err
		}
		coingeckoJob = coingecko.NewJob(cgService, cgInterval, log.Default())
	}

	var defiJob *defi.Job
	if defiEnabled || defiRunOnce {
		defiClient, err := defi.NewClient(os.Getenv("DEFI_LLAMA_BASE_URL"))
		if err != nil {
			return err
		}
		defiStore := defi.NewStore(postgresPool)
		defiService := defi.NewService(defiClient, defiStore, defiProtocolLimit, defiProtocolHistoryLimit, defiChainHistoryLimit, defiHistoryPoints)
		defiInterval, err := parseIntervalWithDefault("DEFI_TVL_INTERVAL", os.Getenv("DEFI_TVL_INTERVAL"), 6*time.Hour)
		if err != nil {
			return err
		}
		defiJob = defi.NewJob(defiService, defiInterval, log.Default())
	}

	var yieldJob *defiyields.Job
	if yieldsEnabled || yieldsRunOnce {
		yieldClient, err := defiyields.NewClient(os.Getenv("DEFI_YIELD_BASE_URL"))
		if err != nil {
			return err
		}
		yieldStore := defiyields.NewStore(postgresPool)
		yieldService := defiyields.NewService(yieldClient, yieldStore, yieldPoolLimit, yieldHistoryLimit, yieldHistoryPoints)
		yieldInterval, err := parseIntervalWithDefault("DEFI_YIELD_INTERVAL", os.Getenv("DEFI_YIELD_INTERVAL"), time.Hour)
		if err != nil {
			return err
		}
		yieldJob = defiyields.NewJob(yieldService, yieldInterval, log.Default())
	}

	var stableJob *defistable.Job
	if stableEnabled || stableRunOnce {
		stableClient, err := defistable.NewClient(os.Getenv("DEFI_STABLECOIN_BASE_URL"))
		if err != nil {
			return err
		}
		stableStore := defistable.NewStore(postgresPool)
		stableService := defistable.NewService(stableClient, stableStore, stableAssetLimit, stableHistoryLimit, stableHistoryPoints, stableDepegPct)
		stableInterval, err := parseIntervalWithDefault("DEFI_STABLECOIN_INTERVAL", os.Getenv("DEFI_STABLECOIN_INTERVAL"), 12*time.Hour)
		if err != nil {
			return err
		}
		stableJob = defistable.NewJob(stableService, stableInterval, log.Default())
	}

	var newsJob *newsengine.Job
	if newsEnabled || newsRunOnce {
		newsFetcher := newsengine.NewRSSFetcher()
		newsStore := newsengine.NewStore(postgresPool)
		newsService := newsengine.NewService(newsFetcher, newsStore, newsSources)
		newsInterval, err := parseIntervalWithDefault("NEWS_INTERVAL", os.Getenv("NEWS_INTERVAL"), 15*time.Minute)
		if err != nil {
			return err
		}
		newsJob = newsengine.NewJob(newsService, newsInterval, log.Default())
	}

	var newsAIJob *newsai.Job
	if newsAIEnabled || newsAIRunOnce {
		newsAIStore := newsai.NewStore(postgresPool)
		newsAIBatchLimit, err := parsePositiveIntEnv("NEWS_AI_BATCH_LIMIT", os.Getenv("NEWS_AI_BATCH_LIMIT"), 25)
		if err != nil {
			return err
		}
		newsAIService := newsai.NewService(newsAIStore, newsAIBatchLimit)
		newsAIInterval, err := parseIntervalWithDefault("NEWS_AI_INTERVAL", os.Getenv("NEWS_AI_INTERVAL"), 5*time.Minute)
		if err != nil {
			return err
		}
		newsAIJob = newsai.NewJob(newsAIService, newsAIInterval, log.Default())
	}

	newsImpactLimit, err := parsePositiveIntEnv("NEWS_IMPACT_BATCH_LIMIT", os.Getenv("NEWS_IMPACT_BATCH_LIMIT"), 50)
	if err != nil {
		return err
	}
	newsImpactInterval, err := parseIntervalWithDefault("NEWS_IMPACT_INTERVAL", os.Getenv("NEWS_IMPACT_INTERVAL"), 30*time.Minute)
	if err != nil {
		return err
	}
	newsImpactTimeframe := strings.TrimSpace(os.Getenv("NEWS_IMPACT_TIMEFRAME"))
	if newsImpactTimeframe == "" {
		newsImpactTimeframe = "1m"
	}
	newsArchiveInterval, err := parseIntervalWithDefault("NEWS_ARCHIVE_INTERVAL", os.Getenv("NEWS_ARCHIVE_INTERVAL"), 12*time.Hour)
	if err != nil {
		return err
	}
	newsArchiveLimit, err := parsePositiveIntEnv("NEWS_ARCHIVE_LIMIT", os.Getenv("NEWS_ARCHIVE_LIMIT"), 25)
	if err != nil {
		return err
	}
	newsArchiveRemoteBase := strings.TrimSpace(os.Getenv("NEWS_ARCHIVE_GDRIVE_PREFIX"))
	if newsArchiveRemoteBase == "" {
		newsArchiveRemoteBase = "Projects/DwizzyOS/dwizzyBrain/news"
	}
	storageExtInterval, err := parseIntervalWithDefault("STORAGE_EXT_INTERVAL", os.Getenv("STORAGE_EXT_INTERVAL"), 24*time.Hour)
	if err != nil {
		return err
	}
	storageGDriveSourcePath := strings.TrimSpace(os.Getenv("STORAGE_GDRIVE_SOURCE_PATH"))
	storageGDriveBackupName := strings.TrimSpace(os.Getenv("STORAGE_GDRIVE_BACKUP_NAME"))
	storageR2SourcePath := strings.TrimSpace(os.Getenv("STORAGE_R2_SOURCE_PATH"))

	var newsImpactJob *newsimpact.Job
	if newsImpactEnabled || newsImpactRunOnce {
		impactTimescalePool, err := storage.NewTimescalePoolFromEnv(ctx)
		if err != nil {
			return err
		}
		defer impactTimescalePool.Close()

		impactStore := newsimpact.NewStore(postgresPool)
		impactMappings := mapping.NewPostgresStore(postgresPool)
		impactCandles := ohlcv.NewTimescaleStore(impactTimescalePool)
		impactPricer := newsimpact.NewPriceResolver(impactMappings, impactCandles, newsImpactTimeframe)
		impactService := newsimpact.NewService(impactStore, impactPricer, newsImpactLimit)
		newsImpactJob = newsimpact.NewJob(impactService, newsImpactInterval, log.Default())
	}

	var newsArchiveJob *newsarchive.Job
	if newsArchiveEnabled || newsArchiveRunOnce {
		archiveRemote := strings.TrimSpace(os.Getenv("NEWS_ARCHIVE_GDRIVE_REMOTE"))
		if archiveRemote == "" {
			archiveRemote = "gdrive"
		}
		archiveBackup := storageext.NewGDriveBackup(storageext.NewRcloneRunner(), archiveRemote, "")
		archiveStore := newsarchive.NewStore(postgresPool)
		archiveService := newsarchive.NewService(archiveStore, archiveBackup, newsArchiveRemoteBase, newsArchiveLimit)
		newsArchiveJob = newsarchive.NewJob(archiveService, newsArchiveInterval, log.Default())
	}

	var storageExtJob *storageext.Job
	if storageExtEnabled || storageExtRunOnce {
		gdriveRemote := strings.TrimSpace(os.Getenv("RCLONE_GDRIVE_REMOTE"))
		r2Remote := strings.TrimSpace(os.Getenv("RCLONE_R2_REMOTE"))
		gdrivePrefix := strings.TrimSpace(os.Getenv("RCLONE_GDRIVE_PREFIX"))
		r2Prefix := strings.TrimSpace(os.Getenv("RCLONE_R2_PREFIX"))

		var gdrive *storageext.GDriveBackup
		if gdriveRemote != "" {
			gdrive = storageext.NewGDriveBackup(storageext.NewRcloneRunner(), gdriveRemote, gdrivePrefix)
		}

		var r2 *storageext.R2Sync
		if r2Remote != "" {
			r2 = storageext.NewR2Sync(storageext.NewRcloneRunner(), r2Remote, r2Prefix)
		}

		if gdrive == nil && r2 == nil {
			return fmt.Errorf("storage ext enabled but no rclone remotes configured")
		}
		if gdrive != nil && storageGDriveSourcePath == "" && r2 != nil && storageR2SourcePath == "" {
			return fmt.Errorf("storage ext enabled but no source paths configured")
		}

		storageExtJob = storageext.NewJob(gdrive, r2, storageGDriveSourcePath, storageGDriveBackupName, storageR2SourcePath, storageExtInterval, log.Default())
	}

	var mappingJob *market.MappingSyncJob
	if mappingEnabled {
		store := mapping.NewPostgresStore(postgresPool)
		builder := mapping.NewMappingBuilder(store)
		validator := mapping.NewMappingValidator(store)
		service := market.NewMappingSyncService(manager, builder, validator)
		mappingJob = market.NewMappingSyncJob(loggedSyncer{service: service}, exchanges, interval, log.Default())
	}

	var ohlcvJob *ohlcv.Scheduler
	if ohlcvEnabled {
		timescalePool, err := storage.NewTimescalePoolFromEnv(ctx)
		if err != nil {
			return err
		}
		defer timescalePool.Close()

		ohlcvStore := ohlcv.NewTimescaleStore(timescalePool)
		service := ohlcv.NewService(sharedResolver, manager, ohlcvStore, sharedPublisher)
		ohlcvInterval, err := parseIntervalWithDefault("OHLCV_SYNC_INTERVAL", os.Getenv("OHLCV_SYNC_INTERVAL"), time.Minute)
		if err != nil {
			return err
		}
		ohlcvJob = ohlcv.NewScheduler(service, ohlcvTargets, ohlcvInterval)
	}

	var binanceJob *ticker.BinanceStreamJob
	if binanceWSEnabled {
		binanceJob = ticker.NewBinanceStreamJob(ws.NewBinanceWSClient(), sharedIngestion, sharedAgg, log.Default())
	}

	var ccxtPollJob *ticker.CCXTPollJob
	if len(pollTargets) > 0 {
		pollInterval, err := parseIntervalWithDefault("TICKER_POLL_INTERVAL", os.Getenv("TICKER_POLL_INTERVAL"), 10*time.Second)
		if err != nil {
			return err
		}
		ccxtPollJob = ticker.NewCCXTPollJob(manager, sharedResolver, sharedIngestion, sharedAgg, pollTargets, pollInterval, log.Default())
	}

	var spreadJob *ticker.SpreadJob
	if spreadEnabled {
		timescalePool, err := storage.NewTimescalePoolFromEnv(ctx)
		if err != nil {
			return err
		}
		defer timescalePool.Close()

		spreadStore := ticker.NewSpreadStore(timescalePool)
		recorder := ticker.NewSpreadRecorder(sharedAgg, spreadStore)
		spreadInterval, err := parseIntervalWithDefault("SPREAD_INTERVAL", os.Getenv("SPREAD_INTERVAL"), 5*time.Minute)
		if err != nil {
			return err
		}
		spreadJob = ticker.NewSpreadJob(recorder, spreadInterval, log.Default())
	}

	var arbitrageJob *arbitrage.Job
	if arbitrageEnabled {
		configStore := arbitrage.NewConfigStore(postgresPool)
		signalStore := arbitrage.NewSignalStore(postgresPool)
		engine := arbitrage.NewEngine(sharedAgg, configStore, signalStore, valkeyClient)
		arbitrageInterval, err := parseIntervalWithDefault("ARBITRAGE_INTERVAL", os.Getenv("ARBITRAGE_INTERVAL"), 5*time.Second)
		if err != nil {
			return err
		}
		arbitrageJob = arbitrage.NewJob(engine, arbitrageInterval, log.Default())
	}

	mappingRunOnce := parseBoolEnv(os.Getenv("MAPPING_SYNC_RUN_ONCE"), false)
	ohlcvRunOnce := parseBoolEnv(os.Getenv("OHLCV_SYNC_RUN_ONCE"), false)
	spreadRunOnce := parseBoolEnv(os.Getenv("SPREAD_RUN_ONCE"), false)
	arbitrageRunOnce := parseBoolEnv(os.Getenv("ARBITRAGE_RUN_ONCE"), false)

	if mappingRunOnce || coingeckoRunOnce || defiRunOnce || yieldsRunOnce || stableRunOnce || newsRunOnce || newsAIRunOnce || newsImpactRunOnce || newsArchiveRunOnce || storageExtRunOnce || ohlcvRunOnce || spreadRunOnce || arbitrageRunOnce {
		if mappingRunOnce && mappingJob != nil {
			if _, err := mappingJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if coingeckoRunOnce && coingeckoJob != nil {
			if _, err := coingeckoJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if defiRunOnce && defiJob != nil {
			if _, err := defiJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if yieldsRunOnce && yieldJob != nil {
			if _, err := yieldJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if stableRunOnce && stableJob != nil {
			if _, err := stableJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if newsRunOnce && newsJob != nil {
			if _, err := newsJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if newsAIRunOnce && newsAIJob != nil {
			if _, err := newsAIJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if newsImpactRunOnce && newsImpactJob != nil {
			if _, err := newsImpactJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if newsArchiveRunOnce && newsArchiveJob != nil {
			if _, err := newsArchiveJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if storageExtRunOnce && storageExtJob != nil {
			if _, err := storageExtJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if ohlcvRunOnce && ohlcvJob != nil {
			if err := ohlcvJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if spreadRunOnce && spreadJob != nil {
			if err := spreadJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		if arbitrageRunOnce && arbitrageJob != nil {
			if _, err := arbitrageJob.RunOnce(ctx); err != nil {
				return err
			}
		}
		return nil
	}

	group, groupCtx := errgroup.WithContext(ctx)
	started := 0
	if mappingJob != nil {
		started++
		group.Go(func() error {
			err := mappingJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if coingeckoJob != nil {
		started++
		group.Go(func() error {
			err := coingeckoJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if defiJob != nil {
		started++
		group.Go(func() error {
			err := defiJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if yieldJob != nil {
		started++
		group.Go(func() error {
			err := yieldJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if stableJob != nil {
		started++
		group.Go(func() error {
			err := stableJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if newsJob != nil {
		started++
		group.Go(func() error {
			err := newsJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if newsAIJob != nil {
		started++
		group.Go(func() error {
			err := newsAIJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if newsImpactJob != nil {
		started++
		group.Go(func() error {
			err := newsImpactJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if newsArchiveJob != nil {
		started++
		group.Go(func() error {
			err := newsArchiveJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if storageExtJob != nil {
		started++
		group.Go(func() error {
			err := storageExtJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if ohlcvJob != nil {
		started++
		group.Go(func() error {
			err := ohlcvJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if binanceJob != nil {
		started++
		group.Go(func() error {
			err := binanceJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if ccxtPollJob != nil {
		started++
		group.Go(func() error {
			err := ccxtPollJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if spreadJob != nil {
		started++
		group.Go(func() error {
			err := spreadJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if arbitrageJob != nil {
		started++
		group.Go(func() error {
			err := arbitrageJob.Run(groupCtx)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		})
	}
	if started == 0 {
		return fmt.Errorf("no engine jobs enabled")
	}

	return group.Wait()
}

type loggedSyncer struct {
	service *market.MappingSyncService
}

func (s loggedSyncer) SyncAll(ctx context.Context, exchanges []string) ([]market.MappingSyncResult, error) {
	return s.service.SyncAll(ctx, exchanges)
}

func parseExchanges(raw string) []string {
	parts := strings.Split(raw, ",")
	exchanges := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		exchanges = append(exchanges, part)
	}
	return exchanges
}

func parseInterval(raw string) (time.Duration, error) {
	return parseIntervalWithDefault("MAPPING_SYNC_INTERVAL", raw, 6*time.Hour)
}

func parseIntervalWithDefault(name, raw string, fallback time.Duration) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be > 0", name)
	}
	return d, nil
}

func parsePositiveIntEnv(name, raw string, fallback int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be > 0", name)
	}

	return value, nil
}

func parsePositiveFloatEnv(name, raw string, fallback float64) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be > 0", name)
	}
	return value, nil
}

func parseBoolEnv(raw string, fallback bool) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseOHLCVTargets(raw string) ([]ohlcv.SyncRequest, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	targets := make([]ohlcv.SyncRequest, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Split(part, ":")
		if len(fields) != 3 {
			return nil, fmt.Errorf("invalid OHLCV target %q, expected coin_id:exchange:timeframe", part)
		}
		targets = append(targets, ohlcv.SyncRequest{
			CoinID:    strings.TrimSpace(fields[0]),
			Exchange:  strings.TrimSpace(fields[1]),
			Timeframe: strings.TrimSpace(fields[2]),
		})
	}

	return targets, nil
}

func parseTickerPollTargets(raw string) ([]ticker.PollTarget, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	targets := make([]ticker.PollTarget, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Split(part, ":")
		if len(fields) != 2 {
			return nil, fmt.Errorf("invalid ticker poll target %q, expected coin_id:exchange", part)
		}
		targets = append(targets, ticker.PollTarget{
			CoinID:   strings.TrimSpace(fields[0]),
			Exchange: strings.TrimSpace(fields[1]),
		})
	}

	return targets, nil
}
