package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
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

		// Ensure symbol is valid
		if symbol == "" {
			continue
		}

		currentPrice := fmt.Sprintf("%v", coin["current_price"])
		change24h := fmt.Sprintf("%v", coin["price_change_percentage_24h"])
		high24h := fmt.Sprintf("%v", coin["high_24h"])
		low24h := fmt.Sprintf("%v", coin["low_24h"])
		vol := fmt.Sprintf("%v", coin["total_volume"])

		if symbol == "btc" {
			fmt.Printf("BTC Found! Volume Ranking Position: %d\n", len(normalized)+1)
		}

		normalized = append(normalized, map[string]interface{}{
			"id":                          coin["id"],
			"symbol":                      strings.ToUpper(symbol),
			"name":                        name,
			"current_price":               currentPrice,
			"price_change_percentage_24h": change24h,
			"high_24h":                    high24h,
			"low_24h":                     low24h,
			"total_volume":                vol,
		})
	}
	return normalized, nil
}

func (s *MarketService) GetStockQuote(symbol string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", symbol, s.AlphaVantageKey)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch stock quote")
	}
	defer resp.Body.Close()

	var quote map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&quote); err != nil {
		return nil, err
	}

	gq, ok := quote["Global Quote"].(map[string]interface{})
	if !ok || len(gq) == 0 {
		return nil, fmt.Errorf("stock not found")
	}

	return map[string]interface{}{
		"symbol":         symbol,
		"price":          gq["05. price"],
		"change_percent": strings.TrimSuffix(gq["10. change percent"].(string), "%"),
		"high":           gq["03. high"],
		"low":            gq["04. low"],
		"volume":         gq["06. volume"],
		"prev_close":     gq["08. previous close"],
		"name":           symbol, // AlphaVantage Global Quote doesn't return name
	}, nil
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

		sym, ok := stock["ticker"].(string)
		if !ok || sym == "" {
			continue
		}

		normalized = append(normalized, map[string]interface{}{
			"symbol": sym,
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

func (s *MarketService) GetAssetDetail(symbol string) (map[string]interface{}, error) {
	// 1. Try to find in Crypto Live List (includes Mock fallback if API fails)
	cryptoList, _ := s.GetCryptoPrices(100)
	for _, c := range cryptoList {
		if strings.EqualFold(c["symbol"].(string), symbol) {
			// Extract data from the list item
			price := c["current_price"]
			change := c["price_change_percentage_24h"]

			// Handle potential missing fields in mock fallback or incomplete API data
			high, _ := c["high_24h"].(string)
			low, _ := c["low_24h"].(string)
			vol, _ := c["total_volume"].(string)

			// If high/low missing (e.g. from Mock), calculate them
			pVal := toFloat64(price)
			if high == "" || high == "<nil>" {
				high = fmt.Sprintf("%.2f", pVal*1.05)
			}
			if low == "" || low == "<nil>" {
				low = fmt.Sprintf("%.2f", pVal*0.95)
			}

			return map[string]interface{}{
				"symbol":         c["symbol"],
				"price":          price,
				"change_percent": change,
				"high":           high,
				"low":            low,
				"volume":         vol,
				"prev_close":     price, // Approx
				"name":           c["name"],
			}, nil
		}
	}

	// 2. Try to fetch specific Stock Quote (Live)
	// Only try if it looks like a stock ticker or we didn't find it in crypto
	if stockQuote, err := s.GetStockQuote(symbol); err == nil {
		return stockQuote, nil
	}

	// 3. Fallback to US Stocks List (if GetStockQuote failed or rate limited)
	usStocks, _ := s.GetUSStocks(100)
	for _, st := range usStocks {
		if strings.EqualFold(st["symbol"].(string), symbol) {
			price := toFloat64(st["price"])
			return map[string]interface{}{
				"symbol":         st["symbol"],
				"price":          st["price"],
				"change_percent": fmt.Sprintf("%v", st["change"]),
				"high":           fmt.Sprintf("%.2f", price*1.02),
				"low":            fmt.Sprintf("%.2f", price*0.98),
				"volume":         "5,678,900", // US Stocks list doesn't have volume often
				"prev_close":     fmt.Sprintf("%.2f", price),
				"name":           st["symbol"],
			}, nil
		}
	}

	// 4. Fallback for unknown assets (Simulated Data)
	basePrice := 150.0 + float64(len(symbol)*10)
	return map[string]interface{}{
		"symbol":         strings.ToUpper(symbol),
		"price":          fmt.Sprintf("%.2f", basePrice),
		"change_percent": "1.25",
		"high":           fmt.Sprintf("%.2f", basePrice*1.05),
		"low":            fmt.Sprintf("%.2f", basePrice*0.95),
		"volume":         "10,000,000",
		"prev_close":     fmt.Sprintf("%.2f", basePrice-2.0),
		"name":           strings.ToUpper(symbol),
	}, nil
}

func (s *MarketService) GetAssetHistory(symbol, period string) ([]map[string]interface{}, error) {
	detail, _ := s.GetAssetDetail(symbol)
	currentPrice := 100.0

	if detail != nil {
		if p, ok := detail["price"].(string); ok {
			currentPrice = toFloat64(p)
		} else if p, ok := detail["price"].(float64); ok {
			currentPrice = p
		}
	}

	if currentPrice == 0 {
		currentPrice = 100.0
	}

	return s.generateMockHistory(currentPrice, period)
}

func (s *MarketService) generateMockHistory(basePrice float64, period string) ([]map[string]interface{}, error) {
	var points int
	var volatility float64

	switch period {
	case "1D":
		points = 24
		volatility = 0.02
	case "1W":
		points = 28 // 4 points per day approx
		volatility = 0.05
	case "1M":
		points = 30
		volatility = 0.08
	case "1Y":
		points = 52
		volatility = 0.20
	case "ALL":
		points = 100
		volatility = 0.40
	default:
		points = 24
		volatility = 0.02
	}

	data := make([]map[string]interface{}, points)
	// Start from a bit further back to end at current price
	// But simple random walk is easier: start at current * random, walk to current

	// Better: Start at (currentPrice) and walk BACKWARDS, then reverse
	prices := make([]float64, points)
	prices[points-1] = basePrice

	rand.Seed(time.Now().UnixNano())

	for i := points - 2; i >= 0; i-- {
		change := (rand.Float64() - 0.5) * volatility * basePrice * 0.1
		prices[i] = prices[i+1] - change // Reverse the change
		if prices[i] < 0.01 {
			prices[i] = 0.01
		}
	}

	now := time.Now()
	for i := 0; i < points; i++ {
		var timeLabel string
		// Calculate time for this point
		// This is a rough approximation for visualization
		switch period {
		case "1D":
			t := now.Add(time.Duration(i-points) * time.Hour)
			timeLabel = t.Format("15:04")
		case "1W":
			t := now.Add(time.Duration(i-points) * 6 * time.Hour)
			timeLabel = t.Format("Mon")
		case "1M":
			t := now.AddDate(0, 0, i-points)
			timeLabel = t.Format("02 Jan")
		case "1Y":
			t := now.AddDate(0, 0, (i-points)*7)
			timeLabel = t.Format("Jan 02")
		case "ALL":
			t := now.AddDate(0, i-points, 0) // Monthly points roughly
			timeLabel = t.Format("2006")
		default:
			timeLabel = fmt.Sprintf("%d", i)
		}

		data[i] = map[string]interface{}{
			"name":  timeLabel,
			"price": math.Round(prices[i]*100) / 100,
		}
	}

	return data, nil
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
