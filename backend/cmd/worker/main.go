package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/database"
	embpkg "ai-ats-platform/backend/internal/embedding"
	"ai-ats-platform/backend/internal/repository"
	"ai-ats-platform/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// Worker process hosts embedding workers (and future async jobs).
// The API also runs in-process embedding workers for responsiveness; this binary
// is available for dedicated scaling later.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := database.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := database.EnsureSchema(context.Background(), pool); err != nil {
		log.Fatal(err)
	}

	provider, err := embpkg.NewProvider(cfg.Embedding)
	if err != nil {
		log.Fatal(err)
	}
	embeddingRepo := repository.NewEmbeddingRepository(pool)
	embeddingService := service.NewEmbeddingService(embeddingRepo, provider)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	embeddingService.StartWorkers(ctx, cfg.Embedding.Workers)

	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "worker",
			"provider": cfg.Embedding.Provider,
			"model":    cfg.Embedding.Model,
		})
	})

	port := cfg.Port
	if port == "8000" {
		port = "8001"
	}

	srv := &http.Server{Addr: ":" + port, Handler: router}
	go func() {
		log.Printf("worker listening on :%s (embeddings provider=%s)", port, cfg.Embedding.Provider)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
