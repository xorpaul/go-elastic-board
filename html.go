package main

import (
	"fmt"
	"net/http"
)

// dashboardHTML contains the full HTML, CSS, and JavaScript for the monitoring page.
// It is updated to use local static assets instead of CDNs.
const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>go-elastic-board</title>
    <!-- Local static assets -->
    <script src="/static/js/tailwindcss.js"></script>
    <script src="/static/js/chart.min.js"></script>
    <script src="/static/js/chartjs-adapter-date-fns.bundle.min.js"></script>
    <style>
        /* Custom styles for a cleaner look */
        body {
            font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
            transition: background-color 0.3s ease, color 0.3s ease;
        }
        .metric-card {
            transition: all 0.3s ease-in-out;
        }
        .metric-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
        }
        .status-dot {
            height: 1rem;
            width: 1rem;
            border-radius: 50%;
            display: inline-block;
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
        
        /* Dark mode styles */
        .dark {
            background-color: rgb(17, 24, 39);
            color: rgb(243, 244, 246);
        }
        .dark .bg-white {
            background-color: rgb(31, 41, 55) !important;
        }
        .dark .bg-gray-50 {
            background-color: rgb(55, 65, 81) !important;
        }
        .dark .bg-gray-100 {
            background-color: rgb(17, 24, 39) !important;
        }
        .dark .text-gray-900 {
            color: rgb(243, 244, 246) !important;
        }
        .dark .text-gray-500 {
            color: rgb(156, 163, 175) !important;
        }
        .dark .text-gray-600 {
            color: rgb(209, 213, 219) !important;
        }
        .dark .text-gray-700 {
            color: rgb(229, 231, 235) !important;
        }
        .dark .border-gray-200 {
            border-color: rgb(55, 65, 81) !important;
        }
        .dark .divide-gray-200 > :not([hidden]) ~ :not([hidden]) {
            border-color: rgb(55, 65, 81) !important;
        }
        .dark .shadow-md {
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.3), 0 2px 4px -1px rgba(0, 0, 0, 0.2);
        }
    </style>
