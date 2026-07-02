package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"image_sync/config"
	"image_sync/handler"
	"image_sync/worker"
)

func main() {
	configPath := flag.String("config", "./config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Asynq client (for enqueuing tasks)
	client := worker.NewClient(*cfg)
	defer client.Close()

	// Asynq server (for processing tasks)
	srv := worker.NewServer(*cfg)
	mux := worker.NewHandler(*cfg)

	// Start worker in background
	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("asynq worker: %v", err)
		}
	}()

	// HTTP server
	r := gin.Default()
	handler.RegisterRoutes(r, client, *cfg)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
