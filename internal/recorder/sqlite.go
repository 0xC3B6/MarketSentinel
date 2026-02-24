package recorder

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteRecorder persists historical data to a SQLite database.
type SQLiteRecorder struct {
	db *sql.DB
	mu sync.Mutex
}

// NewSQLiteRecorder opens (or creates) the SQLite database and runs migrations.
func NewSQLiteRecorder(dbPath string) (*SQLiteRecorder, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// WAL mode for better concurrent read performance (Grafana reads while bot writes).
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	r := &SQLiteRecorder{db: db}
	if err := r.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Printf("[INFO] sqlite recorder opened: %s", dbPath)
	return r, nil
}

func (r *SQLiteRecorder) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS weekly_snapshots (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp       INTEGER NOT NULL,
			current_price   REAL,
			ma200           REAL,
			ma20w           REAL,
			ma50w           REAL,
			weekly_rsi      REAL,
			daily_rsi       REAL,
			high_52w        REAL,
			low_52w         REAL,
			position_52w    REAL,
			factor1_score   REAL,
			factor2_score   REAL,
			factor3_score   REAL,
			factor4_score   REAL,
			factor5_score   REAL,
			total_score     REAL,
			tier_label      TEXT,
			tier_multiplier REAL,
			tier_reserve    REAL,
			base_amount     REAL,
			final_amount    REAL,
			reserve_used    REAL,
			regular_balance REAL,
			reserve_balance REAL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_weekly_ts ON weekly_snapshots(timestamp)`,

		`CREATE TABLE IF NOT EXISTS daily_checks (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp   INTEGER NOT NULL,
			daily_rsi   REAL,
			weekly_rsi  REAL,
			price       REAL,
			event_type  TEXT,
			amount      REAL,
			total_score REAL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_ts ON daily_checks(timestamp)`,

		`CREATE TABLE IF NOT EXISTS fund_history (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp      INTEGER NOT NULL,
			event_type     TEXT,
			regular_before REAL,
			regular_after  REAL,
			reserve_before REAL,
			reserve_after  REAL,
			amount         REAL,
			note           TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fund_ts ON fund_history(timestamp)`,

		`CREATE TABLE IF NOT EXISTS monthly_events (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp     INTEGER NOT NULL,
			regular_added REAL,
			reserve_added REAL,
			regular_after REAL,
			reserve_after REAL,
			avg_score     REAL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_monthly_ts ON monthly_events(timestamp)`,

		`CREATE TABLE IF NOT EXISTS quarterly_events (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp     INTEGER NOT NULL,
			action        TEXT,
			amount        REAL,
			regular_after REAL,
			reserve_after REAL,
			note          TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_quarterly_ts ON quarterly_events(timestamp)`,
	}

	for _, s := range stmts {
		if _, err := r.db.Exec(s); err != nil {
			return fmt.Errorf("exec %q: %w", s[:40], err)
		}
	}
	return nil
}

func (r *SQLiteRecorder) RecordWeekly(snap *WeeklySnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().Unix()
	ind := snap.Indicators
	sig := snap.Signal
	fs := snap.FundState

	// Extract per-factor weighted scores (up to 5).
	factors := make([]float64, 5)
	for i := 0; i < len(sig.Factors) && i < 5; i++ {
		factors[i] = sig.Factors[i].Weighted
	}

	_, err := r.db.Exec(`INSERT INTO weekly_snapshots
		(timestamp, current_price, ma200, ma20w, ma50w, weekly_rsi, daily_rsi,
		 high_52w, low_52w, position_52w,
		 factor1_score, factor2_score, factor3_score, factor4_score, factor5_score,
		 total_score, tier_label, tier_multiplier, tier_reserve,
		 base_amount, final_amount, reserve_used,
		 regular_balance, reserve_balance)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		now, ind.CurrentPrice, ind.MA200, ind.MA20w, ind.MA50w,
		ind.WeeklyRSI, ind.DailyRSI, ind.High52w, ind.Low52w, ind.Position52w,
		factors[0], factors[1], factors[2], factors[3], factors[4],
		sig.TotalScore, sig.Tier.Label, sig.Tier.Multiplier, sig.Tier.UseReserve,
		sig.BaseAmount, sig.FinalAmount, sig.ReserveUsed,
		fs.RegularBalance, fs.ReserveBalance,
	)
	return err
}

func (r *SQLiteRecorder) RecordDailyCheck(evt *DailyCheckEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec(`INSERT INTO daily_checks
		(timestamp, daily_rsi, weekly_rsi, price, event_type, amount, total_score)
		VALUES (?,?,?,?,?,?,?)`,
		time.Now().Unix(), evt.DailyRSI, evt.WeeklyRSI, evt.Price,
		evt.EventType, evt.Amount, evt.TotalScore,
	)
	return err
}

func (r *SQLiteRecorder) RecordFundEvent(evt *FundEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec(`INSERT INTO fund_history
		(timestamp, event_type, regular_before, regular_after, reserve_before, reserve_after, amount, note)
		VALUES (?,?,?,?,?,?,?,?)`,
		time.Now().Unix(), evt.EventType,
		evt.RegularBefore, evt.RegularAfter,
		evt.ReserveBefore, evt.ReserveAfter,
		evt.Amount, evt.Note,
	)
	return err
}

func (r *SQLiteRecorder) RecordMonthly(evt *MonthlyEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec(`INSERT INTO monthly_events
		(timestamp, regular_added, reserve_added, regular_after, reserve_after, avg_score)
		VALUES (?,?,?,?,?,?)`,
		time.Now().Unix(), evt.RegularAdded, evt.ReserveAdded,
		evt.RegularAfter, evt.ReserveAfter, evt.AvgScore,
	)
	return err
}

func (r *SQLiteRecorder) RecordQuarterly(evt *QuarterlyEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec(`INSERT INTO quarterly_events
		(timestamp, action, amount, regular_after, reserve_after, note)
		VALUES (?,?,?,?,?,?)`,
		time.Now().Unix(), evt.Action, evt.Amount,
		evt.RegularAfter, evt.ReserveAfter, evt.Note,
	)
	return err
}

func (r *SQLiteRecorder) Close() error {
	log.Println("[INFO] closing sqlite recorder")
	return r.db.Close()
}
