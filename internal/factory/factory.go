package factory

import (
	"fmt"

	"orderbook/internal/exchange"
	"orderbook/internal/exchange/asterdex"
	"orderbook/internal/exchange/binance"
	"orderbook/internal/exchange/bingx"
	"orderbook/internal/exchange/bybit"
	"orderbook/internal/exchange/coinbase"
	"orderbook/internal/exchange/hyperliquid"
	"orderbook/internal/exchange/kraken"
	"orderbook/internal/exchange/okx"
)

// ExchangeConfig holds configuration for creating an exchange
type ExchangeConfig struct {
	Name   exchange.ExchangeName
	Symbol string
}

// NewExchange creates a new exchange instance based on the configuration
func NewExchange(config ExchangeConfig) (exchange.Exchange, error) {
	switch config.Name {
	case exchange.Binancef:
		return binance.NewFuturesExchange(binance.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.Binance:
		return binance.NewSpotExchange(binance.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.Bybitf:
		return bybit.NewFuturesExchange(bybit.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.Bybit:
		return bybit.NewSpotExchange(bybit.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.Kraken:
		return kraken.NewSpotExchange(kraken.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.OKX:
		return okx.NewSpotExchange(okx.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.Coinbase:
		return coinbase.NewSpotExchange(coinbase.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.Asterdexf:
		return asterdex.NewFuturesExchange(asterdex.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.BingX:
		return bingx.NewSpotExchange(bingx.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.BingXf:
		return bingx.NewFuturesExchange(bingx.Config{
			Symbol: config.Symbol,
		}), nil

	case exchange.Hyperliquidf:
		return hyperliquid.NewFuturesExchange(hyperliquid.Config{
			Symbol: config.Symbol,
		}), nil

	default:
		return nil, fmt.Errorf("unknown exchange: %s", config.Name)
	}
}

// ValidateExchangeName checks if the exchange name is supported
func ValidateExchangeName(name string) bool {
	switch exchange.ExchangeName(name) {
	case exchange.Binancef, exchange.Binance, exchange.Bybitf, exchange.Bybit, exchange.Kraken, exchange.Hyperliquidf, exchange.OKX, exchange.Coinbase, exchange.Asterdexf, exchange.BingX, exchange.BingXf:
		return true
	default:
		return false
	}
}

// GetSupportedExchanges returns a list of all supported exchanges
func GetSupportedExchanges() []exchange.ExchangeName {
	return []exchange.ExchangeName{exchange.Binancef, exchange.Binance, exchange.Bybitf, exchange.Bybit, exchange.Kraken, exchange.Hyperliquidf, exchange.OKX, exchange.Coinbase, exchange.Asterdexf, exchange.BingX, exchange.BingXf}
}

// GetImplementedExchanges returns a list of currently implemented exchanges
func GetImplementedExchanges() []exchange.ExchangeName {
	return []exchange.ExchangeName{exchange.Binancef, exchange.Binance, exchange.Bybitf, exchange.Bybit, exchange.Kraken, exchange.Hyperliquidf, exchange.OKX, exchange.Coinbase, exchange.Asterdexf, exchange.BingX, exchange.BingXf}
}
