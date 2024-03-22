package main

import (
	"binance-trend-notifier/internal/collector"
	"binance-trend-notifier/internal/config"
	"context"
	"fmt"
	"time"

	pkg "binance-trend-notifier/pkg"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig(".", "app_config")
	if err != nil {
		fmt.Printf("Error loading config: %+v\n", err)
		panic(err)
	}

	fmt.Printf("Starting server with config: %+v\n", cfg)
	// init mongo client to be used by the service
	mongoCfg := pkg.Config{
		ConnectionURI: cfg.DatabaseURI,
		DatabaseName:  "binance-trend-notifier",
	}
	mi, err := initMongoClient(ctx, mongoCfg)
	if err != nil {
		panic(err)
	}
	fmt.Println("Mongo client connected")

	fmt.Println("Initializing service")

	// init service to be used by the api
	requestTimeLimit, err := time.ParseDuration(cfg.BinanceConfig.RequestTimeLimit)
	if err != nil {
		fmt.Printf("error parsing request time limit: %v\n", err)
		return
	}
	collectService := initService(ctx,
		mi,
		pkg.NewBinanceClient(
			&pkg.BinanceConfig{
				RequestLimit:     cfg.BinanceConfig.RequestLimit,
				RequestTimeLimit: requestTimeLimit,
			}),
		collector.MongoConfig{
			SymbolCollection: cfg.MongoConfig.SymbolsCollection,
			OffersCollection: cfg.MongoConfig.OffersCollection,
		},
	)
	fmt.Println("Service initialized")

	fmt.Println("Starting services...")
	fatalErr := make(chan error, 1)
	startServices(fatalErr, collectService)
	fmt.Println("Services started...")

	<-fatalErr
}

func initMongoClient(ctx context.Context, cfg pkg.Config) (mi *pkg.MongoInstance, err error) {
	mi, err = pkg.NewMongoClient(cfg)
	return mi, err
}

func initService(ctx context.Context, mi *pkg.MongoInstance, bi pkg.BinanceInstance, cfg collector.MongoConfig) (service collector.Service) {
	persistence, err := collector.InitDB(mi, &cfg)
	if err != nil {
		panic(err)
	}
	newService, err := collector.NewService(persistence, bi)
	if err != nil {
		return nil
	}
	return newService
}

func startServices(fatalErr chan error, collectorService collector.Service) {
	go func() {
		ctx := context.Background()
		fatalErr <- collectorService.Start(ctx)
	}()
}
