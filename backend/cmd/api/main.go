
package main

import (
	"context"
	"log"
	"net/http"

	"ai-ats-platform/backend/internal/database"

	"github.com/gin-gonic/gin"
)

func main() {
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.Background())

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "AI ATS Backend is running 🚀",
		})
	})

	log.Println("🚀 Server running on http://localhost:8000")
	router.Run(":8000")
}

