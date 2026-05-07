# go-elastic-board

![go-elastic-board Screenshot](go-elastic-board.png)

A powerful, real-time Elasticsearch cluster monitoring dashboard written in Go. Features a modern web interface with comprehensive cluster metrics, node monitoring, and cluster settings management.

## Features

### 🔍 **Real-time Cluster Monitoring**

- Live cluster health status with color-coded indicators
- Node statistics with CPU, heap, RAM, and load metrics
- Real-time shard distribution (primary and replica)
- Filesystem usage monitoring with percentage indicators
- Node uptime tracking with color-coded status

### 📊 **Visual Dashboard**

- Interactive charts for CPU, heap, RAM, and load metrics
- Responsive design with automatic table splitting for large screens (2000px+ breakpoint)
- Dark/light theme toggle
- Color-coded node grouping and version differentiation
- Compact table design with optimized column widths

### ⚙️ **Cluster Management**

- View and edit all cluster settings in a comprehensive table
- Quick controls for common settings:
  - Concurrent rebalance allocation
  - Initial primary shard recoveries
  - Rebalance enable/disable dropdown
- Real-time settings updates with validation

### 🔒 **Security Features**

- Optional TLS client certificate authentication
- **Automatic certificate reloading** - certificates are monitored and reloaded without restart
- Configurable allowed CN (Common Name) lists
- CA certificate validation
- Support for both HTTP and HTTPS modes
- Graceful shutdown with proper resource cleanup

### 🖥️ **System Information**

- Elasticsearch version display with unique color coding
- Operating system information with shortened names (e.g., "Oracle Linux Server 9.6" → "OL9.6")
- Node roles with optimized display
- Master node indicators (⭐)

### 🎨 **User Experience**

- Configurable refresh intervals
- Automatic connection status monitoring
- Mobile-responsive design
- Embedded static assets (no external dependencies)
- Favicon support

## Installation

### Prerequisites

- Go 1.24.5 or later
- Access to an Elasticsearch cluster

### Build from Source

```bash
git clone https://github.com/xorpaul/go-elastic-board.git
cd go-elastic-board
go build
```

### Quick Start (HTTP Mode)

```bash
# Run without TLS (development mode)
./go-elastic-board -config example.yaml
```

Then open your browser to `http://localhost:8080`

## Configuration

Create a `config.yaml` file based on the provided `example.yaml`:

```yaml
# Server Configuration
server:
  address: "" # Bind to all interfaces (use "127.0.0.1" for localhost only)
  port: "8080" # Port to listen on

# TLS Configuration (optional)
tls:
  enabled: false # Set to true to enable client certificate authentication
  ca_file: "/path/to/ca.crt"
  cert_file: "/path/to/server.crt"
  key_file: "/path/to/server.key"
  allowed_cns: # List of allowed certificate Common Names
    - "admin"
    - "monitoring-user"
```

### Elasticsearch Backend

By default the dashboard proxies to `http://localhost:9200` with no authentication. To connect to a secured cluster add an `elasticsearch:` block:

```yaml
elasticsearch:
  url: "https://my-es-node:9200"

  # Pick one auth method:
  api_key: "VnVhQ2ZH..."          # "encoded" field from POST /_security/api_key
  # service_account_token: "AAE..." # Bearer token from /_security/service/elastic/...
  # username: "elastic"
  # password: "secret"

  # TLS (optional):
  ca_file: "/etc/pki/ca.pem"      # CA cert to verify the ES server certificate
  insecure_skip_verify: false      # true to skip TLS verification (not recommended)
```

**API key** is the recommended auth method for production. Create one with the `monitor` and `manage` cluster privileges — `monitor` covers all read endpoints the dashboard uses, `manage` is needed for the allocation-exclude button and cluster settings writes:

```
curl -sk -XPOST -u "elastic:PASSWORD" "https://my-es-node:9200/_security/api_key" -H "Content-Type: application/json" -d '{"name":"go-elastic-board","role_descriptors":{"monitor":{"cluster":["monitor","manage"],"indices":[{"names":["*"],"privileges":["monitor"]}]}}}'
```

Use the `encoded` field from the response as the `api_key` value in the config.

### TLS Client Certificate Authentication

For production environments, enable TLS client certificate authentication:

```yaml
tls:
  enabled: true
  ca_file: "/etc/ssl/certs/ca.crt"
  cert_file: "/etc/ssl/certs/server.crt"
  key_file: "/etc/ssl/private/server.key"
  allowed_cns:
    - "elasticsearch-admin"
    - "monitoring-service"
```

### Automatic Certificate Reloading

