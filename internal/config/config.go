package config

import (
	"time"

	"orderbook/internal/exchange"
	"orderbook/internal/types"
)

// Config holds all application configuration
type Config struct {
	Exchanges []ExchangeConfig
	Display   DisplayConfig
	App       AppConfig
}

// ExchangeConfig holds exchange-specific configuration
type ExchangeConfig struct {
	Name   exchange.ExchangeName
	Symbol string
}

// DisplayConfig holds display-related configuration
type DisplayConfig struct {
	Top            int
	UpdateInterval time.Duration
}

// AppConfig holds general application configuration
type AppConfig struct {
	DefaultTickLevel    types.TickLevel
	ReinitCheckInterval time.Duration
	MaxBufferSize       int
	UpdateChannelSize   int
}

// Default returns the default configuration for BTCUSDT on Binance Futures
func Default() Config {
	return Config{
		Exchanges: []ExchangeConfig{
			{
				Name:   exchange.Binancef,
				Symbol: "BTCUSDT",
			},
		},
		Display: DisplayConfig{
			Top:            10,
			UpdateInterval: 2 * time.Second,
		},
		App: AppConfig{
			DefaultTickLevel:    types.Tick1,
			ReinitCheckInterval: 5 * time.Second,
			MaxBufferSize:       100,
			UpdateChannelSize:   1000,
		},
	}
}

// NewBTCUSDT creates a configuration for BTCUSDT trading pair on Binance Futures
func NewBTCUSDT() Config {
	return Default()
}

// NewCustom creates a configuration for a custom trading pair on Binance Futures
func NewCustom(symbol string) Config {
	cfg := Default()
	cfg.Exchanges[0].Symbol = symbol
	return cfg
}

// NewMultiExchange creates a configuration with multiple exchanges
func NewMultiExchange(exchanges []ExchangeConfig) Config {
	cfg := Default()
	cfg.Exchanges = exchanges
	return cfg
}

// SetTickLevel updates the default tick level
func (c *Config) SetTickLevel(tick types.TickLevel) {
	c.App.DefaultTickLevel = tick
}

// SetDisplayTop updates the display top count
func (c *Config) SetDisplayTop(top int) {
	c.Display.Top = top
}

// SetUpdateInterval updates the display update interval
func (c *Config) SetUpdateInterval(interval time.Duration) {
	c.Display.UpdateInterval = interval
}
