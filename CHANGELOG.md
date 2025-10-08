# Changelog

All notable changes to this project will be documented in this file.

## [v0.0.3] - 2025-10-08

### 🔧 Bug Fixes & Layout Improvements

- **Fixed Table Width Issue**: Resolved horizontal scrollbar problem in split node tables
  - Reduced column header text length (e.g., "CPU %" → "CPU", "JVM Uptime" → "Uptime")
  - Minimized padding on table columns for better space utilization
  - Made shard movement columns ultra-compact with arrow-only headers (↑, ↓, ↔)
  - Reduced font sizes across table headers for more compact display
  - Enhanced responsive design to prevent horizontal overflow

### � Build & Release Improvements

- **Release Automation**: Added build and release configuration
- **Documentation**: Updated project documentation and screenshots

---

## [v0.0.2] - 2025-10-07

### 🔒 Security & Operations

- **Certificate Auto-Reloading**: Added automatic SSL certificate monitoring and reloading using `fsnotify`
  - Server now automatically detects certificate file changes and reloads without restart
  - Enables seamless certificate renewals for production deployments
  - Graceful shutdown handling with proper signal management

### 📊 Enhanced Monitoring - Shard Movement Tracking

- **New Node Table Columns**: Added three new columns to track Elasticsearch shard movements:

  - **↑Out** (Orange): Shards relocating out from the node
  - **↓In** (Blue): Shards relocating into the node
  - **↔Init** (Amber): Shards initializing on the node

- **Real-time Shard Movement Data**:
  - Integrated `/_cat/shards` API parsing to track `RELOCATING` and `INITIALIZING` shard states
  - Accurate source/destination node mapping for relocating shards
  - Per-node shard movement counters with color-coded indicators

### 🎨 User Interface Improvements

- **Intuitive Column Headers**: Added arrow icons (↑↓↔) for immediate visual understanding of shard movement directions

- **Enhanced Chart Layout**: Reorganized dashboard charts for better space utilization:

  - Prioritized shard-related charts (Total, Unassigned, Initializing, Relocating)
  - Compacted system metrics charts (JVM, CPU, FS) on same row
  - Added ETA calculations for shard recovery operations

- **Color-Coded Movement Indicators**: Consistent color scheme across shard movement data
  - Orange for outgoing operations
  - Blue for incoming operations
  - Amber for initialization processes

### 🔧 Technical Improvements

- **Async Data Fetching**: Optimized parallel API calls for shard movement data
- **Improved Error Handling**: Enhanced error reporting for shard movement API failures
- **Performance Optimization**: Reduced redundant API calls and improved data processing efficiency

### 🚀 Operational Benefits

- **Production Ready**: Certificate auto-reloading eliminates manual intervention for SSL renewals
- **Enhanced Cluster Visibility**: Real-time shard movement tracking provides critical operational insights
- **Improved Troubleshooting**: Detailed per-node shard activity helps identify cluster rebalancing issues
- **Better Capacity Planning**: Shard movement metrics aid in cluster scaling decisions

### 🛠️ Dependencies

- Added `github.com/fsnotify/fsnotify` for file system monitoring

---

## [v0.0.1] - Initial Release

- Basic Elasticsearch cluster monitoring dashboard
- Node health and metrics visualization
- Cluster health monitoring
- Basic shard information display
