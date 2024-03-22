package collector

import (
	"binance-trend-notifier/pkg"
	"context"
	"sync"
)

type MongoConfig struct {
	SymbolCollection string
	OffersCollection string
}

type DB struct {
	cli        *pkg.MongoInstance
	binanceCli *pkg.BinanceClient
	cfg        *MongoConfig
	mutex      sync.Mutex
}

func (d *DB) insertSymbol(ctx context.Context, s *Symbol) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	sc := d.cli.DB.Collection(d.cfg.SymbolCollection)
	_, err := sc.InsertOne(ctx, s)
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) getAllSymbol(ctx context.Context) ([]*Symbol, error) {

	panic("implement	me")
}

func InitDB(cli *pkg.MongoInstance, cfg *MongoConfig) (*DB, error) {

	return &DB{
		cli: cli,
		cfg: cfg,
	}, nil
}
