// Proxy handler to forward requests to Elasticsearch
package main

import (
	"embed" // Import the embed package
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

//go:embed static
var staticDir embed.FS
var (
	buildversion string
	buildtime    string
	debug        bool
	verbose      bool
)

func main() {

	var (
		versionFlag = flag.Bool("version", false, "show build time and version number")
	)
	flag.BoolVar(&debug, "debug", false, "log debug output, defaults to false")
	flag.BoolVar(&verbose, "verbose", false, "log verbose output, defaults to false")
	flag.Parse()

	version := *versionFlag

	if version {
		fmt.Println("go-elastic-board", buildversion, " Build time:", buildtime, "UTC")
		os.Exit(0)
	}

	// Register the handler for the root URL to serve the main HTML page
	http.HandleFunc("/", dashboardHandler)

	// Create a file server from the embedded filesystem.
	fileServer := http.FileServer(http.FS(staticDir))

	// Register the file server to handle all requests that start with "/static/"
	http.Handle("/static/", fileServer)

	// Register the proxy handler for Elasticsearch requests
	http.HandleFunc("/proxy", proxyHandler)

	// Define the port the server will listen on
	port := "8080"

	// Print a message to the console indicating the server is running
	fmt.Printf("go-elastic-board server version %s with build time %s starting on http://localhost:%s\n", buildversion, buildtime, port)
	fmt.Println("All static assets are embedded. You can now run this binary by itself.")

	// Start the HTTP server and log any errors
	log.Fatal(http.ListenAndServe(":"+port, nil))

}

// Proxy handler to forward requests to Elasticsearch
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	esURL := "http://localhost:9200" + reqBody.Path
	esReq, err := http.NewRequest(http.MethodGet, esURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	esRes, err := http.DefaultClient.Do(esReq)
	if err != nil {
		http.Error(w, "Failed to fetch from Elasticsearch: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer esRes.Body.Close()

	for name, values := range esRes.Header {
		w.Header()[name] = values
	}
	w.WriteHeader(esRes.StatusCode)
	io.Copy(w, esRes.Body)
}
