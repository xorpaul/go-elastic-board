package main

import (
	"embed" // Import the embed package
	"fmt"
	"log"
	"net/http"
)

//go:embed static
var staticDir embed.FS

// dashboardHTML contains the full HTML, CSS, and JavaScript for the monitoring page.
// It is updated to use local static assets instead of CDNs.
const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Elasticsearch Real-time Monitoring Dashboard</title>
    <!-- Local static assets -->
    <script src="/static/js/tailwindcss.js"></script>
    <script src="/static/js/chart.min.js"></script>
    <script src="/static/js/chartjs-adapter-date-fns.bundle.min.js"></script>
    <style>
        /* Custom styles for a cleaner look */
        body {
            font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
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
    </style>
</head>
<body class="bg-gray-100 text-gray-800">

    <div class="container mx-auto p-4 md:p-8">
        <header class="mb-8">
            <h1 class="text-4xl font-bold text-gray-900">Elasticsearch Dashboard</h1>
            <p class="text-gray-600 mt-2">A real-time overview of your cluster's health and performance.</p>
        </header>

        <!-- Connection Settings -->
        <div class="bg-white p-6 rounded-xl shadow-md mb-8">
            <h2 class="text-2xl font-semibold mb-4">Connection Settings</h2>
            <div class="flex flex-col sm:flex-row items-center gap-4">
                <div class="w-full sm:w-2/3">
                    <label for="esHost" class="block text-sm font-medium text-gray-700">Elasticsearch Host URL</label>
                    <input type="text" id="esHost" class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2" placeholder="http://localhost:9200" value="http://localhost:9200">
                </div>
                <div class="w-full sm:w-1/3">
                     <label for="refreshInterval" class="block text-sm font-medium text-gray-700">Refresh Interval (ms)</label>
                    <input type="number" id="refreshInterval" class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2" value="5000">
                </div>
                <button id="connectBtn" class="w-full sm:w-auto bg-indigo-600 text-white font-bold py-2 px-4 rounded-lg hover:bg-indigo-700 transition duration-300 self-end">Connect & Monitor</button>
            </div>
             <div id="connectionStatus" class="mt-4 text-sm"></div>
        </div>
        
        <!-- Info Note -->
        <div id="infoNote" class="bg-blue-100 border-l-4 border-blue-500 text-blue-700 p-4 mb-8 rounded-md" role="alert">
          <p class="font-bold">Note</p>
          <p>This application is designed to run on an Elasticsearch node itself, connecting to 'localhost:9200' by default. This avoids CORS issues without exposing your cluster publicly.</p>
        </div>


        <!-- Main Dashboard Grid -->
        <div id="dashboardContent" class="hidden">
            <!-- Cluster Health Summary -->
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                <div class="metric-card bg-white p-6 rounded-xl shadow-md flex items-center justify-between">
                    <div>
                        <p class="text-sm font-medium text-gray-500">Cluster Status</p>
                        <p id="clusterStatus" class="text-2xl font-bold">-</p>
                    </div>
                    <div id="clusterStatusDot" class="status-dot bg-gray-400"></div>
                </div>
                <div class="metric-card bg-white p-6 rounded-xl shadow-md">
                    <p class="text-sm font-medium text-gray-500">Total Nodes</p>
                    <p id="nodeCount" class="text-2xl font-bold">-</p>
                </div>
                <div class="metric-card bg-white p-6 rounded-xl shadow-md">
                    <p class="text-sm font-medium text-gray-500">Total Shards</p>
                    <p id="shardCount" class="text-2xl font-bold">-</p>
                </div>
                <div class="metric-card bg-white p-6 rounded-xl shadow-md">
                    <p class="text-sm font-medium text-gray-500">Unassigned Shards</p>
                    <p id="unassignedShards" class="text-2xl font-bold text-red-500">-</p>
                </div>
            </div>

            <!-- Charts Grid -->
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
                <div class="metric-card bg-white p-6 rounded-xl shadow-md col-span-1">
                    <h3 class="text-lg font-semibold mb-4">JVM Heap Usage</h3>
                    <canvas id="jvmHeapChart"></canvas>
                </div>
                <div class="metric-card bg-white p-6 rounded-xl shadow-md col-span-1">
                    <h3 class="text-lg font-semibold mb-4">CPU Load (1m Avg)</h3>
                    <canvas id="cpuChart"></canvas>
                </div>
                <div class="metric-card bg-white p-6 rounded-xl shadow-md col-span-1">
                    <h3 class="text-lg font-semibold mb-4">Filesystem Usage</h3>
                    <canvas id="fsChart"></canvas>
                </div>
            </div>
            
            <!-- Historical Chart -->
            <div class="metric-card bg-white p-6 rounded-xl shadow-md mb-8">
                <h3 class="text-lg font-semibold mb-4">JVM Heap Usage Over Time (%)</h3>
                 <canvas id="jvmHistoryChart"></canvas>
            </div>

            <!-- Node List -->
            <div class="bg-white p-6 rounded-xl shadow-md">
                <h3 class="text-lg font-semibold mb-4">Nodes</h3>
                <div class="overflow-x-auto">
                    <table class="min-w-full divide-y divide-gray-200">
                        <thead class="bg-gray-50">
                            <tr>
                                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Role</th>
                                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">CPU %</th>
                                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Heap %</th>
                                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">RAM %</th>
                                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Load (1m)</th>
                            </tr>
                        </thead>
                        <tbody id="nodeList" class="bg-white divide-y divide-gray-200">
                            <!-- Node rows will be inserted here -->
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>

    <script>
        // --- DOM Elements ---
        const esHostInput = document.getElementById('esHost');
        const refreshIntervalInput = document.getElementById('refreshInterval');
        const connectBtn = document.getElementById('connectBtn');
        const connectionStatusEl = document.getElementById('connectionStatus');
        const dashboardContentEl = document.getElementById('dashboardContent');
        
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
        const MAX_HISTORY_POINTS = 30; // Number of data points to show in the history chart

        // --- Event Listeners ---
        connectBtn.addEventListener('click', () => {
            // Stop any existing monitoring
            if (monitorInterval) {
                clearInterval(monitorInterval);
            }
            // Start new monitoring
            startMonitoring();
        });

        /**
         * Initializes and starts the monitoring process.
         */
        function startMonitoring() {
            const esHost = esHostInput.value.trim();
            const refreshInterval = parseInt(refreshIntervalInput.value, 10);

            if (!esHost) {
                updateConnectionStatus('Please enter a valid Elasticsearch host URL.', 'red');
                return;
            }
            
            if (isNaN(refreshInterval) || refreshInterval < 1000) {
                updateConnectionStatus('Refresh interval must be at least 1000ms.', 'red');
                return;
            }

            // Initial fetch to check connection
            fetchAllData(esHost);

            // Set up recurring fetch
            monitorInterval = setInterval(() => fetchAllData(esHost), refreshInterval);
        }

        /**
         * Fetches all necessary data from Elasticsearch endpoints.
         * @param {string} host - The Elasticsearch host URL.
         */
        async function fetchAllData(host) {
            try {
                // Perform fetches in parallel for efficiency
                const [health, nodeStats, catNodes] = await Promise.all([
                    fetch(host + '/_cluster/health'),
                    fetch(host + '/_nodes/stats/jvm,fs,os,process'),
                    fetch(host + '/_cat/nodes?format=json&h=name,heap.percent,ram.percent,cpu,load_1m,node.role,master')
                ]);

                if (!health.ok || !nodeStats.ok || !catNodes.ok) {
                    throw new Error('API request failed. Status: Health(' + health.status + '), Stats(' + nodeStats.status + '), Nodes(' + catNodes.status + ')');
                }

                const healthData = await health.json();
                const nodeStatsData = await nodeStats.json();
                const catNodesData = await catNodes.json();
                
                // If successful, show dashboard and update status
                dashboardContentEl.classList.remove('hidden');
                updateConnectionStatus('Successfully connected to ' + host + '. Last updated: ' + new Date().toLocaleTimeString(), 'green');

                // Update UI with new data
                updateClusterHealth(healthData);
                updateNodeList(catNodesData);
                updateAggregateCharts(nodeStatsData);

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
         */
        function updateNodeList(data) {
            const tbody = document.getElementById('nodeList');
            tbody.innerHTML = ''; // Clear existing rows
            data.forEach(node => {
                const isMaster = node.master === '*';
                const row = '<tr>' +
                        '<td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">' + node.name + (isMaster ? ' ⭐' : '') + '</td>' +
                        '<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">' + node['node.role'] + '</td>' +
                        '<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">' + (node.cpu || '-') + '</td>' +
                        '<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">' + node['heap.percent'] + '</td>' +
                        '<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">' + node['ram.percent'] + '</td>' +
                        '<td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">' + (node.load_1m || '-') + '</td>' +
                    '</tr>';
                tbody.insertAdjacentHTML('beforeend', row);
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

            // Update donut charts
            updateDonutChart('jvmHeapChart', 'JVM Heap', heapUsedPercent);
            updateDonutChart('cpuChart', 'Avg CPU', avgCpuPercent);
            updateDonutChart('fsChart', 'Filesystem', fsUsedPercent);
            
            // Update history chart
            updateHistoryChart(heapUsedPercent);
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

func main() {
	// Register the handler for the root URL to serve the main HTML page
	http.HandleFunc("/", dashboardHandler)

	// Create a file server from the embedded filesystem.
	fileServer := http.FileServer(http.FS(staticDir))

	// Register the file server to handle all requests that start with "/static/"
	http.Handle("/static/", fileServer)

	// Define the port the server will listen on
	port := "8080"

	// Print a message to the console indicating the server is running
	fmt.Printf("Elasticsearch Dashboard server starting on http://localhost:%s\n", port)
	fmt.Println("All static assets are embedded. You can now run this binary by itself.")

	// Start the HTTP server and log any errors
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
