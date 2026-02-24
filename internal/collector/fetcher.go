package collector

import "MarketSentinel/internal/model"

// Fetcher defines the interface for fetching market data.
type Fetcher interface {
	FetchDailyBars(symbol string, days int) ([]model.OHLCV, error)
	FetchWeeklyBars(symbol string, weeks int) ([]model.OHLCV, error)
	FetchCurrentPrice(symbol string) (float64, error)
	Name() string
}
