package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MarketService struct {
	BitgetApiKey     string
	BitgetPrivateKey string
	AlphaVantageKey  string
}

func NewMarketService() *MarketService {
	return &MarketService{
		BitgetApiKey:     os.Getenv("BITGET_API_KEY"),
		BitgetPrivateKey: os.Getenv("BITGET_PRIVATE_KEY"),
		AlphaVantageKey:  os.Getenv("ALPHAVANTAGE_API_KEY"),
	}
}

func (s *MarketService) GetCryptoPrices(limit int) ([]map[string]interface{}, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	// CoinGecko Markets API - sorted by 24h gainers
	url := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=price_change_percentage_24h_desc&per_page=50&page=1&sparkline=false&price_change_percentage=24h"
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("CoinGecko API Error (using fallback): %v\n", err)
		return s.GetMockCrypto(limit), nil
	}
	defer resp.Body.Close()

	var data []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return s.GetMockCrypto(limit), nil
	}

	var normalized []map[string]interface{}
	for _, coin := range data {
		if limit > 0 && len(normalized) >= limit {
			break
		}

		symbol, _ := coin["symbol"].(string)
		name, _ := coin["name"].(string)
		currentPrice := fmt.Sprintf("%v", coin["current_price"])
		change24h := fmt.Sprintf("%v", coin["price_change_percentage_24h_in_currency"])

		if symbol == "" {
			continue
		}

		normalized = append(normalized, map[string]interface{}{
			"id":                          coin["id"],
			"symbol":                      strings.ToUpper(symbol),
			"name":                        name,
			"current_price":               currentPrice,
			"price_change_percentage_24h": change24h,
		})
	}
	return normalized, nil
}

func (s *MarketService) GetUSStocks(limit int) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TOP_GAINERS_LOSERS&apikey=%s", s.AlphaVantageKey)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return s.GetMockStocks("us", limit), nil
	}
	defer resp.Body.Close()

	var result struct {
		TopGainers []map[string]interface{} `json:"top_gainers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.GetMockStocks("us", limit), nil
	}

	if len(result.TopGainers) == 0 {
		return s.GetMockStocks("us", limit), nil
	}

	var normalized []map[string]interface{}
	for i, stock := range result.TopGainers {
		if limit > 0 && i >= limit {
			break
		}
		normalized = append(normalized, map[string]interface{}{
			"symbol": stock["ticker"],
			"price":  stock["price"],
			"change": parseFloat(stock["change_percentage"]),
		})
	}
	return normalized, nil
}

func (s *MarketService) GetIndoStocks(limit int) ([]map[string]interface{}, error) {
	// Pre-defined top Indo blue chips
	symbols := []string{"BBCA.JK", "BBRI.JK", "TLKM.JK", "ASII.JK", "BMRI.JK", "BBNI.JK", "GOTO.JK", "BYAN.JK", "AMRT.JK", "UNVR.JK"}
	var results []map[string]interface{}

	for _, sym := range symbols {
		url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", sym, s.AlphaVantageKey)
		resp, err := http.Get(url)
		if err == nil {
			var quote map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&quote) == nil {
				if gq, ok := quote["Global Quote"].(map[string]interface{}); ok && len(gq) > 0 {
					results = append(results, map[string]interface{}{
						"symbol": sym,
						"price":  gq["05. price"],
						"change": parseFloat(gq["10. change percent"]),
					})
				}
			}
			resp.Body.Close()
		}
		if len(results) >= 10 {
			break
		}
	}

	if len(results) == 0 {
		return s.GetMockStocks("indo", limit), nil
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i]["change"].(float64) > results[j]["change"].(float64)
	})

	if limit > 0 && len(results) > limit {
		return results[:limit], nil
	}
	return results, nil
}

func (s *MarketService) GetMockCrypto(limit int) []map[string]interface{} {
	mocks := []map[string]interface{}{
		{"id": "BTCUSDT", "symbol": "BTCUSDT", "name": "Bitcoin", "current_price": "98420.50", "price_change_percentage_24h": "2.45"},
		{"id": "ETHUSDT", "symbol": "ETHUSDT", "name": "Ethereum", "current_price": "2840.12", "price_change_percentage_24h": "-1.20"},
		{"id": "SOLUSDT", "symbol": "SOLUSDT", "name": "Solana", "current_price": "145.67", "price_change_percentage_24h": "5.67"},
		{"id": "BNBUSDT", "symbol": "BNBUSDT", "name": "BNB", "current_price": "612.30", "price_change_percentage_24h": "0.45"},
		{"id": "XRPUSDT", "symbol": "XRPUSDT", "name": "XRP", "current_price": "0.62", "price_change_percentage_24h": "1.12"},
	}
	if limit > 0 && limit < len(mocks) {
		return mocks[:limit]
	}
	return mocks
}

func (s *MarketService) GetMockStocks(region string, limit int) []map[string]interface{} {
	var mocks []map[string]interface{}
	if region == "us" {
		mocks = []map[string]interface{}{
			{"symbol": "NVDA", "price": "134.50", "change": 4.56},
			{"symbol": "TSLA", "price": "240.12", "change": -2.34},
			{"symbol": "AAPL", "price": "220.67", "change": 1.12},
		}
	} else {
		mocks = []map[string]interface{}{
			{"symbol": "BBCA.JK", "price": "10250", "change": 1.23},
			{"symbol": "BBRI.JK", "price": "4560", "change": 0.89},
			{"symbol": "TLKM.JK", "price": "2890", "change": -0.54},
		}
	}
	if limit > 0 && limit < len(mocks) {
		return mocks[:limit]
	}
	return mocks
}

func parseFloat(s interface{}) float64 {
	str, ok := s.(string)
	if !ok {
		return 0
	}
	str = strings.ReplaceAll(str, "%", "")
	f, _ := strconv.ParseFloat(str, 64)
	return f
}
