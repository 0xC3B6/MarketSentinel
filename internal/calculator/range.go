package calculator

import (
	"errors"
	"math"

	"MarketSentinel/internal/model"
)

// Calculate52WeekRange scans the most recent 252 trading days and returns the high and low.
func Calculate52WeekRange(dailyBars []model.OHLCV) (high, low float64, err error) {
	if len(dailyBars) == 0 {
		return 0, 0, errors.New("no daily bars provided")
	}
	n := len(dailyBars)
	start := n - 252
	if start < 0 {
		start = 0
	}
	high = math.Inf(-1)
	low = math.Inf(1)
	for i := start; i < n; i++ {
		if dailyBars[i].High > high {
			high = dailyBars[i].High
		}
		if dailyBars[i].Low < low {
			low = dailyBars[i].Low
		}
	}
	return high, low, nil
}

// Calculate30DayRange scans the most recent 22 trading days and returns the high and low.
func Calculate30DayRange(dailyBars []model.OHLCV) (high, low float64, err error) {
	if len(dailyBars) == 0 {
		return 0, 0, errors.New("no daily bars provided")
	}
	n := len(dailyBars)
	start := n - 22
	if start < 0 {
		start = 0
	}
	high = math.Inf(-1)
	low = math.Inf(1)
	for i := start; i < n; i++ {
		if dailyBars[i].High > high {
			high = dailyBars[i].High
		}
		if dailyBars[i].Low < low {
			low = dailyBars[i].Low
		}
	}
	return high, low, nil
}

// Calculate52WeekPosition returns where the current price sits within the 52-week range (0.0~1.0).
func Calculate52WeekPosition(current, high, low float64) (float64, error) {
	if high == low {
		return 0.5, nil
	}
	if high < low {
		return 0, errors.New("high must be >= low")
	}
	pos := (current - low) / (high - low)
	if pos < 0 {
		pos = 0
	}
	if pos > 1 {
		pos = 1
	}
	return pos, nil
}
