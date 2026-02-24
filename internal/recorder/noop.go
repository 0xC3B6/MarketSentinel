package recorder

// NoopRecorder is a no-op implementation used when SQLite is not configured.
type NoopRecorder struct{}

func NewNoopRecorder() *NoopRecorder { return &NoopRecorder{} }

func (n *NoopRecorder) RecordWeekly(_ *WeeklySnapshot) error    { return nil }
func (n *NoopRecorder) RecordDailyCheck(_ *DailyCheckEvent) error { return nil }
func (n *NoopRecorder) RecordFundEvent(_ *FundEvent) error       { return nil }
func (n *NoopRecorder) RecordMonthly(_ *MonthlyEvent) error      { return nil }
func (n *NoopRecorder) RecordQuarterly(_ *QuarterlyEvent) error  { return nil }
func (n *NoopRecorder) Close() error                             { return nil }
