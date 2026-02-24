package collector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"MarketSentinel/internal/model"
)

// VsTraderFetcher implements Fetcher using the vstrader REST API.
type VsTraderFetcher struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// NewVsTraderFetcher creates a new fetcher with optional proxy support.
func NewVsTraderFetcher(baseURL, apiKey, proxyURL string) *VsTraderFetcher {
	transport := &http.Transport{}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &VsTraderFetcher{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

func (f *VsTraderFetcher) Name() string { return "vstrader" }

// vsBar is the expected JSON shape from the vstrader API.
type vsBar struct {
	Timestamp int64   `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
}

func (f *VsTraderFetcher) FetchDailyBars(symbol string, days int) ([]model.OHLCV, error) {
	endpoint := fmt.Sprintf("%s/api/v1/bars/daily?symbol=%s&limit=%d", f.BaseURL, symbol, days)
	return f.fetchBars(endpoint)
}

func (f *VsTraderFetcher) FetchWeeklyBars(symbol string, weeks int) ([]model.OHLCV, error) {
	// Try weekly endpoint first; if API only provides daily, aggregate internally.
	endpoint := fmt.Sprintf("%s/api/v1/bars/weekly?symbol=%s&limit=%d", f.BaseURL, symbol, weeks)
	bars, err := f.fetchBars(endpoint)
	if err != nil {
		// Fallback: fetch enough daily bars and aggregate to weekly
		dailyBars, dailyErr := f.FetchDailyBars(symbol, weeks*7)
		if dailyErr != nil {
			return nil, fmt.Errorf("weekly fetch failed: %w; daily fallback also failed: %w", err, dailyErr)
		}
		return aggregateDailyToWeekly(dailyBars), nil
	}
	return bars, nil
}

func (f *VsTraderFetcher) FetchCurrentPrice(symbol string) (float64, error) {
	endpoint := fmt.Sprintf("%s/api/v1/quote?symbol=%s", f.BaseURL, symbol)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, err
	}
	if f.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+f.APIKey)
	}
	resp, err := f.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch current price: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fetch current price: status %d", resp.StatusCode)
	}
	var result struct {
		Price float64 `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode price: %w", err)
	}
	return result.Price, nil
}

func (f *VsTraderFetcher) fetchBars(endpoint string) ([]model.OHLCV, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	if f.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+f.APIKey)
	}
	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch bars: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch bars: status %d, body: %s", resp.StatusCode, string(body))
	}
	var vsBars []vsBar
	if err := json.NewDecoder(resp.Body).Decode(&vsBars); err != nil {
		return nil, fmt.Errorf("decode bars: %w", err)
	}
	bars := make([]model.OHLCV, len(vsBars))
	for i, vb := range vsBars {
		bars[i] = model.OHLCV{
			Time:   time.Unix(vb.Timestamp, 0),
			Open:   vb.Open,
			High:   vb.High,
			Low:    vb.Low,
			Close:  vb.Close,
			Volume: vb.Volume,
		}
	}
	// Ensure chronological order
	sort.Slice(bars, func(i, j int) bool { return bars[i].Time.Before(bars[j].Time) })
	return bars, nil
}

// aggregateDailyToWeekly converts daily bars into weekly bars (Mon-Fri).
func aggregateDailyToWeekly(daily []model.OHLCV) []model.OHLCV {
	if len(daily) == 0 {
		return nil
	}
	var weekly []model.OHLCV
	var week model.OHLCV
	var weekStarted bool

	for _, d := range daily {
		year, isoWeek := d.Time.ISOWeek()
		weekKey := year*100 + isoWeek

		if !weekStarted {
			week = model.OHLCV{Time: d.Time, Open: d.Open, High: d.High, Low: d.Low, Close: d.Close, Volume: d.Volume}
			weekStarted = true
			continue
		}

		cy, cw := week.Time.ISOWeek()
		currentKey := cy*100 + cw

		if weekKey != currentKey {
			weekly = append(weekly, week)
			week = model.OHLCV{Time: d.Time, Open: d.Open, High: d.High, Low: d.Low, Close: d.Close, Volume: d.Volume}
		} else {
			if d.High > week.High {
				week.High = d.High
			}
			if d.Low < week.Low {
				week.Low = d.Low
			}
			week.Close = d.Close
			week.Volume += d.Volume
		}
	}
	if weekStarted {
		weekly = append(weekly, week)
	}
	return weekly
}
