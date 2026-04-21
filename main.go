package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"mini-project/internal/db"
	"mini-project/internal/handler"
	"mini-project/internal/middleware"
	"mini-project/internal/repository"
	"mini-project/internal/service"

	"github.com/gorilla/mux"
)

func main() {
	// 1. Database Connections
	// Ensure your MongoDB URI supports transactions (direct connection or replica set)
	mongoURI := "mongodb://localhost:27017/?connect=direct"
	client, err := db.DBConnect(mongoURI)
	if err != nil {
		log.Fatal("❌ MongoDB Connection Failed:", err)
	}

	// Connect to Redis
	db.ConnectRedis()
	if db.RDB == nil {
		log.Fatal("❌ Redis Connection Failed")
	}

	log.Println("✅ All databases connected successfully")

	// 2. Initialize Layers (Dependency Injection)
	ctx := context.Background()
	userCollection := client.Database("mini_project_db").Collection("users")

	// Initialize the Composite Repository (Handles both Mongo and Redis internally)
	repo := repository.NewUserRepositoryImpl(userCollection, db.RDB)

	// Initialize Service Layer
	userService := service.NewUserService(repo)

	// Initialize Handler Layer
	userHandler := handler.NewUserHandler(userService)

	// 3. Background Pub/Sub Listener
	// This listens for transaction messages published by your repo/service
	go func() {
		sub := db.RDB.Subscribe(ctx, "transactions")
		defer sub.Close()

		log.Println("📢 Transaction Listener Started...")
		for msg := range sub.Channel() {
			log.Printf("🔔 REAL-TIME EVENT: %s", msg.Payload)
		}
	}()

	// 4. Setup Router & Routes
	router := mux.NewRouter()

	// Register User Routes
	userHandler.RegisterRoutes(router)

	// 5. Apply Global Middleware
	// Wrap the router with the LoggingMiddleware so every API call is timed and logged
	loggedRouter := middleware.LoggingMiddleware(router)

	// 6. Start the Server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      loggedRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("🚀 Server starting on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed:", err)
	}
}
