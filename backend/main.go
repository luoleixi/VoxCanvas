package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"voxcanvas/backend/internal/config"
	"voxcanvas/backend/internal/db"
	"voxcanvas/backend/internal/llm"
	"voxcanvas/backend/internal/router"
	"voxcanvas/backend/internal/service"
)

func main() {
	cfg := config.Load()

	database, err := db.Open("data")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	var (
		classifier llm.Classifier
		refiner    llm.Refiner
		generator  llm.Generator
	)

	if cfg.LLMMode == "real" {
		client := llm.NewRealClient(cfg.LLMAPIURL, cfg.LLMAPIKey, cfg.LLMModel)
		classifier = client
		refiner = client
		generator = &llm.RealGenerator{}
	} else {
		classifier = &llm.MockClassifier{}
		refiner = &llm.MockRefiner{}
		generator = &llm.MockGenerator{}
	}

	devStore := service.NewDevStore()
	drawSvc := &service.DrawService{
		Dev:        devStore,
		Classifier: classifier,
		Refiner:    refiner,
		Generator:  generator,
		DB:         database,
	}

	r := router.Setup(drawSvc)

	addr := ":" + envOrDefault("PORT", "6060")
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("VoxCanvas backend listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exited")
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
