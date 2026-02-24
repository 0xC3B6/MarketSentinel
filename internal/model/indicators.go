package model

// MarketIndicators holds all computed technical indicators.
type MarketIndicators struct {
	CurrentPrice float64
	MA200        float64
	MA20w        float64
	MA50w        float64
	WeeklyRSI    float64
	DailyRSI     float64
	High52w      float64
	Low52w       float64
	High30d      float64
	Low30d       float64
	Position52w  float64 // 0.0 ~ 1.0
}
