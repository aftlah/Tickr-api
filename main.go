package main

import (
	"log"
	"os"

	"github.com/altaf/tickr-backend/handlers"
	"github.com/altaf/tickr-backend/internal/cache"
	"github.com/altaf/tickr-backend/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{os.Getenv("CORS_ALLOWED_ORIGINS")}
	r.Use(cors.New(config))

	// Dependencies
	dataCache := cache.NewCache()
	marketService := services.NewMarketService()
	assetHandler := handlers.NewAssetHandler(marketService, dataCache)

	// Routes
	api := r.Group("/api")
	{
		api.GET("/crypto", assetHandler.GetCrypto)
		api.GET("/stocks/us", assetHandler.GetUSStocks)
		api.GET("/stocks/indo", assetHandler.GetIndoStocks)
		api.GET("/asset/:symbol", assetHandler.GetAssetDetail)
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to run server: ", err)
	}
}
