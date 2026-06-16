package main

import (
	"bomb-srpg/server"
	"log"
	"net/http"
	"time"
)

func main() {
	r := http.NewServeMux()
	serverState := server.NewServerStateManager()

	fs := http.FileServer(http.Dir("./web/public"))
	r.Handle("GET /", fs)

	// all other HTTP endpoints
	r.HandleFunc("GET /api/archetypes", serverState.HandleGetAllArchetypes)
	r.HandleFunc("POST /api/match-rooms", serverState.HandleCreateNewMatchRoom)

	s := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Println("Bomb Tactics Server running on http://localhost:8080")
	log.Println("Open http://localhost:8080 in your browser to view the Title Screen!")

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server crashed: %v", err)
	}
}
