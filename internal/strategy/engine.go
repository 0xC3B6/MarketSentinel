package strategy

import "MarketSentinel/internal/model"

// Tiers defines the 7-level investment mapping.
var Tiers = []struct {
	MinScore   float64
	Tier       model.InvestmentTier
}{
	{1.5, model.InvestmentTier{Label: "极限重仓", Multiplier: 1.0, UseReserve: 1.5}},
	{1.2, model.InvestmentTier{Label: "重仓买入", Multiplier: 1.0, UseReserve: 1.0}},
	{0.8, model.InvestmentTier{Label: "加仓买入", Multiplier: 1.0, UseReserve: 0.5}},
	{0.0, model.InvestmentTier{Label: "正常定投", Multiplier: 1.0, UseReserve: 0}},
	{-0.8, model.InvestmentTier{Label: "缩减定投", Multiplier: 0.5, UseReserve: 0}},
	{-1.5, model.InvestmentTier{Label: "轻仓观望", Multiplier: 0.25, UseReserve: 0}},
}

// DefaultTier is the lowest tier for scores < -1.5.
var DefaultTier = model.InvestmentTier{Label: "最低参与", Multiplier: 0.15, UseReserve: 0}

// mapTier maps a total score to an InvestmentTier.
func mapTier(totalScore float64) model.InvestmentTier {
	for _, t := range Tiers {
		if totalScore >= t.MinScore {
			return t.Tier
		}
	}
	return DefaultTier
}

// Evaluate computes the full trade signal from market indicators.
func Evaluate(ind *model.MarketIndicators) *model.TradeSignal {
	// Step a: compute factors 1, 2, 3, 5
	f1 := scoreMA200Deviation(ind)
	f2 := scoreWeeklyRSI(ind)
	f3 := scoreDailyRSI(ind)
	f5 := scoreTrendTracker(ind)

	// Step b: compute otherFactorsAvg for factor 4
	otherFactorsAvg := (f1.RawScore + f2.RawScore + f3.RawScore + f5.RawScore) / 4.0

	// Step c: compute factor 4 with the avg
	f4 := score52WeekPosition(ind, otherFactorsAvg)

	factors := []model.FactorScore{f1, f2, f3, f4, f5}

	// Step d: weighted sum
	totalScore := f1.Weighted + f2.Weighted + f3.Weighted + f4.Weighted + f5.Weighted

	// Step e: map to tier
	tier := mapTier(totalScore)

	signal := &model.TradeSignal{
		Factors:     factors,
		TotalScore:  totalScore,
		Tier:        tier,
		TriggerType: model.TriggerWeekly,
	}

	// Step f: take-profit warning
	if ind.WeeklyRSI > 85 || ind.DailyRSI > 85 {
		signal.WarningMsg = "⚠️ RSI > 85 止盈预警：建议考虑部分止盈"
	}

	return signal
}
