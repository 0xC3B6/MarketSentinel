package fund

import (
	"log"
	"sync"

	"MarketSentinel/internal/model"
)

// Manager handles dual-pool fund operations with concurrency safety.
type Manager struct {
	mu       sync.Mutex
	state    *model.FundState
	filePath string
}

// NewManager creates a Manager, loading or initializing state from disk.
func NewManager(filePath string, monthlyBudget float64) (*Manager, error) {
	state, err := LoadState(filePath)
	if err != nil {
		return nil, err
	}

	// Initialize if fresh state
	if state.MonthlyBudget == 0 {
		weeklyBase := monthlyBudget * 0.70 / 4.33
		state.MonthlyBudget = monthlyBudget
		state.WeeklyBaseN = weeklyBase
		state.RegularBalance = monthlyBudget * 0.70
		state.ReserveBalance = monthlyBudget * 0.30
	}

	m := &Manager{state: state, filePath: filePath}
	if err := m.save(); err != nil {
		return nil, err
	}
	return m, nil
}

// GetState returns a copy of the current fund state.
func (m *Manager) GetState() model.FundState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return *m.state
}

// CalculateWeeklyInvestment computes the weekly investment amount based on the signal tier.
func (m *Manager) CalculateWeeklyInvestment(signal *model.TradeSignal) (finalAmount, reserveUsed float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	baseN := m.state.WeeklyBaseN
	regularAmount := baseN * signal.Tier.Multiplier
	reserveAmount := baseN * signal.Tier.UseReserve

	// Cap to available balances
	if regularAmount > m.state.RegularBalance {
		regularAmount = m.state.RegularBalance
	}
	if reserveAmount > m.state.ReserveBalance {
		reserveAmount = m.state.ReserveBalance
	}

	m.state.RegularBalance -= regularAmount
	m.state.ReserveBalance -= reserveAmount

	// Track score
	m.state.RecentScores = append(m.state.RecentScores, signal.TotalScore)
	if len(m.state.RecentScores) > 12 {
		m.state.RecentScores = m.state.RecentScores[len(m.state.RecentScores)-12:]
	}

	// Track consecutive high-score weeks
	if signal.TotalScore > 1.0 {
		m.state.ConsecutiveHighScoreWeeks++
	} else {
		m.state.ConsecutiveHighScoreWeeks = 0
	}

	if err := m.save(); err != nil {
		log.Printf("[ERROR] failed to save fund state: %v", err)
	}

	return regularAmount + reserveAmount, reserveAmount
}

// CalculateBottomFishInvestment handles intra-week RSI<30 bottom-fishing.
// Only triggers once per week, funded from reserve pool.
func (m *Manager) CalculateBottomFishInvestment(totalScore float64) (amount float64, triggered bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state.BottomFishUsedThisWeek {
		return 0, false
	}

	baseN := m.state.WeeklyBaseN
	var multiplier float64
	switch {
	case totalScore > 1.0:
		multiplier = 1.5
	case totalScore > 0:
		multiplier = 1.0
	case totalScore < -0.5:
		multiplier = 0.5
	default:
		multiplier = 0.75
	}

	amount = baseN * multiplier
	if amount > m.state.ReserveBalance {
		amount = m.state.ReserveBalance
	}

	m.state.ReserveBalance -= amount
	m.state.BottomFishUsedThisWeek = true

	if err := m.save(); err != nil {
		log.Printf("[ERROR] failed to save fund state: %v", err)
	}

	return amount, true
}

// MonthlyReplenish refills both pools from the monthly budget.
func (m *Manager) MonthlyReplenish() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.RegularBalance += m.state.MonthlyBudget * 0.70
	m.state.ReserveBalance += m.state.MonthlyBudget * 0.30

	if err := m.save(); err != nil {
		log.Printf("[ERROR] failed to save fund state after monthly replenish: %v", err)
	}
}

// QuarterlyRebalance adjusts the reserve pool:
// - If reserve > 6N, transfer excess back to regular pool
// - If consecutive 4+ weeks score > 1.0 and reserve < 3N, emergency top-up
func (m *Manager) QuarterlyRebalance() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	baseN := m.state.WeeklyBaseN
	var msg string

	if m.state.ReserveBalance > 6*baseN {
		excess := m.state.ReserveBalance - 6*baseN
		m.state.ReserveBalance -= excess
		m.state.RegularBalance += excess
		msg = "储备池超额，已转回常规池"
	} else if m.state.ConsecutiveHighScoreWeeks >= 4 && m.state.ReserveBalance < 3*baseN {
		topUp := 3*baseN - m.state.ReserveBalance
		m.state.ReserveBalance += topUp
		msg = "连续高分+储备不足，紧急补充储备池"
	} else {
		msg = "季度再平衡：无需调整"
	}

	if err := m.save(); err != nil {
		log.Printf("[ERROR] failed to save fund state after quarterly rebalance: %v", err)
	}

	return msg
}

// ResetWeeklyFlags resets per-week flags (called every Monday).
func (m *Manager) ResetWeeklyFlags() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.BottomFishUsedThisWeek = false

	if err := m.save(); err != nil {
		log.Printf("[ERROR] failed to save fund state after weekly reset: %v", err)
	}
}

func (m *Manager) save() error {
	return SaveState(m.filePath, m.state)
}
