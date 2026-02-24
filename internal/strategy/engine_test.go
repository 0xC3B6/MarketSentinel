package strategy

import (
	"testing"

	"MarketSentinel/internal/model"
)

func TestEvaluate_NormalMarket(t *testing.T) {
	ind := &model.MarketIndicators{
		CurrentPrice: 5800,
		MA200:        5700,
		MA20w:        5750,
		MA50w:        5600,
		WeeklyRSI:    50,
		DailyRSI:     50,
		High52w:      6000,
		Low52w:       5000,
		High30d:      5850,
		Low30d:       5700,
		Position52w:  0.8,
	}
	sig := Evaluate(ind)
	if sig == nil {
		t.Fatal("expected non-nil signal")
	}
	if len(sig.Factors) != 5 {
		t.Fatalf("expected 5 factors, got %d", len(sig.Factors))
	}
	if sig.WarningMsg != "" {
		t.Errorf("unexpected warning: %s", sig.WarningMsg)
	}
}

func TestEvaluate_ExtremeOversold(t *testing.T) {
	ind := &model.MarketIndicators{
		CurrentPrice: 4500,
		MA200:        5500,
		MA20w:        5000,
		MA50w:        5200,
		WeeklyRSI:    22,
		DailyRSI:     20,
		High52w:      6000,
		Low52w:       4400,
		High30d:      4800,
		Low30d:       4500,
		Position52w:  0.06,
	}
	sig := Evaluate(ind)
	if sig.TotalScore < 1.0 {
		t.Errorf("expected high score for oversold market, got %.3f", sig.TotalScore)
	}
	if sig.Tier.UseReserve <= 0 {
		t.Error("expected reserve usage for high-score tier")
	}
}

func TestEvaluate_ExtremeOverbought(t *testing.T) {
	ind := &model.MarketIndicators{
		CurrentPrice: 6500,
		MA200:        5500,
		MA20w:        6200,
		MA50w:        6000,
		WeeklyRSI:    88,
		DailyRSI:     90,
		High52w:      6500,
		Low52w:       5000,
		High30d:      6500,
		Low30d:       6200,
		Position52w:  1.0,
	}
	sig := Evaluate(ind)
	if sig.TotalScore > -0.5 {
		t.Errorf("expected negative score for overbought market, got %.3f", sig.TotalScore)
	}
	if sig.WarningMsg == "" {
		t.Error("expected take-profit warning for RSI > 85")
	}
}

func TestMapTier_AllBoundaries(t *testing.T) {
	tests := []struct {
		score float64
		label string
	}{
		{2.0, "极限重仓"},
		{1.5, "极限重仓"},
		{1.3, "重仓买入"},
		{1.2, "重仓买入"},
		{1.0, "加仓买入"},
		{0.8, "加仓买入"},
		{0.5, "正常定投"},
		{0.0, "正常定投"},
		{-0.5, "缩减定投"},
		{-0.8, "缩减定投"},
		{-1.0, "轻仓观望"},
		{-1.5, "轻仓观望"},
		{-1.6, "最低参与"},
		{-2.0, "最低参与"},
	}
	for _, tt := range tests {
		tier := mapTier(tt.score)
		if tier.Label != tt.label {
			t.Errorf("score %.1f: expected %q, got %q", tt.score, tt.label, tier.Label)
		}
	}
}

func TestFactor4_NonlinearLogic(t *testing.T) {
	// Position > 95%, other factors avg >= -1 → should cap at -1
	ind := &model.MarketIndicators{
		CurrentPrice: 5990,
		MA200:        5800,
		MA20w:        5900,
		MA50w:        5700,
		WeeklyRSI:    55,
		DailyRSI:     55,
		High52w:      6000,
		Low52w:       5000,
		High30d:      5990,
		Low30d:       5800,
		Position52w:  0.99,
	}
	sig := Evaluate(ind)
	// Find factor 4
	var f4 model.FactorScore
	for _, f := range sig.Factors {
		if f.Name == "52周位置" {
			f4 = f
			break
		}
	}
	if f4.RawScore < -1.0 {
		t.Errorf("factor4 should cap at -1 when other factors avg >= -1, got %.1f", f4.RawScore)
	}

	// Position > 95%, other factors avg < -1 → should give -2
	ind2 := &model.MarketIndicators{
		CurrentPrice: 5990,
		MA200:        4800,
		MA20w:        5500,
		MA50w:        5700,
		WeeklyRSI:    82,
		DailyRSI:     82,
		High52w:      6000,
		Low52w:       5000,
		High30d:      5990,
		Low30d:       5800,
		Position52w:  0.99,
	}
	sig2 := Evaluate(ind2)
	var f4b model.FactorScore
	for _, f := range sig2.Factors {
		if f.Name == "52周位置" {
			f4b = f
			break
		}
	}
	if f4b.RawScore != -2.0 {
		t.Errorf("factor4 should be -2 when other factors avg < -1, got %.1f (total=%.3f)", f4b.RawScore, sig2.TotalScore)
	}
}

func TestTrendTracker_BullBear(t *testing.T) {
	// Bullish alignment
	bull := &model.MarketIndicators{
		CurrentPrice: 6000,
		MA200:        5800,
		MA20w:        5900,
		MA50w:        5700,
		WeeklyRSI:    50,
		DailyRSI:     50,
		High52w:      6100,
		Low52w:       5000,
		High30d:      6000,
		Low30d:       5800,
		Position52w:  0.9,
	}
	sig := Evaluate(bull)
	var f5 model.FactorScore
	for _, f := range sig.Factors {
		if f.Name == "趋势追踪" {
			f5 = f
			break
		}
	}
	if f5.RawScore < 1.0 {
		t.Errorf("expected bullish trend score >= 1.0, got %.1f", f5.RawScore)
	}

	// Bearish alignment
	bear := &model.MarketIndicators{
		CurrentPrice: 5000,
		MA200:        5500,
		MA20w:        5200,
		MA50w:        5400,
		WeeklyRSI:    50,
		DailyRSI:     50,
		High52w:      6000,
		Low52w:       4900,
		High30d:      5200,
		Low30d:       5000,
		Position52w:  0.1,
	}
	sig2 := Evaluate(bear)
	var f5b model.FactorScore
	for _, f := range sig2.Factors {
		if f.Name == "趋势追踪" {
			f5b = f
			break
		}
	}
	if f5b.RawScore > -0.5 {
		t.Errorf("expected bearish trend score <= -0.5, got %.1f", f5b.RawScore)
	}
}
