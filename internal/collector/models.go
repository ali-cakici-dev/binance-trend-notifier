package collector

import (
	"binance-trend-notifier/pkg"
	"time"
)

type Symbols []*Symbol
type Symbol struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

func ExchangeSymbolsToSymbols(es []pkg.Symbol) Symbols {
	symbols := make(Symbols, 0, len(es))
	for _, s := range es {
		symbols = append(symbols, ExchangeSymbolToSymbol(s))
	}
	return symbols
}

func ExchangeSymbolToSymbol(es pkg.Symbol) *Symbol {
	return &Symbol{
		Symbol:    es.Symbol,
		Price:     0.0,
		Timestamp: time.Time{},
	}
}