</head>
<body class="bg-gray-100 text-gray-800 transition-colors duration-300">

    <div class="max-w-none mx-auto p-2 md:p-4">
        <header class="mb-4 flex justify-between items-center">
            <div>
                <h1 class="text-4xl font-bold text-gray-900 dark:text-white">go-elastic-board</h1>
                <p class="text-gray-600 dark:text-gray-300 mt-2">Real-time Elasticsearch cluster monitoring</p>
            </div>
            <div class="flex items-center gap-4">
                <div class="flex items-center gap-2">
                    <label for="refreshInterval" class="text-sm font-medium text-gray-700 dark:text-gray-300">Refresh (s):</label>
                    <input type="number" id="refreshInterval" class="w-20 rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-sm p-1" value="2" min="1">
                </div>
                <button id="themeToggle" class="p-2 rounded-lg bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors">
                    <span id="themeIcon">🌙</span>
                </button>
                <button id="connectBtn" class="bg-indigo-600 dark:bg-indigo-700 text-white font-bold py-2 px-4 rounded-lg hover:bg-indigo-700 dark:hover:bg-indigo-800 transition duration-300">Start</button>
            </div>
        </header>
        
        <div id="connectionStatus" class="mb-4 text-sm"></div>


        <!-- Main Dashboard Grid -->
        <div id="dashboardContent" class="hidden">
            <!-- Cluster Health Summary -->
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4 mb-4">
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md flex items-center justify-between">
                    <div>
                        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Cluster Status</p>
                        <p id="clusterStatus" class="text-2xl font-bold text-gray-900 dark:text-white">-</p>
                    </div>
                    <div id="clusterStatusDot" class="status-dot bg-gray-400"></div>
                </div>
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md">
                    <div class="flex items-center justify-between mb-2">
                        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Total Nodes</p>
                        <p id="nodeCount" class="text-xl font-bold text-gray-900 dark:text-white">-</p>
                    </div>
                    <canvas id="nodeCountChart" width="200" height="60"></canvas>
                </div>
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md">
                    <div class="flex items-center justify-between mb-2">
                        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Total Shards</p>
                        <p id="shardCount" class="text-xl font-bold text-gray-900 dark:text-white">-</p>
                    </div>
                    <canvas id="shardCountChart" width="200" height="60"></canvas>
                </div>
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md">
                    <div class="flex items-center justify-between mb-2">
                        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Unassigned Shards</p>
                        <p id="unassignedShards" class="text-xl font-bold text-red-500 dark:text-red-400">-</p>
                    </div>
                    <canvas id="unassignedShardsChart" width="200" height="60"></canvas>
                </div>
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md">
                    <div class="flex items-center justify-between mb-2">
                        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Relocating Shards</p>
                        <p id="relocatingShards" class="text-xl font-bold text-orange-500 dark:text-orange-400">-</p>
                    </div>
                    <canvas id="relocatingShardsChart" width="200" height="60"></canvas>
                </div>
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md">
                    <div class="flex items-center justify-between mb-2">
                        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Initializing Shards</p>
                        <p id="initializingShards" class="text-xl font-bold text-blue-500 dark:text-blue-400">-</p>
                    </div>
                    <canvas id="initializingShardsChart" width="200" height="60"></canvas>
                </div>
            </div>

            <!-- Charts Grid -->
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md col-span-1">
                    <h3 class="text-lg font-semibold mb-2 text-gray-900 dark:text-white">JVM Heap Usage Over Time (%)</h3>
                    <div style="height: 200px; width: 100%; overflow: hidden;">
                        <canvas id="jvmHeapChart" width="800" height="200" style="height: 200px !important; width: 100% !important; max-height: 200px !important;"></canvas>
                    </div>
                </div>
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md col-span-1">
                    <h3 class="text-lg font-semibold mb-2 text-gray-900 dark:text-white">CPU Load Over Time (1m Avg)</h3>
                    <div style="height: 200px; width: 100%; overflow: hidden;">
                        <canvas id="cpuChart" width="800" height="200" style="height: 200px !important; width: 100% !important; max-height: 200px !important;"></canvas>
                    </div>
                </div>
                <div class="metric-card bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md col-span-1">
                    <h3 class="text-lg font-semibold mb-2 text-gray-900 dark:text-white">Filesystem Usage Over Time (%)</h3>
                    <div style="height: 200px; width: 100%; overflow: hidden;">
                        <canvas id="fsChart" width="800" height="200" style="height: 200px !important; width: 100% !important; max-height: 200px !important;"></canvas>
                    </div>
                </div>
            </div>
            
            <!-- Node List - Split into two columns -->
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <div class="bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md">
                    <h3 class="text-lg font-semibold mb-2 text-gray-900 dark:text-white">Nodes (Part 1)</h3>
                    <div class="overflow-x-auto">
                        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead class="bg-gray-50 dark:bg-gray-700">
                                <tr>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Name</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Role</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">CPU %</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Heap %</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">RAM %</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Load (1m)</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">FS %</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Primary</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Replica</th>
                                </tr>
                            </thead>
                            <tbody id="nodeList1" class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                <!-- Node rows will be inserted here -->
                            </tbody>
                        </table>
                    </div>
                </div>
                <div class="bg-white dark:bg-gray-800 p-4 rounded-xl shadow-md">
                    <h3 class="text-lg font-semibold mb-2 text-gray-900 dark:text-white">Nodes (Part 2)</h3>
                    <div class="overflow-x-auto">
                        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead class="bg-gray-50 dark:bg-gray-700">
                                <tr>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Name</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Role</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">CPU %</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Heap %</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">RAM %</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Load (1m)</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">FS %</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Primary</th>
                                    <th scope="col" class="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Replica</th>
                                </tr>
                            </thead>
                            <tbody id="nodeList2" class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                <!-- Node rows will be inserted here -->
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        // --- Debug Function for Canvas Monitoring ---
        function monitorCanvasChanges() {
            const mainCharts = ['jvmHeapChart', 'cpuChart', 'fsChart'];
            mainCharts.forEach(chartId => {
                const canvas = document.getElementById(chartId);
                if (canvas) {
                    console.log('Initial canvas state for', chartId, ':', {
                        styleHeight: canvas.style.height,
                        styleWidth: canvas.style.width,
                        computedHeight: window.getComputedStyle(canvas).height,
                        computedWidth: window.getComputedStyle(canvas).width,
                        actualWidth: canvas.width,
                        actualHeight: canvas.height
                    });
                    
                    // Create a MutationObserver to watch for style changes
                    const observer = new MutationObserver(function(mutations) {
                        mutations.forEach(function(mutation) {
                            if (mutation.type === 'attributes' && mutation.attributeName === 'style') {
                                console.log('Style changed for', chartId, ':', {
                                    styleHeight: canvas.style.height,
                                    styleWidth: canvas.style.width,
                                    computedHeight: window.getComputedStyle(canvas).height,
                                    computedWidth: window.getComputedStyle(canvas).width
                                });
                            }
                        });
                    });
                    
                    observer.observe(canvas, { attributes: true, attributeFilter: ['style'] });
                }
            });
        }
        
        // --- DOM Elements ---
        const refreshIntervalInput = document.getElementById('refreshInterval');
        const connectBtn = document.getElementById('connectBtn');
        const connectionStatusEl = document.getElementById('connectionStatus');
        const dashboardContentEl = document.getElementById('dashboardContent');
        const themeToggle = document.getElementById('themeToggle');
        const themeIcon = document.getElementById('themeIcon');
        
        // --- Theme Management ---
        function initTheme() {
            // Check for saved theme preference or default to 'dark'
            const savedTheme = localStorage.getItem('theme');
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            const theme = savedTheme || (prefersDark ? 'dark' : 'dark'); // Default to dark
            
            applyTheme(theme);
        }
        
        function applyTheme(theme) {
            if (theme === 'dark') {
                document.documentElement.classList.add('dark');
                document.body.classList.add('dark');
                themeIcon.textContent = '☀️';
                localStorage.setItem('theme', 'dark');
            } else {
                document.documentElement.classList.remove('dark');
                document.body.classList.remove('dark');
                themeIcon.textContent = '🌙';
                localStorage.setItem('theme', 'light');
            }
            updateAllChartColors();
        }
        
        function toggleTheme() {
            const isDark = document.documentElement.classList.contains('dark');
            applyTheme(isDark ? 'light' : 'dark');
        }
        
        function updateAllChartColors() {
            // Update chart colors based on theme
            const isDark = document.documentElement.classList.contains('dark');
            const textColor = isDark ? '#F3F4F6' : '#374151';
            const gridColor = isDark ? '#4B5563' : '#E5E7EB';
            
            Object.values(charts).forEach(chart => {
                if (chart && chart.options) {
                    if (chart.options.scales) {
                        Object.values(chart.options.scales).forEach(scale => {
                            if (scale.ticks) scale.ticks.color = textColor;
                            if (scale.grid) scale.grid.color = gridColor;
                            if (scale.title) scale.title.color = textColor;
                        });
                    }
                    if (chart.options.plugins && chart.options.plugins.legend) {
                        chart.options.plugins.legend.labels = chart.options.plugins.legend.labels || {};
                        chart.options.plugins.legend.labels.color = textColor;
                    }
                    chart.update('none');
                }
            });
        }
        
        // --- Chart instances and data ---
        let charts = {};
        let monitorInterval;
        let jvmHistoryData = {
            labels: [],
            datasets: [{
                label: 'Avg Heap Usage',
                data: [],
                borderColor: 'rgba(75, 192, 192, 1)',
                backgroundColor: 'rgba(75, 192, 192, 0.2)',
                fill: true,
                tension: 0.4
            }]
        };
        
        // CPU Load line chart data
        let cpuHistoryData = {
            labels: [],
            datasets: [{
                label: 'Avg CPU Load',
                data: [],
                borderColor: 'rgba(239, 68, 68, 1)',
                backgroundColor: 'rgba(239, 68, 68, 0.2)',
                fill: true,
                tension: 0.4
            }]
        };
        
        // Filesystem Usage line chart data
        let fsHistoryData = {
            labels: [],
            datasets: [{
                label: 'Filesystem Usage',
                data: [],
                borderColor: 'rgba(34, 197, 94, 1)',
                backgroundColor: 'rgba(34, 197, 94, 0.2)',
                fill: true,
                tension: 0.4
            }]
        };
        
        // Small cluster metric charts data
        let nodeCountData = { labels: [], datasets: [{ data: [], borderColor: 'rgba(34, 197, 94, 1)', backgroundColor: 'rgba(34, 197, 94, 0.1)', fill: true, tension: 0.4 }] };
        let shardCountData = { labels: [], datasets: [{ data: [], borderColor: 'rgba(59, 130, 246, 1)', backgroundColor: 'rgba(59, 130, 246, 0.1)', fill: true, tension: 0.4 }] };
        let unassignedShardsData = { labels: [], datasets: [{ data: [], borderColor: 'rgba(239, 68, 68, 1)', backgroundColor: 'rgba(239, 68, 68, 0.1)', fill: true, tension: 0.4 }] };
        let relocatingShardsData = { labels: [], datasets: [{ data: [], borderColor: 'rgba(249, 115, 22, 1)', backgroundColor: 'rgba(249, 115, 22, 0.1)', fill: true, tension: 0.4 }] };
        let initializingShardsData = { labels: [], datasets: [{ data: [], borderColor: 'rgba(59, 130, 246, 1)', backgroundColor: 'rgba(59, 130, 246, 0.1)', fill: true, tension: 0.4 }] };
        
        // Per-node chart data storage
        let nodeChartsData = {};
        
        const MAX_HISTORY_POINTS = 30; // Number of data points to show in the history chart

        // --- Event Listeners ---
        // Initialize theme on page load
        initTheme();
        
        // Start monitoring canvas changes for debugging
        monitorCanvasChanges();
        
        // Theme toggle
        themeToggle.addEventListener('click', toggleTheme);
        
        // Connect button
        connectBtn.addEventListener('click', () => {
            // Stop any existing monitoring
            if (monitorInterval) {
                clearInterval(monitorInterval);
            }
            // Start new monitoring
            startMonitoring();
        });
        
        // Auto-start monitoring on page load
        setTimeout(() => {
            dashboardContentEl.classList.remove('hidden');
            startMonitoring();
        }, 500);

        /**
         * Initializes and starts the monitoring process.
         */
        function startMonitoring() {
            const refreshIntervalSeconds = parseInt(refreshIntervalInput.value, 10);

            if (isNaN(refreshIntervalSeconds) || refreshIntervalSeconds < 1) {
                updateConnectionStatus('Refresh interval must be at least 1 second.', 'red');
                return;
            }

            // Convert seconds to milliseconds
            const refreshInterval = refreshIntervalSeconds * 1000;

            // Initial fetch to check connection
            fetchAllData();

            // Set up recurring fetch
            monitorInterval = setInterval(() => fetchAllData(), refreshInterval);
        }

        /**
         * Fetches all necessary data from Elasticsearch endpoints.
         */
        async function fetchAllData() {
            try {
                // Perform fetches in parallel for efficiency using the proxy endpoint
                const proxyFetch = (path) => fetch('/proxy', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path })
                });
                
                const [health, nodeStats, catNodes, catShards] = await Promise.all([
                    proxyFetch('/_cluster/health'),
                    proxyFetch('/_nodes/stats/jvm,fs,os,process'),
                    proxyFetch('/_cat/nodes?format=json&h=name,heap.percent,ram.percent,cpu,load_1m,node.role,master'),
                    proxyFetch('/_cat/shards?format=json&h=index,shard,prirep,state,node')
                ]);

                if (!health.ok || !nodeStats.ok || !catNodes.ok || !catShards.ok) {
                    throw new Error('API request failed. Status: Health(' + health.status + '), Stats(' + nodeStats.status + '), Nodes(' + catNodes.status + '), Shards(' + catShards.status + ')');
                }

                const healthData = await health.json();
                const nodeStatsData = await nodeStats.json();
                const catNodesData = await catNodes.json();
                const catShardsData = await catShards.json();
                
                // If successful, show dashboard and update status
                dashboardContentEl.classList.remove('hidden');
                updateConnectionStatus('Successfully connected to Elasticsearch. Last updated: ' + new Date().toLocaleTimeString(), 'green');

                // Update UI with new data
                updateClusterHealth(healthData);
                updateNodeList(catNodesData, catShardsData, nodeStatsData);
                updateAggregateCharts(nodeStatsData);
                updateSmallCharts(healthData);

            } catch (error) {
                console.error('Error fetching Elasticsearch data:', error);
                
                const detailedError = '<strong>Connection Failed.</strong><br>' +
                'Please check that the Elasticsearch Host URL is correct and the server is running.';

                connectionStatusEl.innerHTML = detailedError;
                connectionStatusEl.style.color = 'red';
                
                dashboardContentEl.classList.add('hidden');
                if (monitorInterval) {
                    clearInterval(monitorInterval);
                }
            }
        }

        /**
         * Updates the connection status message.
         * @param {string} message - The message to display.
         * @param {string} color - The color of the message (e.g., 'green', 'red').
         */
        function updateConnectionStatus(message, color) {
            connectionStatusEl.innerHTML = message;
            connectionStatusEl.style.color = color;
        }

        /**
         * Updates the cluster health summary cards.
         * @param {object} data - The data from the /_cluster/health endpoint.
         */
        function updateClusterHealth(data) {
            document.getElementById('clusterStatus').textContent = data.status.charAt(0).toUpperCase() + data.status.slice(1);
            document.getElementById('nodeCount').textContent = data.number_of_nodes;
            document.getElementById('shardCount').textContent = data.active_shards;
            document.getElementById('unassignedShards').textContent = data.unassigned_shards;
            document.getElementById('relocatingShards').textContent = data.relocating_shards || 0;
            document.getElementById('initializingShards').textContent = data.initializing_shards || 0;

            const statusDot = document.getElementById('clusterStatusDot');
            statusDot.classList.remove('bg-green-500', 'bg-yellow-500', 'bg-red-500', 'bg-gray-400');
            if (data.status === 'green') {
                statusDot.classList.add('bg-green-500');
            } else if (data.status === 'yellow') {
                statusDot.classList.add('bg-yellow-500');
            } else {
                statusDot.classList.add('bg-red-500');
            }
        }

        /**
         * Updates the list of nodes in the table.
         * @param {Array} data - The data from the /_cat/nodes endpoint.
         * @param {Array} shardsData - The data from the /_cat/shards endpoint.
         */
        /**
         * Formats a percentage value with consistent padding for alignment
         * @param {string|number} value - The percentage value
         * @param {boolean} isPercentage - Whether to add % suffix
         * @returns {string} Formatted value with padding
         */
        function formatMetricValue(value, isPercentage = true) {
            if (!value || value === '-') return '  -';
            
            const numValue = parseFloat(value);
            if (isNaN(numValue)) return '  -';
            
            if (isPercentage) {
                // Right-align percentage values (pad to 3 characters + %)
                if (numValue >= 100) return numValue.toFixed(0) + '%';
                if (numValue >= 10) return ' ' + numValue.toFixed(0) + '%';
                return '  ' + numValue.toFixed(0) + '%';
            } else {
                // For load values, just ensure consistent formatting
                return numValue.toFixed(2);
            }
        }

        /**
         * Gets the color for filesystem usage percentage
         * @param {number} fsPercent - The filesystem usage percentage
         * @returns {string} The color hex code
         */
        function getFsColor(fsPercent) {
            if (fsPercent >= 90) {
                return '#ef4444'; // red
            } else if (fsPercent >= 85) {
                return '#f97316'; // orange
            } else if (fsPercent >= 45) {
                return '#eab308'; // yellow
            } else {
                return '#22c55e'; // green
            }
        }

        function updateNodeList(data, shardsData, nodeStatsData) {
            const tbody1 = document.getElementById('nodeList1');
            const tbody2 = document.getElementById('nodeList2');
            
            // Create a map of node name to filesystem data
            const fsData = {};
            if (nodeStatsData && nodeStatsData.nodes) {
                for (const nodeId in nodeStatsData.nodes) {
                    const node = nodeStatsData.nodes[nodeId];
                    if (node.name && node.fs && node.fs.total) {
                        const fsTotal = node.fs.total.total_in_bytes;
                        const fsFree = node.fs.total.free_in_bytes;
                        const fsUsedPercent = fsTotal > 0 ? ((fsTotal - fsFree) / fsTotal * 100) : 0;
                        fsData[node.name] = fsUsedPercent;
                    }
                }
            }
            
            // Calculate shard counts per node
            const shardCounts = {};
            if (shardsData) {
                shardsData.forEach(shard => {
                    if (shard.node && shard.state === 'STARTED') {
                        if (!shardCounts[shard.node]) {
                            shardCounts[shard.node] = { primary: 0, replica: 0 };
                        }
                        if (shard.prirep === 'p') {
                            shardCounts[shard.node].primary++;
                        } else if (shard.prirep === 'r') {
                            shardCounts[shard.node].replica++;
                        }
                    }
                });
            }

            // Split data in half
            const midpoint = Math.ceil(data.length / 2);
            const data1 = data.slice(0, midpoint);
            const data2 = data.slice(midpoint);

            // Get current node names from data
            const currentNodes = data.map(node => node.name);
            
            // Get existing rows from both tables
            const existingRows1 = Array.from(tbody1.querySelectorAll('tr'));
            const existingRows2 = Array.from(tbody2.querySelectorAll('tr'));
            const existingNodeNames1 = existingRows1.map(row => {
                const nameCell = row.querySelector('td:first-child');
                return nameCell ? nameCell.textContent.replace(' ⭐', '') : null;
            }).filter(name => name);
            const existingNodeNames2 = existingRows2.map(row => {
                const nameCell = row.querySelector('td:first-child');
                return nameCell ? nameCell.textContent.replace(' ⭐', '') : null;
            }).filter(name => name);

            // Check if we need to rebuild the tables (nodes added/removed or redistribution needed)
            const currentNodes1 = data1.map(node => node.name);
            const currentNodes2 = data2.map(node => node.name);
            
            const table1Changed = currentNodes1.length !== existingNodeNames1.length || 
                                 !currentNodes1.every(name => existingNodeNames1.includes(name));
            const table2Changed = currentNodes2.length !== existingNodeNames2.length || 
                                 !currentNodes2.every(name => existingNodeNames2.includes(name));

            if (table1Changed || table2Changed) {
                // Destroy existing charts before clearing DOM
                Object.keys(charts).forEach(chartId => {
                    if (chartId.includes('Chart_')) {
                        try {
                            charts[chartId].destroy();
                            delete charts[chartId];
                        } catch (error) {
                            console.warn('Error destroying chart:', chartId, error);
                        }
                    }
                });
                
                tbody1.innerHTML = ''; // Clear existing rows
                tbody2.innerHTML = ''; // Clear existing rows
                
                // Rebuild both tables
                if (data1.length > 0) {
                    this.buildNodeTable(data1, shardCounts, tbody1, fsData);
                }
                if (data2.length > 0) {
                    this.buildNodeTable(data2, shardCounts, tbody2, fsData);
                }
            } else {
                // Just update existing rows
                if (data1.length > 0) {
                    this.updateExistingNodeRows(data1, shardCounts, tbody1, fsData);
                }
                if (data2.length > 0) {
                    this.updateExistingNodeRows(data2, shardCounts, tbody2, fsData);
                }
            }
        }

        function buildNodeTable(data, shardCounts, tbody, fsData) {
            data.forEach((node, index) => {
                const isMaster = node.master === '*';
                const nodeShards = shardCounts[node.name] || { primary: 0, replica: 0 };
                const nodeName = node.name;
                
                // Initialize individual metric chart data if not exists
                const metrics = ['cpu', 'heap', 'ram', 'load'];
                metrics.forEach(metric => {
                    const chartKey = nodeName + '_' + metric;
                    if (!nodeChartsData[chartKey]) {
                        let color = 'rgba(34, 197, 94, 1)'; // default green
                        let bgColor = 'rgba(34, 197, 94, 0.1)';
                        
                        if (metric === 'heap') {
                            color = 'rgba(59, 130, 246, 1)'; // blue
                            bgColor = 'rgba(59, 130, 246, 0.1)';
                        } else if (metric === 'ram') {
                            color = 'rgba(239, 68, 68, 1)'; // red
                            bgColor = 'rgba(239, 68, 68, 0.1)';
                        } else if (metric === 'load') {
                            color = 'rgba(168, 85, 247, 1)'; // purple
                            bgColor = 'rgba(168, 85, 247, 0.1)';
                        }
                        
                        nodeChartsData[chartKey] = {
                            labels: [],
                            datasets: [{
                                label: metric.toUpperCase() + ' %',
                                data: [],
                                borderColor: color,
                                backgroundColor: bgColor,
                                fill: true,
                                tension: 0.4
                            }]
                        };
                        
                        // Add initial dummy data point to ensure chart is visible
                        const initialTime = new Date(Date.now() - 1000); // 1 second ago
                        nodeChartsData[chartKey].labels.push(initialTime);
                        nodeChartsData[chartKey].datasets[0].data.push(0);
                    }
                });
                
                this.addCurrentDataToCharts(node, nodeName);
                
                const cpuChartId = 'cpuChart_' + nodeName;
                const heapChartId = 'heapChart_' + nodeName;
                const ramChartId = 'ramChart_' + nodeName;
                const loadChartId = 'loadChart_' + nodeName;
                
                const fsPercent = fsData[nodeName] || 0;
                const fsColor = getFsColor(fsPercent);
                
                const row = '<tr data-node="' + nodeName + '">' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">' + node.name + (isMaster ? ' ⭐' : '') + '</td>' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">' + node['node.role'] + '</td>' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">' +
                            '<div class="flex items-center space-x-2">' +
                                '<span style="font-family: monospace; min-width: 2.5em; text-align: right;">' + formatMetricValue(node.cpu) + '</span>' +
                                '<canvas id="' + cpuChartId + '" width="80" height="30" style="width: 80px; height: 30px; flex-shrink: 0;"></canvas>' +
                            '</div>' +
                        '</td>' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">' +
                            '<div class="flex items-center space-x-2">' +
                                '<span style="font-family: monospace; min-width: 2.5em; text-align: right;">' + formatMetricValue(node['heap.percent']) + '</span>' +
                                '<canvas id="' + heapChartId + '" width="80" height="30" style="width: 80px; height: 30px; flex-shrink: 0;"></canvas>' +
                            '</div>' +
                        '</td>' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">' +
                            '<div class="flex items-center space-x-2">' +
                                '<span style="font-family: monospace; min-width: 2.5em; text-align: right;">' + formatMetricValue(node['ram.percent']) + '</span>' +
                                '<canvas id="' + ramChartId + '" width="80" height="30" style="width: 80px; height: 30px; flex-shrink: 0;"></canvas>' +
                            '</div>' +
                        '</td>' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">' +
                            '<div class="flex items-center space-x-2">' +
                                '<span style="font-family: monospace; min-width: 3em; text-align: right;">' + formatMetricValue(node.load_1m, false) + '</span>' +
                                '<canvas id="' + loadChartId + '" width="80" height="30" style="width: 80px; height: 30px; flex-shrink: 0;"></canvas>' +
                            '</div>' +
                        '</td>' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">' +
                                '<span style="font-family: monospace; min-width: 2.5em; text-align: right; color: ' + fsColor + '; font-weight: bold;">' + formatMetricValue(fsPercent.toFixed(1)) + '</span>' +
                        '</td>' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">' + nodeShards.primary + '</td>' +
                        '<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">' + nodeShards.replica + '</td>' +
                    '</tr>';
                tbody.insertAdjacentHTML('beforeend', row);
                
                // Create charts for this node
                requestAnimationFrame(() => {
                    setTimeout(() => {
                        const cpuValue = parseFloat(node.cpu) || 0;
                        const heapValue = parseFloat(node['heap.percent']) || 0;
                        const ramValue = parseFloat(node['ram.percent']) || 0;
                        const loadValue = parseFloat(node.load_1m) || 0;
                        
                        initializeMetricChart(cpuChartId, nodeChartsData[nodeName + '_cpu'], cpuValue, 50, 'cpu');
                        initializeMetricChart(heapChartId, nodeChartsData[nodeName + '_heap'], heapValue, 65, 'heap');
                        initializeMetricChart(ramChartId, nodeChartsData[nodeName + '_ram'], ramValue, 100, 'ram');
                        initializeMetricChart(loadChartId, nodeChartsData[nodeName + '_load'], loadValue, 100, 'load');
                    }, 50);
                });
            });
        }

        function updateExistingNodeRows(data, shardCounts, tbody, fsData) {
            data.forEach((node, index) => {
                const isMaster = node.master === '*';
                const nodeShards = shardCounts[node.name] || { primary: 0, replica: 0 };
                const nodeName = node.name;
                
                // Find the row for this node
                const row = tbody.querySelector('tr[data-node="' + nodeName + '"]');
                if (!row) return;
                
                // Update text values in the row
                const fsPercent = fsData[nodeName] || 0;
                const fsColor = getFsColor(fsPercent);
                const cells = row.querySelectorAll('td');
                if (cells.length >= 9) {
                    cells[0].innerHTML = node.name + (isMaster ? ' ⭐' : '');
                    cells[1].textContent = node['node.role'];
                    cells[2].querySelector('span').textContent = formatMetricValue(node.cpu);
                    cells[3].querySelector('span').textContent = formatMetricValue(node['heap.percent']);
                    cells[4].querySelector('span').textContent = formatMetricValue(node['ram.percent']);
                    cells[5].querySelector('span').textContent = formatMetricValue(node.load_1m, false);
                    const fsSpan = cells[6].querySelector('span');
                    fsSpan.textContent = formatMetricValue(fsPercent.toFixed(1));
                    fsSpan.style.color = fsColor;
                    fsSpan.style.fontWeight = 'bold';
                    cells[7].textContent = nodeShards.primary;
                    cells[8].textContent = nodeShards.replica;
                }
                
                // Add new data to charts and update them
                this.addCurrentDataToCharts(node, nodeName);
                
                const cpuValue = parseFloat(node.cpu) || 0;
                const heapValue = parseFloat(node['heap.percent']) || 0;
                const ramValue = parseFloat(node['ram.percent']) || 0;
                const loadValue = parseFloat(node.load_1m) || 0;
                
                // Update charts (these should exist already)
                updateMetricChart('cpuChart_' + nodeName, nodeChartsData[nodeName + '_cpu'], cpuValue, 50, 'cpu');
                updateMetricChart('heapChart_' + nodeName, nodeChartsData[nodeName + '_heap'], heapValue, 65, 'heap');
                updateMetricChart('ramChart_' + nodeName, nodeChartsData[nodeName + '_ram'], ramValue, 100, 'ram');
                updateMetricChart('loadChart_' + nodeName, nodeChartsData[nodeName + '_load'], loadValue, 100, 'load');
            });
        }

        function addCurrentDataToCharts(node, nodeName) {
            // Add current data points
            const now = new Date();
            const cpuValue = parseFloat(node.cpu) || 0;
            const heapValue = parseFloat(node['heap.percent']) || 0;
            const ramValue = parseFloat(node['ram.percent']) || 0;
            const loadValue = parseFloat(node.load_1m) || 0;
            
            // Update chart data
            const metrics = [
                { key: nodeName + '_cpu', value: cpuValue },
                { key: nodeName + '_heap', value: heapValue },
                { key: nodeName + '_ram', value: ramValue },
                { key: nodeName + '_load', value: loadValue }
            ];
            
            metrics.forEach(metric => {
                if (nodeChartsData[metric.key]) {
                    nodeChartsData[metric.key].labels.push(now);
                    nodeChartsData[metric.key].datasets[0].data.push(metric.value);
                    
                    // Keep only last MAX_HISTORY_POINTS
                    if (nodeChartsData[metric.key].labels.length > MAX_HISTORY_POINTS) {
                        nodeChartsData[metric.key].labels.shift();
                        nodeChartsData[metric.key].datasets[0].data.shift();
                    }
                }
            });
        }

        /**
         * Calculates aggregate stats and updates the charts.
         * @param {object} data - The data from the /_nodes/stats endpoint.
         */
        function updateAggregateCharts(data) {
            let totalHeapMax = 0;
            let totalHeapUsed = 0;
            let totalFsTotal = 0;
            let totalFsFree = 0;
            let totalCpuPercent = 0;
            let totalLoadAvg = 0;
            let nodeCount = 0;

            for (const nodeId in data.nodes) {
                const node = data.nodes[nodeId];
                nodeCount++;
                
                totalHeapMax += node.jvm.mem.heap_max_in_bytes;
                totalHeapUsed += node.jvm.mem.heap_used_in_bytes;
                
                if(node.fs.total) { // Check if fs data is available
                   totalFsTotal += node.fs.total.total_in_bytes;
                   totalFsFree += node.fs.total.free_in_bytes;
                }
                
                if(node.process.cpu) { // Check if process cpu data is available
                    totalCpuPercent += node.process.cpu.percent;
                }

                if(node.os.cpu && node.os.cpu.load_average && node.os.cpu.load_average['1m']) {
                    totalLoadAvg += node.os.cpu.load_average['1m'];
                }
            }
            
            const avgCpuPercent = nodeCount > 0 ? totalCpuPercent / nodeCount : 0;
            const avgLoad = nodeCount > 0 ? totalLoadAvg / nodeCount : 0;
            const heapUsedPercent = totalHeapMax > 0 ? (totalHeapUsed / totalHeapMax * 100) : 0;
            const fsUsedPercent = totalFsTotal > 0 ? ((totalFsTotal - totalFsFree) / totalFsTotal * 100) : 0;

            // Update all charts as line charts
            updateLineChart('jvmHeapChart', jvmHistoryData, heapUsedPercent);
            updateLineChart('cpuChart', cpuHistoryData, avgLoad);
            updateLineChart('fsChart', fsHistoryData, fsUsedPercent);
        }

        /**
         * Creates or updates a donut chart.
         * @param {string} chartId - The canvas element ID for the chart.
         * @param {string} label - The label for the dataset.
         * @param {number} usedPercent - The percentage value for the 'used' segment.
         */
        function updateDonutChart(chartId, label, usedPercent) {
            const data = {
                labels: [label + ' Used', label + ' Free'],
                datasets: [{
                    data: [usedPercent.toFixed(2), (100 - usedPercent).toFixed(2)],
                    backgroundColor: ['rgba(75, 192, 192, 0.8)', 'rgba(229, 231, 235, 0.8)'],
                    borderColor: ['rgba(75, 192, 192, 1)', 'rgba(209, 213, 219, 1)'],
                    borderWidth: 1
                }]
            };

            const options = {
                responsive: true,
                maintainAspectRatio: true,
                cutout: '70%',
                plugins: {
                    legend: { display: false },
                    tooltip: {
                        callbacks: {
                            label: (context) => context.label + ': ' + context.raw + '%'
                        }
                    }
                }
            };

            if (charts[chartId]) {
                charts[chartId].data.datasets[0].data = data.datasets[0].data;
                charts[chartId].update();
            } else {
                const ctx = document.getElementById(chartId).getContext('2d');
                charts[chartId] = new Chart(ctx, { type: 'doughnut', data, options });
            }
        }
        
        /**
         * Creates or updates the historical line chart.
         * @param {number} currentHeapUsage - The latest heap usage percentage.
         */
        function updateHistoryChart(currentHeapUsage) {
            // Add new data
            jvmHistoryData.labels.push(new Date());
            jvmHistoryData.datasets[0].data.push(currentHeapUsage.toFixed(2));

            // Remove old data if we exceed the max points
            if (jvmHistoryData.labels.length > MAX_HISTORY_POINTS) {
                jvmHistoryData.labels.shift();
                jvmHistoryData.datasets[0].data.shift();
            }

            if (charts.jvmHistoryChart) {
                charts.jvmHistoryChart.update();
            } else {
                const ctx = document.getElementById('jvmHistoryChart').getContext('2d');
                charts.jvmHistoryChart = new Chart(ctx, {
                    type: 'line',
                    data: jvmHistoryData,
                    options: {
                        responsive: true,
                        scales: {
                            x: {
                                type: 'time',
                                time: {
                                    unit: 'second',
                                    displayFormats: {
                                        second: 'HH:mm:ss'
                                    }
                                },
                                title: {
                                    display: true,
                                    text: 'Time'
                                }
                            },
                            y: {
                                beginAtZero: true,
                                max: 100,
                                title: {
                                    display: true,
                                    text: 'Usage (%)'
                                }
                            }
                        },
                        plugins: {
                            legend: { display: false }
                        }
                    }
                });
            }
        }
        
        /**
         * Updates small cluster metric charts.
         * @param {object} healthData - The cluster health data.
         */
        function updateSmallCharts(healthData) {
            const now = new Date();
            
            // Update node count chart
            nodeCountData.labels.push(now);
            nodeCountData.datasets[0].data.push(healthData.number_of_nodes);
            if (nodeCountData.labels.length > MAX_HISTORY_POINTS) {
                nodeCountData.labels.shift();
                nodeCountData.datasets[0].data.shift();
            }
            updateSmallLineChart('nodeCountChart', nodeCountData);
            
            // Update shard count chart
            shardCountData.labels.push(now);
            shardCountData.datasets[0].data.push(healthData.active_shards);
            if (shardCountData.labels.length > MAX_HISTORY_POINTS) {
                shardCountData.labels.shift();
                shardCountData.datasets[0].data.shift();
            }
            updateSmallLineChart('shardCountChart', shardCountData);
            
            // Update unassigned shards chart
            unassignedShardsData.labels.push(now);
            unassignedShardsData.datasets[0].data.push(healthData.unassigned_shards);
            if (unassignedShardsData.labels.length > MAX_HISTORY_POINTS) {
                unassignedShardsData.labels.shift();
                unassignedShardsData.datasets[0].data.shift();
            }
            updateSmallLineChart('unassignedShardsChart', unassignedShardsData);
            
            // Update relocating shards chart
            relocatingShardsData.labels.push(now);
            relocatingShardsData.datasets[0].data.push(healthData.relocating_shards || 0);
            if (relocatingShardsData.labels.length > MAX_HISTORY_POINTS) {
                relocatingShardsData.labels.shift();
                relocatingShardsData.datasets[0].data.shift();
            }
            updateSmallLineChart('relocatingShardsChart', relocatingShardsData);
            
            // Update initializing shards chart
            initializingShardsData.labels.push(now);
            initializingShardsData.datasets[0].data.push(healthData.initializing_shards || 0);
            if (initializingShardsData.labels.length > MAX_HISTORY_POINTS) {
                initializingShardsData.labels.shift();
                initializingShardsData.datasets[0].data.shift();
            }
            updateSmallLineChart('initializingShardsChart', initializingShardsData);
        }
        
        /**
         * Creates or updates a small line chart.
         * @param {string} chartId - The canvas element ID.
         * @param {object} chartData - The chart data.
         */
        function updateSmallLineChart(chartId, chartData) {
            const options = {
                responsive: true,
                maintainAspectRatio: true,
                aspectRatio: 2.5,
                scales: {
                    x: { display: false },
                    y: { display: false }
                },
                plugins: {
                    legend: { display: false },
                    tooltip: { enabled: false }
                },
                elements: {
                    point: { radius: 0 },
                    line: { borderWidth: 2 }
                }
            };

            if (charts[chartId]) {
                charts[chartId].update();
            } else {
                const ctx = document.getElementById(chartId).getContext('2d');
                charts[chartId] = new Chart(ctx, { type: 'line', data: chartData, options });
            }
        }
        
        /**
         * Creates or updates a line chart (for JVM heap).
         * @param {string} chartId - The canvas element ID.
         * @param {object} chartData - The chart data.
         * @param {number} currentValue - The current value to add.
         */
        function updateLineChart(chartId, chartData, currentValue) {
            console.log('updateLineChart called for:', chartId, 'with value:', currentValue);
            
            // Add new data
            chartData.labels.push(new Date());
            chartData.datasets[0].data.push(currentValue.toFixed(2));

            // Remove old data if we exceed the max points
            if (chartData.labels.length > MAX_HISTORY_POINTS) {
                chartData.labels.shift();
                chartData.datasets[0].data.shift();
            }

            if (charts[chartId]) {
                charts[chartId].update();
                console.log('Updated existing chart:', chartId);
                // Log canvas dimensions after update
                const canvas = document.getElementById(chartId);
                if (canvas) {
                    console.log('Canvas dimensions after update:', chartId, '- style height:', canvas.style.height, 'computed height:', window.getComputedStyle(canvas).height);
                }
            } else {
                const ctx = document.getElementById(chartId).getContext('2d');
                console.log('Creating new chart:', chartId);
                console.log('Canvas element before chart creation:', ctx.canvas);
                console.log('Canvas style before chart creation - height:', ctx.canvas.style.height, 'width:', ctx.canvas.style.width);
                console.log('Canvas computed style before:', window.getComputedStyle(ctx.canvas).height, window.getComputedStyle(ctx.canvas).width);
                
                // Force canvas dimensions and prevent Chart.js from changing them
                ctx.canvas.width = 800;
                ctx.canvas.height = 200;
                ctx.canvas.style.height = '200px';
                ctx.canvas.style.width = '100%';
                ctx.canvas.style.maxHeight = '200px';
                
                // Add a protective observer to prevent size changes
                const preventResize = function() {
                    if (ctx.canvas.height !== 200) {
                        console.log('Preventing resize attempt on', chartId, 'from height', ctx.canvas.height, 'to 200');
                        ctx.canvas.height = 200;
                        ctx.canvas.style.height = '200px';
                        ctx.canvas.style.maxHeight = '200px';
                    }
                };
                
                // Call preventResize periodically
                setInterval(preventResize, 100);
                
                // Configure y-axis based on chart type
                let yAxisConfig = {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: 'Usage (%)'
                    }
                };
                
                if (chartId === 'jvmHeapChart' || chartId === 'fsUsageChart') {
                    yAxisConfig.max = 100;
                    yAxisConfig.title.text = 'Usage (%)';
                } else if (chartId === 'cpuLoadChart') {
                    // CPU load can go above 1.0 on multi-core systems
                    yAxisConfig.title.text = 'Load Average';
                    delete yAxisConfig.max; // Let it auto-scale
                }
                
                charts[chartId] = new Chart(ctx, {
                    type: 'line',
                    data: chartData,
                    options: {
                        responsive: false,
                        maintainAspectRatio: false,
                        animation: false,
                        scales: {
                            x: {
                                type: 'time',
                                time: {
                                    unit: 'second',
                                    displayFormats: {
                                        second: 'HH:mm:ss'
                                    }
                                },
                                title: {
                                    display: true,
                                    text: 'Time'
                                }
                            },
                            y: yAxisConfig
                        },
                        plugins: {
                            legend: { display: false }
                        }
                    }
                });
                
                // Force dimensions again after chart creation
                ctx.canvas.width = 800;
                ctx.canvas.height = 200;
                ctx.canvas.style.height = '200px';
                ctx.canvas.style.width = '100%';
                ctx.canvas.style.maxHeight = '200px';
                
                console.log('Chart created:', chartId);
                console.log('Canvas style after chart creation - height:', ctx.canvas.style.height, 'width:', ctx.canvas.style.width);
                console.log('Canvas computed style after:', window.getComputedStyle(ctx.canvas).height, window.getComputedStyle(ctx.canvas).width);
                console.log('Canvas actual dimensions after:', ctx.canvas.width, 'x', ctx.canvas.height);
            }
        }
        
        /**
         * Initializes a small line chart for a specific metric with error checking.
         * @param {string} chartId - The canvas element ID.
         * @param {object} chartData - The chart data.
         * @param {number} currentValue - The current metric value.
         * @param {number} threshold - The threshold for color switching.
         * @param {string} metricType - The type of metric (cpu, heap, ram, load).
         */
        function initializeMetricChart(chartId, chartData, currentValue, threshold, metricType) {
            const ctx = document.getElementById(chartId);
            
            if (!ctx) {
                console.warn('Canvas element not found:', chartId);
                return;
            }
            
            // Skip if chart already exists
            if (charts[chartId]) {
                updateMetricChart(chartId, chartData, currentValue, threshold, metricType);
                return;
            }

            // Update colors based on current values and thresholds
            if (chartData.datasets && chartData.datasets.length > 0) {
                let color, bgColor;
                
                if (metricType === 'cpu' && currentValue > threshold) {
                    color = 'rgba(234, 179, 8, 1)'; // yellow
                    bgColor = 'rgba(234, 179, 8, 0.1)';
                } else if (metricType === 'heap' && currentValue > threshold) {
                    color = 'rgba(234, 179, 8, 1)'; // yellow
                    bgColor = 'rgba(234, 179, 8, 0.1)';
                } else {
                    // Default colors based on metric type
                    if (metricType === 'cpu') {
                        color = 'rgba(34, 197, 94, 1)'; // green
                        bgColor = 'rgba(34, 197, 94, 0.1)';
                    } else if (metricType === 'heap') {
                        color = 'rgba(59, 130, 246, 1)'; // blue
                        bgColor = 'rgba(59, 130, 246, 0.1)';
                    } else if (metricType === 'ram') {
                        color = 'rgba(239, 68, 68, 1)'; // red
                        bgColor = 'rgba(239, 68, 68, 0.1)';
                    } else if (metricType === 'load') {
                        color = 'rgba(168, 85, 247, 1)'; // purple
                        bgColor = 'rgba(168, 85, 247, 0.1)';
                    }
                }
                
                chartData.datasets[0].borderColor = color;
                chartData.datasets[0].backgroundColor = bgColor;
            }

            const options = {
                responsive: false,
                maintainAspectRatio: false,
                animation: false, // Disable animations for faster rendering
                scales: {
                    x: { display: false },
                    y: { 
                        display: false, 
                        beginAtZero: true, 
                        max: metricType === 'load' ? undefined : 100  // Let load auto-scale
                    }
                },
                plugins: {
                    legend: { display: false },
                    tooltip: { enabled: false }
                },
                elements: {
                    point: { radius: 0 },
                    line: { borderWidth: 1 }
                }
            };

            try {
                // Set fixed canvas dimensions
                ctx.width = 80;
                ctx.height = 30;
                ctx.style.width = '80px';
                ctx.style.height = '30px';
                ctx.style.border = '1px solid rgba(200, 200, 200, 0.3)'; // Add subtle border for visibility
                ctx.style.backgroundColor = 'rgba(0, 0, 0, 0.05)'; // Very light background
                
                charts[chartId] = new Chart(ctx.getContext('2d'), { 
                    type: 'line', 
                    data: chartData, 
                    options: options 
                });
            } catch (error) {
                console.error('Error creating chart:', chartId, error);
            }
        }

        /**
         * Updates a small line chart for a specific metric.
         * @param {string} chartId - The canvas element ID.
         * @param {object} chartData - The chart data.
         * @param {number} currentValue - The current metric value.
         * @param {number} threshold - The threshold for color switching.
         * @param {string} metricType - The type of metric (cpu, heap, ram, load).
         */
        function updateMetricChart(chartId, chartData, currentValue, threshold, metricType) {
            if (!charts[chartId]) {
                return;
            }

            // Update colors based on current values and thresholds
            if (chartData.datasets && chartData.datasets.length > 0) {
                let color, bgColor;
                
                if (metricType === 'cpu' && currentValue > threshold) {
                    color = 'rgba(234, 179, 8, 1)'; // yellow
                    bgColor = 'rgba(234, 179, 8, 0.1)';
                } else if (metricType === 'heap' && currentValue > threshold) {
                    color = 'rgba(234, 179, 8, 1)'; // yellow
                    bgColor = 'rgba(234, 179, 8, 0.1)';
                } else {
                    // Default colors based on metric type
                    if (metricType === 'cpu') {
                        color = 'rgba(34, 197, 94, 1)'; // green
                        bgColor = 'rgba(34, 197, 94, 0.1)';
                    } else if (metricType === 'heap') {
                        color = 'rgba(59, 130, 246, 1)'; // blue
                        bgColor = 'rgba(59, 130, 246, 0.1)';
                    } else if (metricType === 'ram') {
                        color = 'rgba(239, 68, 68, 1)'; // red
                        bgColor = 'rgba(239, 68, 68, 0.1)';
                    } else if (metricType === 'load') {
                        color = 'rgba(168, 85, 247, 1)'; // purple
                        bgColor = 'rgba(168, 85, 247, 0.1)';
                    }
                }
                
                chartData.datasets[0].borderColor = color;
                chartData.datasets[0].backgroundColor = bgColor;
            }

            try {
                charts[chartId].update('none'); // Use 'none' mode for immediate update without animation
            } catch (error) {
                console.error('Error updating chart:', chartId, error);
            }
        }
    </script>

</body>
</html>
`

// dashboardHandler serves the HTML page.
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Set the content type to HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Write the HTML content to the response
	fmt.Fprint(w, dashboardHTML)
}
