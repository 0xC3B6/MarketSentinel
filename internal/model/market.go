package model

import "time"

// OHLCV represents a single candlestick bar.
type OHLCV struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// PriceSeries holds raw price data for analysis.
type PriceSeries struct {
	Symbol      string
	DailyBars   []OHLCV
	WeeklyBars  []OHLCV
	CurrentPrice float64
	FetchedAt   time.Time
}
