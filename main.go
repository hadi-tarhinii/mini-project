package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"mini-project/internal/db"
	"mini-project/internal/handler"
	"mini-project/internal/middleware"
	"mini-project/internal/repository"
	"mini-project/internal/service"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  Warning: No .env file found, using system environment variables")
	}

	// 2. Database Connections
	mongoURI := os.Getenv("MONGO_URI")
	dbName := os.Getenv("MONGO_DB_NAME")
	redisAddr := os.Getenv("REDIS_ADDR")

	client, err := db.DBConnect(mongoURI)
	if err != nil {
		log.Fatal("❌ MongoDB Connection Failed:", err)
	}

	// Update your ConnectRedis function to accept the address
	db.ConnectRedis(redisAddr)
	if db.RDB == nil {
		log.Fatal("❌ Redis Connection Failed")
	}

	log.Println("✅ All databases connected successfully")

	// 3. Initialize Layers (Dependency Injection)
	ctx := context.Background()
	userCollection := client.Database(dbName).Collection("users")

	repo := repository.NewUserRepositoryImpl(userCollection, db.RDB)
	userService := service.NewUserService(repo)
	userHandler := handler.NewUserHandler(userService)

	// 4. Background Pub/Sub Listener
	go func() {
		sub := db.RDB.Subscribe(ctx, "transactions")
		defer sub.Close()

		log.Println("📢 Transaction Listener Started...")
		for msg := range sub.Channel() {
			log.Printf("🔔 REAL-TIME EVENT: %s", msg.Payload)
		}
	}()

	// 5. Setup Router & Routes
	router := mux.NewRouter()

	// Register all routes (Login, Transfer, etc.)
	userHandler.RegisterRoutes(router)

	// 6. Apply Global Middleware
	// LoggingMiddleware wraps the entire router
	loggedRouter := middleware.LoggingMiddleware(router)

	// 7. Start the Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      loggedRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("🚀 Server starting on http://localhost:%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed:", err)
	}
}