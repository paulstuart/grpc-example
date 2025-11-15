package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
)

//go:embed dashboard.html
var content embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Serve the dashboard
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := content.ReadFile("dashboard.html")
		if err != nil {
			http.Error(w, "Dashboard not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy","service":"control-plane"}`)
	})

	log.Printf("Control Plane Dashboard starting on port %s", port)
	log.Printf("Access at: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
