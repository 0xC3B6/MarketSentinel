package model

// TriggerType indicates what triggered the signal.
type TriggerType string

const (
	TriggerWeekly    TriggerType = "WEEKLY"
	TriggerBottomFish TriggerType = "BOTTOM_FISH"
	TriggerTakeProfit TriggerType = "TAKE_PROFIT"
	TriggerMonthly   TriggerType = "MONTHLY"
	TriggerQuarterly TriggerType = "QUARTERLY"
	TriggerManual    TriggerType = "MANUAL"
)

// FactorScore represents a single factor's scoring result.
type FactorScore struct {
	Name       string
	RawScore   float64
	Weight     float64
	Weighted   float64
	Commentary string
}

// InvestmentTier maps a total score range to an action.
type InvestmentTier struct {
	Label      string
	Multiplier float64
	UseReserve float64 // extra multiplier from reserve pool, 0 means none
}

// TradeSignal is the final output of the strategy engine.
type TradeSignal struct {
	Factors     []FactorScore
	TotalScore  float64
	Tier        InvestmentTier
	BaseAmount  float64
	FinalAmount float64
	ReserveUsed float64
	TriggerType TriggerType
	WarningMsg  string
}
