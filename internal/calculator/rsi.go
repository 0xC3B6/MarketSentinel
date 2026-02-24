package calculator

import (
	"errors"

	"MarketSentinel/internal/model"
)

// CalculateRSI computes the Wilder-smoothed RSI over the given period.
// Requires at least period+1 bars. Returns 50.0 if data is insufficient.
func CalculateRSI(bars []model.OHLCV, period int) (float64, error) {
	if period <= 0 {
		return 0, errors.New("period must be positive")
	}
	if len(bars) < period+1 {
		return 50.0, nil // default when data insufficient
	}

	closes := extractCloses(bars)

	// Initial average gain/loss over the first `period` changes
	var avgGain, avgLoss float64
	for i := 1; i <= period; i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			avgGain += change
		} else {
			avgLoss -= change // make positive
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Wilder smoothing for remaining bars
	for i := period + 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		gain, loss := 0.0, 0.0
		if change > 0 {
			gain = change
		} else {
			loss = -change
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	if avgLoss == 0 {
		return 100.0, nil
	}
	rs := avgGain / avgLoss
	rsi := 100.0 - 100.0/(1.0+rs)
	return rsi, nil
}
