package notifier

import (
	"fmt"
	"strings"
	"time"

	"MarketSentinel/internal/model"
)

// FormatWeeklyReport formats the weekly trade signal into a Telegram message.
func FormatWeeklyReport(ind *model.MarketIndicators, signal *model.TradeSignal) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("📊 <b>MarketSentinel 周报</b> | %s\n\n", time.Now().Format("2006-01-02")))

	// Price and MAs
	b.WriteString(fmt.Sprintf("当前价格: %.2f\n", ind.CurrentPrice))
	ma200Dev := 0.0
	if ind.MA200 > 0 {
		ma200Dev = (ind.CurrentPrice - ind.MA200) / ind.MA200 * 100
	}
	b.WriteString(fmt.Sprintf("MA200: %.2f (偏离 %+.1f%%)\n", ind.MA200, ma200Dev))
	b.WriteString(fmt.Sprintf("MA20周: %.2f | MA50周: %.2f\n\n", ind.MA20w, ind.MA50w))

	// Factor details
	b.WriteString("📈 <b>因子评分明细:</b>\n")
	for _, f := range signal.Factors {
		b.WriteString(fmt.Sprintf("  %s(%s): %+.0f (×%.2f) = %+.3f\n",
			f.Name, f.Commentary, f.RawScore, f.Weight, f.Weighted))
	}
	b.WriteString("  ─────────────────\n")
	b.WriteString(fmt.Sprintf("  综合评分: %+.3f\n\n", signal.TotalScore))

	// Action
	b.WriteString(fmt.Sprintf("💰 <b>本周操作:</b> %s %.2fx\n", signal.Tier.Label, signal.Tier.Multiplier))
	b.WriteString(fmt.Sprintf("   投入金额: ¥%.0f (基准¥%.0f)\n", signal.FinalAmount, signal.BaseAmount))
	if signal.ReserveUsed > 0 {
		b.WriteString(fmt.Sprintf("   储备金动用: ¥%.0f\n", signal.ReserveUsed))
	}

	// Warning
	if signal.WarningMsg != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", signal.WarningMsg))
	}

	return b.String()
}

// FormatFundStatus formats the current fund state for display.
func FormatFundStatus(state *model.FundState) string {
	var b strings.Builder
	b.WriteString("📦 <b>资金池状态</b>\n\n")
	b.WriteString(fmt.Sprintf("月度预算: ¥%.0f\n", state.MonthlyBudget))
	b.WriteString(fmt.Sprintf("周基准N: ¥%.0f\n", state.WeeklyBaseN))
	b.WriteString(fmt.Sprintf("常规池: ¥%.0f\n", state.RegularBalance))
	b.WriteString(fmt.Sprintf("储备池: ¥%.0f\n", state.ReserveBalance))
	b.WriteString(fmt.Sprintf("本周已抄底: %v\n", state.BottomFishUsedThisWeek))
	b.WriteString(fmt.Sprintf("连续高分周数: %d\n", state.ConsecutiveHighScoreWeeks))
	b.WriteString(fmt.Sprintf("更新时间: %s\n", state.UpdatedAt.Format("2006-01-02 15:04")))
	return b.String()
}

// FormatMonthlySummary formats a monthly summary report.
func FormatMonthlySummary(state *model.FundState) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📅 <b>月度汇总</b> | %s\n\n", time.Now().Format("2006-01")))
	b.WriteString(fmt.Sprintf("常规池余额: ¥%.0f\n", state.RegularBalance))
	b.WriteString(fmt.Sprintf("储备池余额: ¥%.0f\n", state.ReserveBalance))

	if len(state.RecentScores) > 0 {
		sum := 0.0
		for _, s := range state.RecentScores {
			sum += s
		}
		avg := sum / float64(len(state.RecentScores))
		b.WriteString(fmt.Sprintf("近期平均评分: %+.3f (%d周)\n", avg, len(state.RecentScores)))
	}

	b.WriteString("\n已完成月度资金补充 ✅")
	return b.String()
}
