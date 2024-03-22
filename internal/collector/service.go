package collector

import (
	"binance-trend-notifier/pkg"
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type service struct {
	store persistence
	bCli  pkg.BinanceInstance
}

type persistence interface {
	collectorPersistence
}

type collectorPersistence interface {
	insertSymbol(ctx context.Context, s *Symbol) error
	getAllSymbol(ctx context.Context) ([]*Symbol, error)
}

type Service interface {
	InsertSymbol(ctx context.Context, s *Symbol) error
	Start(ctx context.Context) error
}

func (svc *service) Start(ctx context.Context) error {

	syms, err := svc.bCli.GetAllSymbols()
	if err != nil {
		return fmt.Errorf("error getting all symbol: %w", err)
	}
	allSymbols := ExchangeSymbolsToSymbols(syms)

	var wg sync.WaitGroup
	for _, s := range allSymbols {
		wg.Add(1)
		go func(s *Symbol) {
			defer wg.Done()
			err := svc.SymbolRoutine(ctx, s)
			if err != nil {
				return
			}
		}(s)
	}
	wg.Wait()
	return nil
}

func (svc *service) SymbolRoutine(ctx context.Context, s *Symbol) error {
	ec := make(chan error, 1)
	go func(errChan chan error) {
		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()

		if err := svc.performSymbolOperation(ctx, s); err != nil {
			errChan <- err
			return
		}

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("Stopping symbol routine for: %+v\n", s)
				return
			case <-ticker.C:
				if err := svc.performSymbolOperation(ctx, s); err != nil {
					errChan <- err
					return
				}
			}
		}
	}(ec)

	return <-ec
}

func (svc *service) performSymbolOperation(ctx context.Context, s *Symbol) error {
	symbolPrice, err := svc.bCli.GetTickerPrice(s.Symbol)
	if err != nil {
		fmt.Printf("Error getting symbol price for %s: %+v\n", s.Symbol, err)
		return nil
	}

	price, err := strconv.ParseFloat(symbolPrice, 64)
	if err != nil {
		fmt.Printf("Error parsing symbol price for %s: %+v\n", s.Symbol, err)
		return nil
	}

	err = svc.store.insertSymbol(ctx, &Symbol{
		Symbol:    s.Symbol,
		Price:     price,
		Timestamp: time.Now(),
	})
	if err != nil {
		fmt.Printf("Error inserting symbol %s: %+v\n", s.Symbol, err)
	}
	return nil
}

func (svc *service) InsertSymbol(ctx context.Context, s *Symbol) error {
	err := svc.store.insertSymbol(ctx, s)
	if err != nil {
		return err
	}
	return nil

}

func NewService(s persistence, binance pkg.BinanceInstance) (Service, error) {
	return &service{
		store: s,
		bCli:  binance,
	}, nil
}
