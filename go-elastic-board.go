// Proxy handler to forward requests to Elasticsearch
package main

import (
	"context"
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
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
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
}

// CertificateManager handles automatic reloading of TLS certificates
type CertificateManager struct {
	certFile    string
	keyFile     string
	caFile      string
	certificate *tls.Certificate
	caCertPool  *x509.CertPool
	mutex       sync.RWMutex
	watcher     *fsnotify.Watcher
	logger      *log.Logger
}

var (
	buildversion string
	buildtime    string
	debug        bool
	verbose      bool
	config       Config
	certManager  *CertificateManager
)

// NewCertificateManager creates a new certificate manager with file watching
func NewCertificateManager(certFile, keyFile, caFile string) (*CertificateManager, error) {
	cm := &CertificateManager{
		certFile: certFile,
		keyFile:  keyFile,
		caFile:   caFile,
		logger:   log.New(os.Stdout, "[CertManager] ", log.LstdFlags),
	}

	// Initial load of certificates
	if err := cm.loadCertificates(); err != nil {
		return nil, fmt.Errorf("failed to load initial certificates: %v", err)
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %v", err)
	}
	cm.watcher = watcher

	// Watch certificate files
	if err := cm.watchFiles(); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch certificate files: %v", err)
	}

	// Start watching for file changes in a goroutine
	go cm.watchForChanges()

	cm.logger.Printf("Certificate manager initialized, watching: %s, %s, %s", certFile, keyFile, caFile)
	return cm, nil
}

// loadCertificates loads the server certificate, key, and CA certificate
func (cm *CertificateManager) loadCertificates() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(cm.certFile, cm.keyFile)
	if err != nil {
		return fmt.Errorf("failed to load server certificate: %v", err)
	}
	cm.certificate = &cert

	// Load CA certificate
	caCert, err := os.ReadFile(cm.caFile)
	if err != nil {
		return fmt.Errorf("failed to read CA file: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate")
	}
	cm.caCertPool = caCertPool

	cm.logger.Printf("Certificates loaded successfully")
	return nil
}

// watchFiles adds certificate files to the file watcher
func (cm *CertificateManager) watchFiles() error {
	files := []string{cm.certFile, cm.keyFile, cm.caFile}

	for _, file := range files {
		// Watch the directory containing the file (some tools replace files atomically)
		dir := filepath.Dir(file)
		if err := cm.watcher.Add(dir); err != nil {
			return fmt.Errorf("failed to watch directory %s: %v", dir, err)
		}

		// Also watch the file itself
		if err := cm.watcher.Add(file); err != nil {
			cm.logger.Printf("Warning: failed to watch file %s directly: %v", file, err)
		}
	}

	return nil
}

// watchForChanges monitors file system events and reloads certificates when needed
func (cm *CertificateManager) watchForChanges() {
	defer cm.watcher.Close()

	// Debounce rapid file changes (common with atomic file replacements)
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		<-debounceTimer.C
	}

	for {
		select {
		case event, ok := <-cm.watcher.Events:
			if !ok {
				return
			}

			// Check if the event affects any of our certificate files
			if cm.isRelevantFile(event.Name) {
				if debug {
					cm.logger.Printf("File system event: %s %s", event.Op.String(), event.Name)
				}

				// Reset the debounce timer
				debounceTimer.Reset(500 * time.Millisecond)
			}

		case err, ok := <-cm.watcher.Errors:
			if !ok {
				return
			}
			cm.logger.Printf("Watcher error: %v", err)

		case <-debounceTimer.C:
			// Reload certificates after debounce period
			cm.logger.Printf("Certificate files changed, reloading...")
			if err := cm.loadCertificates(); err != nil {
				cm.logger.Printf("Failed to reload certificates: %v", err)
			} else {
				cm.logger.Printf("Certificates reloaded successfully")
			}
		}
	}
}

// isRelevantFile checks if a file path is one of our certificate files
func (cm *CertificateManager) isRelevantFile(filePath string) bool {
	files := []string{cm.certFile, cm.keyFile, cm.caFile}
	for _, file := range files {
		if filePath == file || filepath.Base(filePath) == filepath.Base(file) {
			return true
		}
	}
	return false
}

// GetCertificate returns the current server certificate for TLS configuration
func (cm *CertificateManager) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.certificate, nil
}

// GetCACertPool returns the current CA certificate pool
func (cm *CertificateManager) GetCACertPool() *x509.CertPool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.caCertPool
}

// Close stops the file watcher and cleans up resources
func (cm *CertificateManager) Close() error {
	if cm.watcher != nil {
		return cm.watcher.Close()
	}
	return nil
}

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
		log.Printf("Loaded config: Server address=%s, port=%s, TLS enabled=%v, CA file=%s, allowed CNs=%v",
			config.Server.Address, config.Server.Port, config.TLS.Enabled, config.TLS.CAFile, config.TLS.AllowedCNs)
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
	protocol := "http"
	if config.TLS.Enabled {
		protocol = "https"
	}
	fmt.Printf("go-elastic-board server version %s with build time %s starting on %s://%s\n", buildversion, buildtime, protocol, listenAddr)
	fmt.Println("All static assets are embedded. You can now run this binary by itself.")

	if config.TLS.Enabled {
		fmt.Printf("TLS client certificate authentication enabled with CA: %s\n", config.TLS.CAFile)
		fmt.Printf("Allowed client certificate CNs: %v\n", config.TLS.AllowedCNs)

		// Initialize certificate manager with file watching
		var err error
		certManager, err = NewCertificateManager(config.TLS.CertFile, config.TLS.KeyFile, config.TLS.CAFile)
		if err != nil {
			log.Fatalf("Failed to initialize certificate manager: %v", err)
		}

		// Configure TLS with certificate manager
		tlsConfig := &tls.Config{
			ClientAuth:     tls.RequireAndVerifyClientCert,
			ClientCAs:      certManager.GetCACertPool(),
			GetCertificate: certManager.GetCertificate,
			// Enable automatic certificate reloading
			GetClientCertificate: nil,
		}

		server := &http.Server{
			Addr:      listenAddr,
			TLSConfig: tlsConfig,
		}

		fmt.Printf("Certificate monitoring enabled - certificates will be automatically reloaded on file changes\n")

		// Set up graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start server in a goroutine
		serverErr := make(chan error, 1)
		go func() {
			// Use ListenAndServeTLS for TLS-enabled server
			serverErr <- server.ListenAndServeTLS("", "")
		}()

		// Wait for shutdown signal or server error
		select {
		case err := <-serverErr:
			if err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		case sig := <-sigChan:
			log.Printf("Received signal %s, shutting down gracefully...", sig)

			// Shutdown server with timeout
			shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
			defer shutdownCancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("Server shutdown error: %v", err)
			}

			// Clean up certificate manager
			if err := certManager.Close(); err != nil {
				log.Printf("Certificate manager cleanup error: %v", err)
			} else {
				log.Printf("Certificate manager stopped")
			}
		}
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

	esURL := "http://localhost:9200" + reqBody.Path

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
