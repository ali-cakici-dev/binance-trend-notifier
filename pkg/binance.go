package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	API_URL      = "https://api.binance.com"
	API_KEY      = ""
	API_SECRET   = ""
	AVG_PRICE    = "/api/v3/avgPrice"
	LATEST_PRICE = "/api/v3/ticker/price"
	EXCHANGE     = "/api/v3/exchangeInfo"
)

var BannedSymbols = []string{
	"AUD",
	"BTC",
	"UP",
	"DOWN",
	"USDC",
	"BULL",
	"BEAR",
	"TUSD",
	"PLN",
	"ZAR",
	"TRY",
	"RUB",
	"NGN",
	"UAH",
	"GBP",
	"EUR",
	"IDRT",
	"TBRL",
	"TARS",
	"DCR",
	"USDS",
	"BIDR",
	"PLN",
	"BKRW",
	"ABC",
	"FDUSD",
	"BUSD",
}

type ExchangeInfoResponse struct {
	Timezone   string   `json:"timezone"`
	ServerTime int64    `json:"serverTime"`
	Symbols    []Symbol `json:"symbols"`
}

type Symbol struct {
	Symbol                          string   `json:"symbol"`
	Status                          string   `json:"status"`
	BaseAsset                       string   `json:"baseAsset"`
	BaseAssetPrecision              int      `json:"baseAssetPrecision"`
	QuoteAsset                      string   `json:"quoteAsset"`
	QuotePrecision                  int      `json:"quotePrecision"`
	QuoteAssetPrecision             int      `json:"quoteAssetPrecision"`
	OrderTypes                      []string `json:"orderTypes"`
	IcebergAllowed                  bool     `json:"icebergAllowed"`
	OcoAllowed                      bool     `json:"ocoAllowed"`
	QuoteOrderQtyMarketAllowed      bool     `json:"quoteOrderQtyMarketAllowed"`
	AllowTrailingStop               bool     `json:"allowTrailingStop"`
	CancelReplaceAllowed            bool     `json:"cancelReplaceAllowed"`
	IsSpotTradingAllowed            bool     `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed          bool     `json:"isMarginTradingAllowed"`
	Permissions                     []string `json:"permissions"`
	DefaultSelfTradePreventionMode  string   `json:"defaultSelfTradePreventionMode"`
	AllowedSelfTradePreventionModes []string `json:"allowedSelfTradePreventionModes"`
}

const (
	GetAllSymbolsWeight   = 20
	GetAveragePriceWeight = 2
	GetTickerPriceWeight  = 2
)

type BinanceInstance interface {
	GetAllSymbols() ([]Symbol, error)
	GetAveragePrice(symbol string) (string, error)
	GetTickerPrice(symbol string) (string, error)
}

// BinanceConfig is the configuration for the BinanceClient
type BinanceConfig struct {
	// How many requests can be made in the time limit
	RequestLimit     int
	RequestTimeLimit time.Duration
}

type BinanceClient struct {
	BinanceInstance
	mutex         sync.Mutex
	cond          *sync.Cond
	requestWeight int
	windowStart   time.Time
	config        *BinanceConfig
}

func (b *BinanceClient) checkRateLimit(requestWeight int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for b.requestWeight >= b.config.RequestLimit && time.Since(b.windowStart) < b.config.RequestTimeLimit {
		b.cond.Wait()
	}

	if time.Since(b.windowStart) >= b.config.RequestTimeLimit {
		b.requestWeight = 0
		b.windowStart = time.Now()
	}

	b.requestWeight += requestWeight
	if b.requestWeight == 1 {
		b.cond.Broadcast()
	}
}

func (b *BinanceClient) GetAllSymbols() ([]Symbol, error) {
	b.checkRateLimit(GetAllSymbolsWeight)

	// get all symbols from binance
	response, err := http.Get(API_URL + EXCHANGE)
	if err != nil {
		return nil, fmt.Errorf("error getting all symbols: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting all symbols: %s", response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response data: %w", err)
	}

	var exchangeInfoResponse ExchangeInfoResponse

	err = json.Unmarshal(body, &exchangeInfoResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response data: %w", err)
	}
	res := make([]Symbol, 0)
	for _, s := range exchangeInfoResponse.Symbols {
		containUSDT := strings.Contains(s.Symbol, "USDT")
		containBanned := false
		for _, banned := range BannedSymbols {
			if strings.Contains(s.Symbol, banned) {
				containBanned = true
				break
			}
		}
		if containUSDT &&
			!containBanned && s.IsSpotTradingAllowed {
			res = append(res, s)
		}
	}

	return res, nil
}

func (b *BinanceClient) GetAveragePrice(symbol string) (string, error) {
	b.checkRateLimit(GetAveragePriceWeight)
	fmt.Println("GetAveragePrice called for symbol: ", symbol)

	// get average price for a symbol
	response, err := http.Get(API_URL + AVG_PRICE + "?symbol=" + symbol)
	if err != nil {
		return "", fmt.Errorf("error getting average price: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusTooManyRequests {
			b.requestWeight += 1000000
			fmt.Println("Too many requests, increasing weight to 1000000")
		}
		return "", fmt.Errorf("error getting average price: %s", response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response data: %w", err)
	}

	var avgPriceResponse struct {
		Minutes   int    `json:"mins"`
		Price     string `json:"price"`
		CloseTime int    `json:"closeTime"`
	}

	err = json.Unmarshal(body, &avgPriceResponse)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response data: %w", err)
	}

	return avgPriceResponse.Price, nil
}

func (b *BinanceClient) GetTickerPrice(symbol string) (string, error) {
	b.checkRateLimit(GetTickerPriceWeight)
	fmt.Println("GetTickerPRice called for symbol: ", symbol)

	// get average price for a symbol
	response, err := http.Get(API_URL + LATEST_PRICE + "?symbol=" + symbol)
	if err != nil {
		return "", fmt.Errorf("error getting ticker price: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusTooManyRequests {
			b.requestWeight += 1000000
			fmt.Println("Too many requests, increasing weight to 1000000")
		}
		return "", fmt.Errorf("error getting ticker price: %s", response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("error reading getting ticker response data: %w", err)
	}

	var tickerPriceResponse struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	err = json.Unmarshal(body, &tickerPriceResponse)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response data: %w", err)
	}

	return tickerPriceResponse.Price, nil
}

func NewBinanceClient(cfg *BinanceConfig) *BinanceClient {
	client := &BinanceClient{
		config:      cfg,
		windowStart: time.Now(),
	}
	client.cond = sync.NewCond(&client.mutex)
	return client
}
