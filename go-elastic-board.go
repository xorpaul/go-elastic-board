// Proxy handler to forward requests to Elasticsearch
package main

import (
	"crypto/tls"
	"crypto/x509"
	"embed" // Import the embed package
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

//go:embed static
var staticDir embed.FS

// TLSConfig holds the TLS configuration for client certificate authentication
type TLSConfig struct {
	Enabled    bool     `yaml:"enabled" default:"true"`
	CAFile     string   `yaml:"ca_file"`
	CertFile   string   `yaml:"cert_file"`
	KeyFile    string   `yaml:"key_file"`
	AllowedCNs []string `yaml:"allowed_cns"`
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	Address string `yaml:"address"`
	Port    string `yaml:"port"`
}

// Config holds the application configuration
type Config struct {
	Server ServerConfig `yaml:"server"`
	TLS    TLSConfig    `yaml:"tls"`

	// Elasticsearch connection settings
	Elasticsearch struct {
		// Base URL to the Elasticsearch HTTP endpoint, e.g. http://localhost:9200
		URL string `yaml:"url"`
	} `yaml:"elasticsearch"`
}

var (
	buildversion string
	buildtime    string
	debug        bool
	verbose      bool
	config       Config
)

// loadConfig loads configuration from YAML file
func loadConfig(configFile string) error {
	if configFile == "" {
		// No config file specified, use defaults (TLS disabled)
		config = Config{
			Server: ServerConfig{
				Address: "",
				Port:    "8080",
			},
			TLS: TLSConfig{
				Enabled: false,
			},
			Elasticsearch: struct {
				URL string `yaml:"url"`
			}{
				URL: "http://localhost:9200",
			},
		}
		return nil
	}

	// Set defaults before loading config file
	config = Config{
		Server: ServerConfig{
			Address: "",
			Port:    "8080",
		},
		TLS: TLSConfig{
			Enabled: true,
		},
		Elasticsearch: struct {
			URL string `yaml:"url"`
		}{
			URL: "http://localhost:9200",
		},
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", configFile, err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config file %s: %v", configFile, err)
	}

	if debug {
		log.Printf("Loaded config: Server address=%s, port=%s, TLS enabled=%v, CA file=%s, allowed CNs=%v, ES URL=%s",
			config.Server.Address, config.Server.Port, config.TLS.Enabled, config.TLS.CAFile, config.TLS.AllowedCNs, config.Elasticsearch.URL)
	}

	return nil
}

// clientCertAuthMiddleware validates client certificates
func clientCertAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !config.TLS.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "Client certificate required", http.StatusUnauthorized)
			return
		}

		clientCert := r.TLS.PeerCertificates[0]
		clientCN := clientCert.Subject.CommonName

		// Check if the CN is in the allowed list
		allowed := false
		for _, allowedCN := range config.TLS.AllowedCNs {
			if clientCN == allowedCN {
				allowed = true
				break
			}
		}

		if !allowed {
			if debug {
				log.Printf("Client certificate CN '%s' not in allowed list: %v", clientCN, config.TLS.AllowedCNs)
			}
			http.Error(w, "Client certificate not authorized", http.StatusForbidden)
			return
		}

		if debug {
			log.Printf("Client authenticated with CN: %s", clientCN)
		}

		next.ServeHTTP(w, r)
	})
}

func main() {

	var (
		versionFlag = flag.Bool("version", false, "show build time and version number")
		configFile  = flag.String("config", "", "path to YAML configuration file")
	)
	flag.BoolVar(&debug, "debug", false, "log debug output, defaults to false")
	flag.BoolVar(&verbose, "verbose", false, "log verbose output, defaults to false")
	flag.Parse()

	version := *versionFlag

	if version {
		fmt.Println("go-elastic-board", buildversion, " Build time:", buildtime, "UTC")
		os.Exit(0)
	}

	// Load configuration
	err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Wrap handlers with client cert auth middleware
	authMiddleware := clientCertAuthMiddleware

	// Register the handler for the root URL to serve the main HTML page
	http.Handle("/", authMiddleware(http.HandlerFunc(dashboardHandler)))

	// Create a file server from the embedded filesystem.
	fileServer := http.FileServer(http.FS(staticDir))

	// Register the file server to handle all requests that start with "/static/"
	http.Handle("/static/", authMiddleware(fileServer))

	// Register the favicon handler
	http.Handle("/favicon.ico", authMiddleware(http.HandlerFunc(faviconHandler)))

	// Register the proxy handler for Elasticsearch requests
	http.Handle("/proxy", authMiddleware(http.HandlerFunc(proxyHandler)))

	// Get server address and port from config
	address := config.Server.Address
	port := config.Server.Port

	// Construct the listen address
	listenAddr := address + ":" + port
	if address == "" {
		listenAddr = "0.0.0.0:" + port
	}

	// Print a message to the console indicating the server is running
	fmt.Printf("go-elastic-board server version %s with build time %s starting on http://%s\n", buildversion, buildtime, listenAddr)
	fmt.Println("All static assets are embedded. You can now run this binary by itself.")
        fmt.Printf("Elasticsearch endpoint: %s\n", config.Elasticsearch.URL)

	if config.TLS.Enabled {
		fmt.Printf("TLS client certificate authentication enabled with CA: %s\n", config.TLS.CAFile)
		fmt.Printf("Allowed client certificate CNs: %v\n", config.TLS.AllowedCNs)

		// Load CA certificate
		caCert, err := os.ReadFile(config.TLS.CAFile)
		if err != nil {
			log.Fatalf("Failed to read CA file: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			log.Fatalf("Failed to parse CA certificate")
		}

		// Configure TLS
		tlsConfig := &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  caCertPool,
		}

		server := &http.Server{
			Addr:      listenAddr,
			TLSConfig: tlsConfig,
		}

		// Start HTTPS server with client certificate authentication
		log.Fatal(server.ListenAndServeTLS(config.TLS.CertFile, config.TLS.KeyFile))
	} else {
		// Start HTTP server without TLS
		log.Fatal(http.ListenAndServe(listenAddr, nil))
	}

}

// Favicon handler to serve favicon.ico from the embedded static folder
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	// Open the favicon file from the embedded static directory
	file, err := staticDir.Open("static/favicon.ico")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// Set the content type for favicon
	w.Header().Set("Content-Type", "image/x-icon")

	// Copy the file content to the response
	io.Copy(w, file)
}

// Proxy handler to forward requests to Elasticsearch
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Path   string `json:"path"`
		Method string `json:"method,omitempty"`
		Body   string `json:"body,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default to GET if no method specified
	method := reqBody.Method
	if method == "" {
		method = http.MethodGet
	}

        esURL := strings.TrimSuffix(config.Elasticsearch.URL, "/") + reqBody.Path

	var esReq *http.Request
	var err error

	// Create request with body if provided
	if reqBody.Body != "" {
		esReq, err = http.NewRequest(method, esURL, strings.NewReader(reqBody.Body))
		if err != nil {
			http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
			return
		}
		esReq.Header.Set("Content-Type", "application/json")
	} else {
		esReq, err = http.NewRequest(method, esURL, nil)
		if err != nil {
			http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	esRes, err := http.DefaultClient.Do(esReq)
	if err != nil {
		http.Error(w, "Failed to fetch from Elasticsearch: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer esRes.Body.Close()

	maps.Copy(w.Header(), esRes.Header)
	w.WriteHeader(esRes.StatusCode)
	io.Copy(w, esRes.Body)
}
