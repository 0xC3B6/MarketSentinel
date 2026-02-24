package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"MarketSentinel/internal/collector"
	"MarketSentinel/internal/config"
	"MarketSentinel/internal/fund"
	"MarketSentinel/internal/notifier"
	"MarketSentinel/internal/recorder"
	"MarketSentinel/internal/scheduler"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[INFO] MarketSentinel starting...")

	// Load config
	cfgPath := "configs/config.yaml"
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		cfgPath = v
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("[FATAL] load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("[FATAL] config validation: %v", err)
	}

	// Init fetcher
	var fetcher collector.Fetcher
	if cfg.DataSource.BaseURL != "" {
		fetcher = collector.NewVsTraderFetcher(cfg.DataSource.BaseURL, cfg.DataSource.APIKey, cfg.Proxy)
	} else {
		fetcher = collector.NewYahooFetcher(cfg.Proxy)
	}
	log.Printf("[INFO] data source: %s", fetcher.Name())

	// Init collector
	col := collector.NewCollector(fetcher, cfg.DataSource.Symbol)

	// Init fund manager
	fm, err := fund.NewManager(cfg.Fund.StateFile, cfg.Fund.MonthlyBudget)
	if err != nil {
		log.Fatalf("[FATAL] init fund manager: %v", err)
	}

	// Init Telegram notifier
	tn := notifier.NewTelegramNotifier(cfg.Telegram.BotToken, cfg.Telegram.ChatID, cfg.Proxy)

	// Init recorder
	var rec recorder.Recorder
	if cfg.Database.SQLitePath != "" {
		sr, err := recorder.NewSQLiteRecorder(cfg.Database.SQLitePath)
		if err != nil {
			log.Printf("[WARN] init sqlite recorder failed, using noop: %v", err)
			rec = recorder.NewNoopRecorder()
		} else {
			rec = sr
			defer sr.Close()
		}
	} else {
		rec = recorder.NewNoopRecorder()
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init scheduler
	sched := scheduler.NewScheduler(ctx, col, fm, tn, rec)
	if err := sched.RegisterAll(cfg.Schedule.WeeklyCron, cfg.Schedule.DailyCron, cfg.Schedule.MonthlyCron); err != nil {
		log.Fatalf("[FATAL] register cron tasks: %v", err)
	}
	sched.Start()
	defer sched.Stop()

	// Start Telegram polling
	go tn.StartPolling(ctx, sched.HandleCommand)
	log.Println("[INFO] Telegram polling started")

	// Optional: run immediately on start
	if os.Getenv("RUN_ON_START") == "true" {
		log.Println("[INFO] RUN_ON_START enabled, executing weekly task now")
		go sched.RunWeeklyNow()
	}

	log.Println("[INFO] MarketSentinel is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("[INFO] shutdown signal received, stopping...")
	cancel()
	log.Println("[INFO] MarketSentinel stopped")
}
