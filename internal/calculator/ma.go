package calculator

import (
	"errors"

	"MarketSentinel/internal/model"
)

// CalculateSMA computes the simple moving average of the given prices over the specified period.
func CalculateSMA(prices []float64, period int) (float64, error) {
	if period <= 0 {
		return 0, errors.New("period must be positive")
	}
	if len(prices) < period {
		return 0, errors.New("not enough data for SMA calculation")
	}
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period), nil
}

// CalculateMA200 returns the 200-day simple moving average from daily bars.
func CalculateMA200(dailyBars []model.OHLCV) (float64, error) {
	closes := extractCloses(dailyBars)
	return CalculateSMA(closes, 200)
}

// CalculateMA20w returns the 20-week simple moving average from weekly bars.
func CalculateMA20w(weeklyBars []model.OHLCV) (float64, error) {
	closes := extractCloses(weeklyBars)
	return CalculateSMA(closes, 20)
}

// CalculateMA50w returns the 50-week simple moving average from weekly bars.
func CalculateMA50w(weeklyBars []model.OHLCV) (float64, error) {
	closes := extractCloses(weeklyBars)
	return CalculateSMA(closes, 50)
}

func extractCloses(bars []model.OHLCV) []float64 {
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	return closes
}
