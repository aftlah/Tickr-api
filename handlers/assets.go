package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/altaf/tickr-backend/internal/cache"
	"github.com/altaf/tickr-backend/services"
	"github.com/gin-gonic/gin"
)

type AssetHandler struct {
	marketService *services.MarketService
	cache         *cache.Cache
}

func NewAssetHandler(ms *services.MarketService, c *cache.Cache) *AssetHandler {
	return &AssetHandler{
		marketService: ms,
		cache:         c,
	}
}

func (h *AssetHandler) GetCrypto(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "0")
	limit, _ := strconv.Atoi(limitStr)

	cacheKey := "crypto_prices_" + limitStr
	if cached, found := h.cache.Get(cacheKey); found {
		c.JSON(http.StatusOK, cached)
		return
	}

	prices, err := h.marketService.GetCryptoPrices(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cacheKey, prices, 5*time.Minute)
	c.JSON(http.StatusOK, prices)
}

func (h *AssetHandler) GetUSStocks(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "0")
	limit, _ := strconv.Atoi(limitStr)

	cacheKey := "us_stocks_" + limitStr
	if cached, found := h.cache.Get(cacheKey); found {
		c.JSON(http.StatusOK, cached)
		return
	}

	stocks, err := h.marketService.GetUSStocks(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cacheKey, stocks, 5*time.Minute)
	c.JSON(http.StatusOK, stocks)
}

func (h *AssetHandler) GetIndoStocks(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "0")
	limit, _ := strconv.Atoi(limitStr)

	cacheKey := "indo_stocks_" + limitStr
	if cached, found := h.cache.Get(cacheKey); found {
		c.JSON(http.StatusOK, cached)
		return
	}

	stocks, err := h.marketService.GetIndoStocks(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cacheKey, stocks, 5*time.Minute)
	c.JSON(http.StatusOK, stocks)
}

func (h *AssetHandler) GetAssetDetail(c *gin.Context) {
	symbol := c.Param("symbol")

	// Check cache
	cacheKey := "asset_detail_" + symbol
	if cached, found := h.cache.Get(cacheKey); found {
		c.JSON(http.StatusOK, cached)
		return
	}

	detail, err := h.marketService.GetAssetDetail(symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cacheKey, detail, 1*time.Minute)
	c.JSON(http.StatusOK, detail)
}

func (h *AssetHandler) GetAssetHistory(c *gin.Context) {
	symbol := c.Param("symbol")
	period := c.DefaultQuery("period", "1D")

	cacheKey := "asset_history_" + symbol + "_" + period
	if cached, found := h.cache.Get(cacheKey); found {
		c.JSON(http.StatusOK, cached)
		return
	}

	history, err := h.marketService.GetAssetHistory(symbol, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.cache.Set(cacheKey, history, 1*time.Minute) // Short cache for "live" feel
	c.JSON(http.StatusOK, history)
}
