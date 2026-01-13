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

	url := "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=volume_desc&per_page=50&page=1&sparkline=false&price_change_percentage=24h"
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

	// Robust sort by volume
	sort.Slice(data, func(i, j int) bool {
		vi := toFloat64(data[i]["total_volume"])
		vj := toFloat64(data[j]["total_volume"])
		return vi > vj
	})

	var normalized []map[string]interface{}
	for _, coin := range data {
		symbol, _ := coin["symbol"].(string)

		if symbol == "usdt" || symbol == "usdc" || symbol == "fdusd" || symbol == "dai" {
			continue
		}

		if limit > 0 && len(normalized) >= limit {
			break
		}

		name, _ := coin["name"].(string)
		currentPrice := fmt.Sprintf("%v", coin["current_price"])
		change24h := fmt.Sprintf("%v", coin["price_change_percentage_24h"])

		if symbol == "btc" {
			fmt.Printf("BTC Found! Volume Ranking Position: %d\n", len(normalized)+1)
		}

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
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=MOST_ACTIVELY_TRADED&apikey=%s", s.AlphaVantageKey)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return s.GetMockStocks("us", limit), nil
	}
	defer resp.Body.Close()

	var result struct {
		MostActive []map[string]interface{} `json:"most_actively_traded"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.GetMockStocks("us", limit), nil
	}

	if len(result.MostActive) == 0 {
		return s.GetMockStocks("us", limit), nil
	}

	var normalized []map[string]interface{}
	for i, stock := range result.MostActive {
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
					vol, _ := strconv.ParseFloat(gq["06. volume"].(string), 64)
					results = append(results, map[string]interface{}{
						"symbol": sym,
						"price":  gq["05. price"],
						"change": parseFloat(gq["10. change percent"]),
						"volume": vol,
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

	// Sort by volume
	sort.Slice(results, func(i, j int) bool {
		return results[i]["volume"].(float64) > results[j]["volume"].(float64)
	})

	if limit > 0 && len(results) > limit {
		return results[:limit], nil
	}
	return results, nil
}

func (s *MarketService) GetMockCrypto(limit int) []map[string]interface{} {
	mocks := []map[string]interface{}{
		{"id": "bitcoin", "symbol": "BTC", "name": "Bitcoin", "current_price": "91200", "price_change_percentage_24h": "1.2"},
		{"id": "ethereum", "symbol": "ETH", "name": "Ethereum", "current_price": "2600", "price_change_percentage_24h": "-0.5"},
		{"id": "tether", "symbol": "USDT", "name": "Tether", "current_price": "1.00", "price_change_percentage_24h": "0.01"},
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
			{"symbol": "TSLA", "price": "240.12", "change": -2.34},
			{"symbol": "NVDA", "price": "134.50", "change": 4.56},
			{"symbol": "AAPL", "price": "220.67", "change": 1.12},
		}
	} else {
		mocks = []map[string]interface{}{
			{"symbol": "BBRI.JK", "price": "4560", "change": 0.89},
			{"symbol": "BBCA.JK", "price": "10250", "change": 1.23},
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

func toFloat64(v interface{}) float64 {
	switch i := v.(type) {
	case float64:
		return i
	case float32:
		return float64(i)
	case int64:
		return float64(i)
	case int32:
		return float64(i)
	case int:
		return float64(i)
	case string:
		f, _ := strconv.ParseFloat(i, 64)
		return f
	default:
		return 0
	}
}
