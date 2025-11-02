package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"

	"powerpoint-quiz/internal/config"
	"powerpoint-quiz/internal/db"
	"powerpoint-quiz/internal/handlers"
	"powerpoint-quiz/internal/services"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/quiz.db" // Default path
	}
	if err := db.InitDatabase(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	wsService := services.NewWebSocketService()
	buttonService := services.NewButtonService(db.DB)
	wsService.SetButtonService(buttonService)
	go wsService.Run()

	// Initialize handlers
	wsHandler := handlers.NewWebSocketHandler(wsService)
	staticHandler := handlers.NewStaticHandler()
	buttonHandler := handlers.NewButtonHandler(wsService, buttonService)

	// Setup routes
	router := handlers.SetupRoutes(wsHandler, staticHandler, buttonHandler)

	// Configure server
	server := &http.Server{
		Addr:    cfg.Server.Host + ":" + cfg.Server.Port,
		Handler: router,
	}

	// Configure TLS if enabled
	if cfg.TLS.Enabled {
		server.TLSConfig = &tls.Config{
			MinVersion: getTLSVersion(cfg.TLS.MinVersion),
		}

		log.Printf("Starting HTTPS server on %s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("TLS Certificate: %s", cfg.TLS.CertFile)
		log.Printf("TLS Key: %s", cfg.TLS.KeyFile)
		log.Printf("TLS Min Version: %s", cfg.TLS.MinVersion)

		log.Fatal(server.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile))
	} else {
		log.Printf("Starting HTTP server on %s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Warning: HTTP mode is not recommended for production")

		log.Fatal(server.ListenAndServe())
	}
}

// getTLSVersion converts string version to tls.Version constant
func getTLSVersion(version string) uint16 {
	switch version {
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12
	}
}
