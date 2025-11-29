package main

import (
	"context"
	"embed"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/paulstuart/grpc-example/ux/client"
	"github.com/paulstuart/grpc-example/ux/handlers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func initTracer() *sdktrace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("frontend-service"),
			attribute.String("environment", "development"),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}

// TODO: it would be handy to be able to switch between embedded and filesystem templates based on a flag
// TODO: also perhaps have a watch mode for development that reloads templates on change
// TODO: also gzip the files to keep binary size down

//go:embed templates/*.html
var templateFS embed.FS

func main() {
	// Parse command line flags
	var (
		port     = flag.String("port", "8080", "HTTP server port")
		apiURL   = flag.String("api-url", "https://localhost:11000", "gRPC Gateway API URL")
		jwtToken = flag.String("token", "", "JWT authentication token")
	)
	flag.Parse()

	// Check for JWT token from environment if not provided
	if *jwtToken == "" {
		*jwtToken = os.Getenv("JWT_TOKEN")
	}

	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Set the global propagator for context propagation NOTE: this was copied from AI and needs review
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Create API client
	apiClient := client.NewClient(*apiURL, *jwtToken)

	// Load templates with custom functions
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html"))
	// tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

	// Create handler
	h := handlers.NewHandler(apiClient, tmpl)

	// Setup routes using Go 1.22+ enhanced ServeMux
	mux := http.NewServeMux()

	// Static pages
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"Title": "CrudBox - Home",
		}
		if err := tmpl.ExecuteTemplate(w, "home.html", data); err != nil {
			log.Printf("template error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"Title": "CrudBox - Users",
		}
		if err := tmpl.ExecuteTemplate(w, "users-page.html", data); err != nil {
			log.Printf("template error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	// API endpoints for HTMX
	mux.HandleFunc("GET /users/list", h.ListUsers)
	mux.HandleFunc("GET /users/new", h.NewUserForm)
	mux.HandleFunc("POST /users", h.CreateUser)
	mux.HandleFunc("GET /users/{id}", h.GetUser)
	mux.HandleFunc("GET /users/{id}/edit", h.EditUserForm)
	mux.HandleFunc("PUT /users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /users/{id}", h.DeleteUser)
	mux.HandleFunc("GET /users/filter", h.FilterByRole)

	// Start server
	addr := ":" + *port
	log.Printf("Starting server on %s", addr)
	log.Printf("API URL: %s", *apiURL)
	if *jwtToken != "" {
		log.Printf("Using JWT authentication")
	} else {
		log.Printf("Warning: No JWT token provided, API requests may fail if authentication is required")
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
