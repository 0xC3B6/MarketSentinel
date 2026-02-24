package strategy

import (
	"fmt"
	"math"

	"MarketSentinel/internal/model"
)

// scoreMA200Deviation scores based on how far the current price deviates from MA200.
// Weight: 0.35
func scoreMA200Deviation(ind *model.MarketIndicators) model.FactorScore {
	if ind.MA200 == 0 {
		return model.FactorScore{Name: "MA200偏离度", RawScore: 0, Weight: 0.35, Weighted: 0, Commentary: "MA200不可用"}
	}
	deviation := (ind.CurrentPrice - ind.MA200) / ind.MA200 * 100 // percentage

	var score float64
	switch {
	case deviation <= -20:
		score = 2.0
	case deviation <= -10:
		score = 1.5
	case deviation <= -5:
		score = 1.0
	case deviation <= 0:
		score = 0.5
	case deviation <= 5:
		score = 0
	case deviation <= 10:
		score = -0.5
	case deviation <= 15:
		score = -1.0
	case deviation <= 20:
		score = -1.5
	default:
		score = -2.0
	}

	return model.FactorScore{
		Name:       "MA200偏离度",
		RawScore:   score,
		Weight:     0.35,
		Weighted:   score * 0.35,
		Commentary: fmt.Sprintf("偏离 %+.1f%%", deviation),
	}
}

// scoreWeeklyRSI scores based on the weekly RSI(14).
// Weight: 0.25
func scoreWeeklyRSI(ind *model.MarketIndicators) model.FactorScore {
	rsi := ind.WeeklyRSI
	var score float64
	switch {
	case rsi <= 25:
		score = 2.0
	case rsi <= 30:
		score = 1.5
	case rsi <= 40:
		score = 1.0
	case rsi <= 45:
		score = 0.5
	case rsi <= 55:
		score = 0
	case rsi <= 60:
		score = -0.5
	case rsi <= 70:
		score = -1.0
	case rsi <= 80:
		score = -1.5
	default:
		score = -2.0
	}

	return model.FactorScore{
		Name:       "周线RSI",
		RawScore:   score,
		Weight:     0.25,
		Weighted:   score * 0.25,
		Commentary: fmt.Sprintf("RSI=%.0f", rsi),
	}
}

// scoreDailyRSI scores based on the daily RSI(14).
// Weight: 0.15
func scoreDailyRSI(ind *model.MarketIndicators) model.FactorScore {
	rsi := ind.DailyRSI
	var score float64
	switch {
	case rsi <= 25:
		score = 2.0
	case rsi <= 30:
		score = 1.5
	case rsi <= 40:
		score = 1.0
	case rsi <= 45:
		score = 0.5
	case rsi <= 55:
		score = 0
	case rsi <= 60:
		score = -0.5
	case rsi <= 70:
		score = -1.0
	case rsi <= 80:
		score = -1.5
	default:
		score = -2.0
	}

	return model.FactorScore{
		Name:       "日线RSI",
		RawScore:   score,
		Weight:     0.15,
		Weighted:   score * 0.15,
		Commentary: fmt.Sprintf("RSI=%.0f", rsi),
	}
}

// score52WeekPosition scores based on where the price sits in the 52-week range.
// Weight: 0.10
// Special logic: when position > 95%, requires otherFactorsAvg < -1 to give -2, otherwise caps at -1.
func score52WeekPosition(ind *model.MarketIndicators, otherFactorsAvg float64) model.FactorScore {
	pos := ind.Position52w * 100 // convert to percentage

	var score float64
	switch {
	case pos <= 10:
		score = 2.0
	case pos <= 20:
		score = 1.5
	case pos <= 30:
		score = 1.0
	case pos <= 40:
		score = 0.5
	case pos <= 60:
		score = 0
	case pos <= 70:
		score = -0.5
	case pos <= 80:
		score = -1.0
	case pos <= 95:
		score = -1.5
	default:
		// > 95%: need other factors avg < -1 to give -2, otherwise cap at -1
		if otherFactorsAvg < -1 {
			score = -2.0
		} else {
			score = -1.0
		}
	}

	return model.FactorScore{
		Name:       "52周位置",
		RawScore:   score,
		Weight:     0.10,
		Weighted:   score * 0.10,
		Commentary: fmt.Sprintf("位置=%.0f%%", pos),
	}
}

// scoreTrendTracker scores based on MA alignment and 30-day extremes.
// Weight: 0.15
// Bull alignment: price > MA20w > MA50w
// Bear alignment: price < MA20w < MA50w
func scoreTrendTracker(ind *model.MarketIndicators) model.FactorScore {
	bullish := ind.CurrentPrice > ind.MA20w && ind.MA20w > ind.MA50w
	bearish := ind.CurrentPrice < ind.MA20w && ind.MA20w < ind.MA50w

	near30dHigh := math.Abs(ind.CurrentPrice-ind.High30d)/ind.High30d < 0.01
	near30dLow := math.Abs(ind.CurrentPrice-ind.Low30d)/ind.Low30d < 0.01

	var score float64
	var commentary string

	switch {
	case bullish && near30dHigh:
		score = 1.5
		commentary = "多头排列+30日新高"
	case bullish:
		score = 1.0
		commentary = "多头排列"
	case bearish && near30dLow:
		score = -1.0
		commentary = "空头排列+30日新低"
	case bearish:
		score = -0.5
		commentary = "空头排列"
	default:
		score = 0
		commentary = "震荡"
	}

	return model.FactorScore{
		Name:       "趋势追踪",
		RawScore:   score,
		Weight:     0.15,
		Weighted:   score * 0.15,
		Commentary: commentary,
	}
}
