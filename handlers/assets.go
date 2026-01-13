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
	// For detail, we reuse the existing logic or add a specific one.
	// For now, let's keep it simple and just return the symbol data.
	// (Actual implementation could fetch specific ticker/quote)
	c.JSON(http.StatusOK, gin.H{"symbol": symbol})
}
