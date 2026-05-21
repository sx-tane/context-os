package main

import (
	"encoding/json" // used to encode the health response as JSON
	"log"           // used to print startup and error messages
	"net/http"      // provides the HTTP server and request/response types
	"os"            // used to read environment variables
)

// defaultAddr is the port the API binds to when API_ADDR is not set.
const defaultAddr = ":8080"

// main is the entry point — it configures and starts the HTTP server.
func main() {
	addr := os.Getenv("API_ADDR") // read the address from the environment so it can be overridden
	if addr == "" {
		addr = defaultAddr // fall back to :8080 if no environment variable is set
	}

	mux := http.NewServeMux()              // create a new router to register handlers on
	mux.HandleFunc("/health", healthHandler) // register the health check endpoint

	log.Printf("context-os api listening on %s", addr) // log the address so the operator knows it started
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("api server error: %v", err) // exit with error details if the server fails to start
	}
}

// healthHandler responds to GET /health with a JSON payload indicating the service is alive.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json") // tell the client to parse the body as JSON
	json.NewEncoder(w).Encode(map[string]string{        //nolint:errcheck — encode and stream the JSON body directly to the response
		"status":  "ok",              // signals the service is healthy
		"service": "context-os-api", // identifies which service responded
	})
}
