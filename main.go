package main

import (
	"log"
	"net/http"
	"os"

	"realtime_chat/internal/auth"
	"realtime_chat/internal/chat"
	"realtime_chat/internal/db"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://chatuser:chatpass@localhost:5432/chatapp?sslmode=disable"
	}

	jwtSecret := os.Getenv("CHAT_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "yaswanth1801"
	}

	sqlDB, err := db.New(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	if err := db.InitSchema(sqlDB); err != nil {
		log.Fatal(err)
	}

	jwtSvc := auth.NewJWT(jwtSecret)
	authHandlers := auth.NewHandlers(sqlDB, jwtSvc)
	chatRepo := chat.NewRepository(sqlDB)
	hub := chat.NewHub()
	go hub.Run()

	chatHandlers := chat.NewHTTPHandlers(chatRepo, jwtSvc)
	wsHandler := chat.NewWSHandler(hub, chatRepo, jwtSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", authHandlers.Register)
	mux.HandleFunc("/api/login", authHandlers.Login)
	mux.HandleFunc("/api/rooms", chatHandlers.Rooms)
	mux.HandleFunc("/api/messages", chatHandlers.Messages)
	mux.HandleFunc("/ws", wsHandler.Handle)

	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	addr := ":8080"
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
