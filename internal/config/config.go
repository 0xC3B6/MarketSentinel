package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Telegram struct {
		BotToken string `yaml:"bot_token"`
		ChatID   string `yaml:"chat_id"`
	} `yaml:"telegram"`
	DataSource struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		Symbol  string `yaml:"symbol"`
	} `yaml:"data_source"`
	Schedule struct {
		WeeklyCron  string `yaml:"weekly_cron"`
		DailyCron   string `yaml:"daily_cron"`
		MonthlyCron string `yaml:"monthly_cron"`
	} `yaml:"schedule"`
	Fund struct {
		MonthlyBudget float64 `yaml:"monthly_budget"`
		StateFile     string  `yaml:"state_file"`
	} `yaml:"fund"`
	Database struct {
		SQLitePath string `yaml:"sqlite_path"`
	} `yaml:"database"`
	Proxy string `yaml:"proxy"`
}

// Load reads config from a YAML file, then applies environment variable overrides.
func Load(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}

	// Environment variable overrides
	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
		cfg.Telegram.BotToken = v
	}
	if v := os.Getenv("TELEGRAM_CHAT_ID"); v != "" {
		cfg.Telegram.ChatID = v
	}
	if v := os.Getenv("VSTRADER_BASE_URL"); v != "" {
		cfg.DataSource.BaseURL = v
	}
	if v := os.Getenv("VSTRADER_API_KEY"); v != "" {
		cfg.DataSource.APIKey = v
	}
	if v := os.Getenv("HTTPS_PROXY"); v != "" {
		cfg.Proxy = v
	}
	if v := os.Getenv("MONTHLY_BUDGET"); v != "" {
		var budget float64
		if _, err := fmt.Sscanf(v, "%f", &budget); err == nil {
			cfg.Fund.MonthlyBudget = budget
		}
	}
	if v := os.Getenv("CRON_WEEKLY"); v != "" {
		cfg.Schedule.WeeklyCron = v
	}
	if v := os.Getenv("SQLITE_PATH"); v != "" {
		cfg.Database.SQLitePath = v
	}

	// Defaults
	if cfg.DataSource.Symbol == "" {
		cfg.DataSource.Symbol = "SPX500"
	}
	if cfg.Schedule.WeeklyCron == "" {
		cfg.Schedule.WeeklyCron = "0 0 8 * * 1"
	}
	if cfg.Schedule.DailyCron == "" {
		cfg.Schedule.DailyCron = "0 0 22 * * 1-5"
	}
	if cfg.Schedule.MonthlyCron == "" {
		cfg.Schedule.MonthlyCron = "0 0 9 1 * *"
	}
	if cfg.Fund.MonthlyBudget == 0 {
		cfg.Fund.MonthlyBudget = 10000
	}
	if cfg.Fund.StateFile == "" {
		cfg.Fund.StateFile = "data/fund_state.json"
	}
	if cfg.Database.SQLitePath == "" {
		cfg.Database.SQLitePath = "data/market_sentinel.db"
	}

	return cfg, nil
}

// Validate checks that all required fields are set.
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("telegram.bot_token is required")
	}
	if c.Telegram.ChatID == "" {
		return fmt.Errorf("telegram.chat_id is required")
	}
	if c.DataSource.BaseURL == "" {
		return fmt.Errorf("data_source.base_url is required")
	}
	if c.Fund.MonthlyBudget <= 0 {
		return fmt.Errorf("fund.monthly_budget must be positive")
	}
	return nil
}
