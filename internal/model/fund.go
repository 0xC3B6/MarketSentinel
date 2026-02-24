package model

import "time"

// FundState tracks the dual-pool fund status.
type FundState struct {
	MonthlyBudget             float64   `json:"monthly_budget"`
	WeeklyBaseN               float64   `json:"weekly_base_n"`
	RegularBalance            float64   `json:"regular_balance"`
	ReserveBalance            float64   `json:"reserve_balance"`
	BottomFishUsedThisWeek    bool      `json:"bottom_fish_used_this_week"`
	ConsecutiveHighScoreWeeks int       `json:"consecutive_high_score_weeks"`
	RecentScores              []float64 `json:"recent_scores"`
	LastReplenishAt           time.Time `json:"last_replenish_at"`
	LastRebalanceAt           time.Time `json:"last_rebalance_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}
