package scheduler

import (
	"context"
	"fmt"
	"log"

	"MarketSentinel/internal/collector"
	"MarketSentinel/internal/fund"
	"MarketSentinel/internal/model"
	"MarketSentinel/internal/notifier"
	"MarketSentinel/internal/recorder"
	"MarketSentinel/internal/strategy"

	"github.com/robfig/cron/v3"
)

// Scheduler manages all cron tasks.
type Scheduler struct {
	Cron      *cron.Cron
	Collector *collector.Collector
	Fund      *fund.Manager
	Notifier  *notifier.TelegramNotifier
	Recorder  recorder.Recorder
	Ctx       context.Context
}

// NewScheduler creates a new Scheduler.
func NewScheduler(ctx context.Context, col *collector.Collector, fm *fund.Manager, tn *notifier.TelegramNotifier, rec recorder.Recorder) *Scheduler {
	return &Scheduler{
		Cron:      cron.New(cron.WithSeconds()),
		Collector: col,
		Fund:      fm,
		Notifier:  tn,
		Recorder:  rec,
		Ctx:       ctx,
	}
}

// RegisterAll registers weekly, daily, monthly, and quarterly tasks.
func (s *Scheduler) RegisterAll(weeklyCron, dailyCron, monthlyCron string) error {
	if _, err := s.Cron.AddFunc(weeklyCron, s.weeklyTask); err != nil {
		return fmt.Errorf("register weekly task: %w", err)
	}
	if _, err := s.Cron.AddFunc(dailyCron, s.dailyCheck); err != nil {
		return fmt.Errorf("register daily task: %w", err)
	}
	if _, err := s.Cron.AddFunc(monthlyCron, s.monthlyTask); err != nil {
		return fmt.Errorf("register monthly task: %w", err)
	}
	// Quarterly: 1st of Jan, Apr, Jul, Oct
	if _, err := s.Cron.AddFunc("0 0 9 1 1,4,7,10 *", s.quarterlyTask); err != nil {
		return fmt.Errorf("register quarterly task: %w", err)
	}
	// Weekly flag reset: every Monday 00:00
	if _, err := s.Cron.AddFunc("0 0 0 * * 1", func() {
		s.Fund.ResetWeeklyFlags()
		log.Println("[INFO] weekly flags reset")
	}); err != nil {
		return fmt.Errorf("register weekly reset: %w", err)
	}
	return nil
}

// Start starts the cron scheduler.
func (s *Scheduler) Start() {
	s.Cron.Start()
	log.Println("[INFO] scheduler started")
}

// Stop stops the cron scheduler gracefully.
func (s *Scheduler) Stop() {
	s.Cron.Stop()
	log.Println("[INFO] scheduler stopped")
}

// RunWeeklyNow executes the weekly task immediately (for manual trigger / RUN_ON_START).
func (s *Scheduler) RunWeeklyNow() {
	s.weeklyTask()
}

func (s *Scheduler) weeklyTask() {
	log.Println("[INFO] running weekly task")
	ind, err := s.Collector.Collect()
	if err != nil {
		log.Printf("[ERROR] weekly collect: %v", err)
		s.trySend(fmt.Sprintf("❌ 周任务数据采集失败: %v", err))
		return
	}

	signal := strategy.Evaluate(ind)
	signal.TriggerType = model.TriggerWeekly

	state := s.Fund.GetState()
	signal.BaseAmount = state.WeeklyBaseN

	stateBefore := s.Fund.GetState()
	finalAmount, reserveUsed := s.Fund.CalculateWeeklyInvestment(signal)
	signal.FinalAmount = finalAmount
	signal.ReserveUsed = reserveUsed

	report := notifier.FormatWeeklyReport(ind, signal)

	// Append fund status
	updatedState := s.Fund.GetState()
	report += "\n" + notifier.FormatFundStatus(&updatedState)

	s.trySend(report)

	// Record to SQLite
	if err := s.Recorder.RecordWeekly(&recorder.WeeklySnapshot{
		Indicators: ind,
		Signal:     signal,
		FundState:  &updatedState,
	}); err != nil {
		log.Printf("[ERROR] record weekly: %v", err)
	}
	s.recordFundEvent("WEEKLY", &stateBefore, &updatedState, finalAmount+reserveUsed, "周定投")
}

