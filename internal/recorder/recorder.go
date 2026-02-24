package recorder

import "MarketSentinel/internal/model"

// WeeklySnapshot holds all data for a weekly evaluation record.
type WeeklySnapshot struct {
	Indicators  *model.MarketIndicators
	Signal      *model.TradeSignal
	FundState   *model.FundState
}

// DailyCheckEvent holds data for a daily RSI trigger event.
type DailyCheckEvent struct {
	DailyRSI    float64
	WeeklyRSI   float64
	Price       float64
	EventType   string // "BOTTOM_FISH" or "TAKE_PROFIT"
	Amount      float64
	TotalScore  float64
}

// FundEvent records a fund balance change.
type FundEvent struct {
	EventType      string // "WEEKLY", "BOTTOM_FISH", "MONTHLY", "QUARTERLY"
	RegularBefore  float64
	RegularAfter   float64
	ReserveBefore  float64
	ReserveAfter   float64
	Amount         float64
	Note           string
}

// MonthlyEvent records a monthly replenishment.
type MonthlyEvent struct {
	RegularAdded  float64
	ReserveAdded  float64
	RegularAfter  float64
	ReserveAfter  float64
	AvgScore      float64
}

// QuarterlyEvent records a quarterly rebalance.
type QuarterlyEvent struct {
	Action        string // "TRANSFER_EXCESS", "EMERGENCY_TOPUP", "NO_ACTION"
	Amount        float64
	RegularAfter  float64
	ReserveAfter  float64
	Note          string
}

// Recorder persists historical data for analysis.
type Recorder interface {
	RecordWeekly(snap *WeeklySnapshot) error
	RecordDailyCheck(evt *DailyCheckEvent) error
	RecordFundEvent(evt *FundEvent) error
	RecordMonthly(evt *MonthlyEvent) error
	RecordQuarterly(evt *QuarterlyEvent) error
	Close() error
}
