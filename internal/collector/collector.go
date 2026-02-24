package collector

import (
	"fmt"
	"log"
	"time"

	"MarketSentinel/internal/calculator"
	"MarketSentinel/internal/model"
)

// MockFetcher returns controllable fixed data for development and testing.
type MockFetcher struct {
	Price      float64
	DailyData  []model.OHLCV
	WeeklyData []model.OHLCV
}

func (m *MockFetcher) Name() string { return "mock" }

func (m *MockFetcher) FetchDailyBars(_ string, days int) ([]model.OHLCV, error) {
	if m.DailyData != nil {
		return m.DailyData, nil
	}
	return generateMockBars(m.Price, days), nil
}

func (m *MockFetcher) FetchWeeklyBars(_ string, weeks int) ([]model.OHLCV, error) {
	if m.WeeklyData != nil {
		return m.WeeklyData, nil
	}
	return generateMockBars(m.Price, weeks), nil
}

func (m *MockFetcher) FetchCurrentPrice(_ string) (float64, error) {
	return m.Price, nil
}

func generateMockBars(basePrice float64, count int) []model.OHLCV {
	bars := make([]model.OHLCV, count)
	for i := 0; i < count; i++ {
		p := basePrice * (1 + float64(i-count/2)*0.001)
		bars[i] = model.OHLCV{
			Time:   time.Now().AddDate(0, 0, -(count - i)),
			Open:   p * 0.999,
			High:   p * 1.005,
			Low:    p * 0.995,
			Close:  p,
			Volume: 1000000,
		}
	}
	return bars
}

// Collector orchestrates data fetching and indicator computation.
type Collector struct {
	Fetcher Fetcher
	Symbol  string
}

// NewCollector creates a new Collector.
func NewCollector(fetcher Fetcher, symbol string) *Collector {
	return &Collector{Fetcher: fetcher, Symbol: symbol}
}

// Collect fetches market data and computes all indicators.
func (c *Collector) Collect() (*model.MarketIndicators, error) {
	dailyBars, err := c.Fetcher.FetchDailyBars(c.Symbol, 300)
	if err != nil {
		return nil, fmt.Errorf("fetch daily bars: %w", err)
	}
	weeklyBars, err := c.Fetcher.FetchWeeklyBars(c.Symbol, 60)
	if err != nil {
		return nil, fmt.Errorf("fetch weekly bars: %w", err)
	}
	currentPrice, err := c.Fetcher.FetchCurrentPrice(c.Symbol)
	if err != nil {
		return nil, fmt.Errorf("fetch current price: %w", err)
	}

	ind := &model.MarketIndicators{CurrentPrice: currentPrice}

	// MA200
	if ma, err := calculator.CalculateMA200(dailyBars); err != nil {
		log.Printf("[WARN] MA200 calculation failed: %v, using current price", err)
		ind.MA200 = currentPrice
	} else {
		ind.MA200 = ma
	}

	// MA20w
	if ma, err := calculator.CalculateMA20w(weeklyBars); err != nil {
		log.Printf("[WARN] MA20w calculation failed: %v, using current price", err)
		ind.MA20w = currentPrice
	} else {
		ind.MA20w = ma
	}

	// MA50w
	if ma, err := calculator.CalculateMA50w(weeklyBars); err != nil {
		log.Printf("[WARN] MA50w calculation failed: %v, using current price", err)
		ind.MA50w = currentPrice
	} else {
		ind.MA50w = ma
	}

	// Weekly RSI
	if rsi, err := calculator.CalculateRSI(weeklyBars, 14); err != nil {
		log.Printf("[WARN] Weekly RSI calculation failed: %v, defaulting to 50", err)
		ind.WeeklyRSI = 50
	} else {
		ind.WeeklyRSI = rsi
	}

	// Daily RSI
	if rsi, err := calculator.CalculateRSI(dailyBars, 14); err != nil {
		log.Printf("[WARN] Daily RSI calculation failed: %v, defaulting to 50", err)
		ind.DailyRSI = 50
	} else {
		ind.DailyRSI = rsi
	}

	// 52-week range
	if h, l, err := calculator.Calculate52WeekRange(dailyBars); err != nil {
		log.Printf("[WARN] 52-week range calculation failed: %v", err)
		ind.High52w = currentPrice
		ind.Low52w = currentPrice
	} else {
		ind.High52w = h
		ind.Low52w = l
	}

	// 30-day range
	if h, l, err := calculator.Calculate30DayRange(dailyBars); err != nil {
		log.Printf("[WARN] 30-day range calculation failed: %v", err)
		ind.High30d = currentPrice
		ind.Low30d = currentPrice
	} else {
		ind.High30d = h
		ind.Low30d = l
	}

	// 52-week position
	if pos, err := calculator.Calculate52WeekPosition(currentPrice, ind.High52w, ind.Low52w); err != nil {
		log.Printf("[WARN] 52-week position calculation failed: %v", err)
		ind.Position52w = 0.5
	} else {
		ind.Position52w = pos
	}

	return ind, nil
}