func (s *Scheduler) dailyCheck() {
	log.Println("[INFO] running daily check")
	ind, err := s.Collector.Collect()
	if err != nil {
		log.Printf("[ERROR] daily collect: %v", err)
		return
	}

	// Bottom-fish trigger: daily RSI < 30
	if ind.DailyRSI < 30 {
		signal := strategy.Evaluate(ind)
		stateBefore := s.Fund.GetState()
		amount, triggered := s.Fund.CalculateBottomFishInvestment(signal.TotalScore)
		if triggered {
			msg := fmt.Sprintf("🎣 <b>抄底触发</b> | 日线RSI=%.0f\n\n综合评分: %+.3f\n抄底金额: ¥%.0f (储备池)\n",
				ind.DailyRSI, signal.TotalScore, amount)
			s.trySend(msg)

			stateAfter := s.Fund.GetState()
			if err := s.Recorder.RecordDailyCheck(&recorder.DailyCheckEvent{
				DailyRSI: ind.DailyRSI, WeeklyRSI: ind.WeeklyRSI, Price: ind.CurrentPrice,
				EventType: "BOTTOM_FISH", Amount: amount, TotalScore: signal.TotalScore,
			}); err != nil {
				log.Printf("[ERROR] record daily check: %v", err)
			}
			s.recordFundEvent("BOTTOM_FISH", &stateBefore, &stateAfter, amount, "抄底触发")
		}
	}

	// Take-profit warning: RSI > 85
	if ind.DailyRSI > 85 || ind.WeeklyRSI > 85 {
		msg := fmt.Sprintf("⚠️ <b>止盈预警</b>\n\n日线RSI: %.0f | 周线RSI: %.0f\n当前价格: %.2f\n建议考虑部分止盈",
			ind.DailyRSI, ind.WeeklyRSI, ind.CurrentPrice)
		s.trySend(msg)

		if err := s.Recorder.RecordDailyCheck(&recorder.DailyCheckEvent{
			DailyRSI: ind.DailyRSI, WeeklyRSI: ind.WeeklyRSI, Price: ind.CurrentPrice,
			EventType: "TAKE_PROFIT",
		}); err != nil {
			log.Printf("[ERROR] record daily check: %v", err)
		}
	}
}

func (s *Scheduler) monthlyTask() {
	log.Println("[INFO] running monthly task")
	stateBefore := s.Fund.GetState()
	s.Fund.MonthlyReplenish()
	state := s.Fund.GetState()
	report := notifier.FormatMonthlySummary(&state)
	s.trySend(report)

	budget := state.MonthlyBudget
	regularAdded := budget * 0.7
	reserveAdded := budget * 0.3
	var avgScore float64
	if len(state.RecentScores) > 0 {
		sum := 0.0
		for _, sc := range state.RecentScores {
			sum += sc
		}
		avgScore = sum / float64(len(state.RecentScores))
	}
	if err := s.Recorder.RecordMonthly(&recorder.MonthlyEvent{
		RegularAdded: regularAdded, ReserveAdded: reserveAdded,
		RegularAfter: state.RegularBalance, ReserveAfter: state.ReserveBalance,
		AvgScore: avgScore,
	}); err != nil {
		log.Printf("[ERROR] record monthly: %v", err)
	}
	s.recordFundEvent("MONTHLY", &stateBefore, &state, budget, "月度补充")
}

func (s *Scheduler) quarterlyTask() {
	log.Println("[INFO] running quarterly rebalance")
	stateBefore := s.Fund.GetState()
	result := s.Fund.QuarterlyRebalance()
	state := s.Fund.GetState()
	msg := fmt.Sprintf("📊 <b>季度再平衡</b>\n\n%s\n\n%s", result, notifier.FormatFundStatus(&state))
	s.trySend(msg)

	action := "NO_ACTION"
	var amount float64
	if state.ReserveBalance < stateBefore.ReserveBalance {
		action = "TRANSFER_EXCESS"
		amount = stateBefore.ReserveBalance - state.ReserveBalance
	} else if state.ReserveBalance > stateBefore.ReserveBalance {
		action = "EMERGENCY_TOPUP"
		amount = state.ReserveBalance - stateBefore.ReserveBalance
	}
	if err := s.Recorder.RecordQuarterly(&recorder.QuarterlyEvent{
		Action: action, Amount: amount,
		RegularAfter: state.RegularBalance, ReserveAfter: state.ReserveBalance,
		Note: result,
	}); err != nil {
		log.Printf("[ERROR] record quarterly: %v", err)
	}
	s.recordFundEvent("QUARTERLY", &stateBefore, &state, amount, "季度再平衡")
}

// HandleCommand processes a user command and returns a reply.
func (s *Scheduler) HandleCommand(command string) string {
	switch command {
	case "查看本周建议", "/weekly":
		s.weeklyTask()
		return ""
	case "查看资金状态", "/fund":
		state := s.Fund.GetState()
		return notifier.FormatFundStatus(&state)
	case "查看月报", "/monthly":
		state := s.Fund.GetState()
		return notifier.FormatMonthlySummary(&state)
	default:
		return "可用命令:\n• 查看本周建议\n• 查看资金状态\n• 查看月报"
	}
}

func (s *Scheduler) recordFundEvent(eventType string, before, after *model.FundState, amount float64, note string) {
	if err := s.Recorder.RecordFundEvent(&recorder.FundEvent{
		EventType:     eventType,
		RegularBefore: before.RegularBalance,
		RegularAfter:  after.RegularBalance,
		ReserveBefore: before.ReserveBalance,
		ReserveAfter:  after.ReserveBalance,
		Amount:        amount,
		Note:          note,
	}); err != nil {
		log.Printf("[ERROR] record fund event: %v", err)
	}
}

func (s *Scheduler) trySend(text string) {
	if err := s.Notifier.SendWithRetry(s.Ctx, text, 3); err != nil {
		log.Printf("[ERROR] send notification: %v", err)
	}
}
