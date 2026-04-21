package main

import (
	"log"
	"os"

	"eaglepoint/backend/internal/httpserver"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	router := httpserver.NewRouter()
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("api server failed: %v", err)
	}
}
