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

// YahooFetcher implements Fetcher using Yahoo Finance public API.
type YahooFetcher struct {
	Client    *http.Client
	SymbolMap map[string]string // maps internal symbol to Yahoo ticker
}

// NewYahooFetcher creates a new Yahoo Finance fetcher.
func NewYahooFetcher(proxyURL string) *YahooFetcher {
	transport := &http.Transport{}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &YahooFetcher{
		Client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		SymbolMap: map[string]string{
			"SPX500": "^GSPC",
			"SPX":    "^GSPC",
			"SP500":  "^GSPC",
		},
	}
}

func (f *YahooFetcher) Name() string { return "yahoo" }

func (f *YahooFetcher) yahooSymbol(symbol string) string {
	if mapped, ok := f.SymbolMap[symbol]; ok {
		return mapped
	}
	return symbol
}

// yahooChart is the response structure from Yahoo Finance chart API.
type yahooChart struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []interface{} `json:"open"`
					High   []interface{} `json:"high"`
					Low    []interface{} `json:"low"`
					Close  []interface{} `json:"close"`
					Volume []interface{} `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

func toFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return 0
	}
}

func (f *YahooFetcher) fetchChart(symbol, interval, rng string) ([]model.OHLCV, error) {
	u := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=%s&range=%s",
		url.PathEscape(f.yahooSymbol(symbol)), interval, rng)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("yahoo read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo: status %d, body: %s", resp.StatusCode, string(body))
	}

	var chart yahooChart
	if err := json.Unmarshal(body, &chart); err != nil {
		return nil, fmt.Errorf("yahoo decode: %w", err)
	}
	if chart.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo api error: %s", chart.Chart.Error.Description)
	}
	if len(chart.Chart.Result) == 0 || len(chart.Chart.Result[0].Timestamp) == 0 {
		return nil, fmt.Errorf("yahoo: no data returned")
	}

	result := chart.Chart.Result[0]
	quote := result.Indicators.Quote[0]
	bars := make([]model.OHLCV, 0, len(result.Timestamp))

	for i, ts := range result.Timestamp {
		o := toFloat(quote.Open[i])
		h := toFloat(quote.High[i])
		l := toFloat(quote.Low[i])
		c := toFloat(quote.Close[i])
		if o == 0 && h == 0 && l == 0 && c == 0 {
			continue // skip null bars (holidays etc.)
		}
		bars = append(bars, model.OHLCV{
			Time:   time.Unix(ts, 0),
			Open:   o,
			High:   h,
			Low:    l,
			Close:  c,
			Volume: toFloat(quote.Volume[i]),
		})
	}

	sort.Slice(bars, func(i, j int) bool { return bars[i].Time.Before(bars[j].Time) })
	return bars, nil
}

func (f *YahooFetcher) FetchDailyBars(symbol string, days int) ([]model.OHLCV, error) {
	// Yahoo range: max "2y" for daily interval
	rng := "2y"
	if days <= 30 {
		rng = "1mo"
	} else if days <= 90 {
		rng = "3mo"
	} else if days <= 180 {
		rng = "6mo"
	} else if days <= 365 {
		rng = "1y"
	}
	bars, err := f.fetchChart(symbol, "1d", rng)
	if err != nil {
		return nil, err
	}
	// Trim to requested count
	if len(bars) > days {
		bars = bars[len(bars)-days:]
	}
	return bars, nil
}

func (f *YahooFetcher) FetchWeeklyBars(symbol string, weeks int) ([]model.OHLCV, error) {
	rng := "2y"
	if weeks <= 26 {
		rng = "6mo"
	} else if weeks <= 52 {
		rng = "1y"
	}
	bars, err := f.fetchChart(symbol, "1wk", rng)
	if err != nil {
		return nil, err
	}
	if len(bars) > weeks {
		bars = bars[len(bars)-weeks:]
	}
	return bars, nil
}

func (f *YahooFetcher) FetchCurrentPrice(symbol string) (float64, error) {
	bars, err := f.fetchChart(symbol, "1d", "1d")
	if err != nil {
		return 0, err
	}
	if len(bars) == 0 {
		return 0, fmt.Errorf("yahoo: no price data")
	}
	return bars[len(bars)-1].Close, nil
}