When TLS is enabled, go-elastic-board automatically monitors certificate files for changes and reloads them without requiring a server restart. This is particularly useful for:

- **Let's Encrypt automatic renewals** - certificates are renewed and reloaded seamlessly
- **Corporate PKI environments** - IT departments can update certificates without service interruption
- **Certificate rotation policies** - supports regular certificate updates as required by security policies

**Features:**

- Monitors server certificate, private key, and CA certificate files
- Watches both files and their parent directories (handles atomic file replacements)
- Debounced reloading (500ms) to handle rapid file system events
- Detailed logging of certificate reload events (enable with `-debug`)
- Graceful error handling - continues with existing certificates if reload fails

**File System Events Handled:**

- Direct file modifications
- Atomic file replacements (common with automated tools)
- Directory-level changes affecting certificate files
- Symbolic link updates

The certificate monitor will log successful reloads and any errors encountered during the process.

## Usage

### Command Line Options

```bash
./go-elastic-board [options]

Options:
  -config string    Path to YAML configuration file
  -debug           Enable debug logging
  -verbose         Enable verbose logging
  -version         Show version and build information
```

### Connecting to Elasticsearch

1. Start the go-elastic-board server
2. Open your web browser to the configured address (default: http://localhost:8080)
3. Enter your Elasticsearch cluster URL (e.g., `http://elasticsearch:9200`)
4. Click "Start" to begin monitoring

### Dashboard Features

- **Refresh Interval**: Adjust the data refresh rate (default: 2 seconds)
- **Theme Toggle**: Switch between light and dark modes
- **Responsive Tables**: Tables automatically split on wide screens for better readability
- **Interactive Settings**: Click on cluster settings to modify values in real-time
- **Visual Indicators**: Color-coded health status, versions, and metrics

## Architecture

- **Frontend**: Modern HTML5/CSS3/JavaScript with Chart.js for visualizations
- **Backend**: Go HTTP server with embedded static assets
- **Security**: Optional TLS with client certificate validation
- **Data Source**: Direct Elasticsearch REST API integration
- **Deployment**: Single binary with all assets embedded

## API Endpoints

The application proxies requests to Elasticsearch through these internal endpoints:

- `/_cluster/health` - Cluster health status
- `/_cat/nodes` - Node information with extended fields
- `/_nodes` - Detailed node statistics
- `/_cluster/settings` - Cluster configuration
- `/_cat/shards` - Shard distribution information

## Browser Compatibility

- Chrome/Chromium 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Dependencies

- Go standard library
- `gopkg.in/yaml.v3` for configuration parsing
- Embedded Chart.js for visualizations
- Embedded Tailwind CSS for styling

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source. Please check the repository for license details.

## Version Information

To check the version and build information:

```bash
./go-elastic-board -version
```

## Troubleshooting

### Common Issues

1. **Connection Failed / all requests 401**: Elasticsearch has security enabled. Add an `elasticsearch:` block with the correct `url` and auth credentials. Check that the `api_key` value is the `encoded` field from the key creation response, not the raw `id` or `api_key` fields.
2. **Browser login popup**: The dashboard is proxying a `WWW-Authenticate` header from Elasticsearch — auth credentials are missing or invalid. Verify the `api_key` value and that the key has not been invalidated.
3. **TLS Certificate Errors**: Verify certificate paths and permissions in the configuration. For a secured ES cluster set `ca_file` to the CA that issued the ES node certificates.
4. **Permission Denied**: Check that the specified ports are available and the user has binding permissions.
5. **Certificate Reload Failures**: Check file permissions and paths. Enable debug mode to see detailed reload logs.
6. **File Watcher Issues**: Ensure the application has read access to certificate directories.

### Debug Mode

Enable debug logging for troubleshooting:

```bash
./go-elastic-board -debug -config config.yaml
```

### Certificate Reloading Verification

To test certificate reloading in debug mode:

```bash
# Start the server with debug logging
./go-elastic-board -debug -config config.yaml

# In another terminal, simulate certificate renewal
cp /path/to/new-server.crt /etc/ssl/certs/server.crt
cp /path/to/new-server.key /etc/ssl/private/server.key

# Check logs for reload confirmation:
# [CertManager] Certificate files changed, reloading...
# [CertManager] Certificates loaded successfully
# [CertManager] Certificates reloaded successfully
```

## Screenshots

The dashboard provides a comprehensive view of your Elasticsearch cluster with:

- Real-time metrics and charts
- Cluster health overview
- Node statistics and performance graphs
- Cluster settings management interface
- Responsive design for all screen sizes

---

**go-elastic-board** - Simple, powerful Elasticsearch cluster monitoring made easy.
