package main

import (
	"encoding/json"

	"log"
	"net/http"
	"net/url"

	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type VisitedURLs struct {
	urls map[string]bool
	mux  sync.Mutex
}

type CrawlResult struct {
	URL      string    `json:"url"`
	WorkerID int       `json:"worker_id"`
	Time     time.Time `json:"time"`
	Status   string    `json:"status"`
}

var (
	rateLimiter = time.Tick(500 * time.Millisecond)
	results     = struct {
		sync.Mutex
		Data []CrawlResult
	}{Data: make([]CrawlResult, 0)}
	urlsChan   = make(chan string, 100) // Buffered channel
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.Mutex
	upgrader   = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true }, // Allow all origins
	}
	stopped    bool
	stoppedMux sync.Mutex
	crawlWG    sync.WaitGroup
	//robotsMap  = make(map[string]*robotstxt.RobotsData)
	//robotsMutex sync.Mutex
)

func main() {
	log.Println("Starting Concurrent Web Crawler with Frontend")

	// Start HTTP server
	go func() {
		http.HandleFunc("/", serveFrontend)
		http.HandleFunc("/ws", handleWebSocket)
		http.HandleFunc("/stop", handleStop)
		http.HandleFunc("/start", handleStart)
		log.Println("Starting HTTP server on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Keep main thread alive
	select {}
}

func (v *VisitedURLs) IsVisited(url string) bool {
	v.mux.Lock()
	defer v.mux.Unlock()
	return v.urls[url]
}

func (v *VisitedURLs) Add(url string) {
	v.mux.Lock()
	defer v.mux.Unlock()
	v.urls[url] = true
}

func worker(id int, wg *sync.WaitGroup, visited *VisitedURLs, baseURL string) {
	defer wg.Done()
	for url := range urlsChan {
		<-rateLimiter
		stoppedMux.Lock()
		if stopped {
			stoppedMux.Unlock()
			continue
		}
		stoppedMux.Unlock()

		if visited.IsVisited(url) {
			continue
		}

		visited.Add(url)
		log.Printf("Worker %d fetching URL: %s\n", id, url)

		status := "Success"
		if !canFetch(url) {
			status = "Blocked by robots.txt"
			broadcastResult(CrawlResult{URL: url, WorkerID: id, Time: time.Now(), Status: status})
			continue
		}

		content, err := fetch(url)
		if err != nil {
			status = "Fetch error: " + err.Error()
			log.Println("Error fetching page:", err)
			broadcastResult(CrawlResult{URL: url, WorkerID: id, Time: time.Now(), Status: status})
			continue
		}

		broadcastResult(CrawlResult{URL: url, WorkerID: id, Time: time.Now(), Status: status})

		links, err := parseLinks(content)
		if err != nil {
			log.Println("Error parsing page:", err)
			continue
		}
		for _, link := range links {
			absoluteURL := normalizeURL(link, baseURL)
			urlsChan <- absoluteURL
		}
	}
}

func broadcastResult(result CrawlResult) {
	results.Lock()
	results.Data = append(results.Data, result)
	results.Unlock()

	clientsMux.Lock()
	defer clientsMux.Unlock()
	data, _ := json.Marshal(result)
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("WebSocket write error:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func serveFrontend(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Web Crawler</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 p-6">
	<div class="max-w-4xl mx-auto bg-white p-6 rounded-lg shadow-lg">
		<h1 class="text-2xl font-bold text-center mb-4">Web Crawler</h1>
		
		<div class="mb-6">
			<form id="startForm" class="flex flex-col sm:flex-row gap-4">
				<input 
					type="url" 
					id="startUrl" 
					placeholder="Enter URL to crawl (e.g. https://example.com)" 
					required
					class="flex-grow border rounded px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
				>
				<div class="flex gap-2">
					<button 
						type="submit" 
						class="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 flex-grow"
					>
						Start Crawler
					</button>
					<select 
						id="numWorkers" 
						class="border rounded px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
					>
						<option value="5">5 Workers</option>
						<option value="10" selected>10 Workers</option>
						<option value="15">15 Workers</option>
						<option value="20">20 Workers</option>
					</select>
				</div>
			</form>
		</div>
		
		<div class="flex justify-between mb-4">
			<button onclick="stopCrawler()" class="bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600">Stop Crawler</button>
			<button onclick="clearResults()" class="bg-gray-500 text-white px-4 py-2 rounded hover:bg-gray-600">Clear Results</button>
		</div>
		<div id="progress" class="text-blue-600 font-semibold text-center mb-4">Ready to crawl</div>
		<div class="overflow-x-auto">
			<table class="min-w-full bg-white border rounded-lg shadow-md">
				<thead>
					<tr class="bg-gray-200 text-gray-700">
						<th class="py-2 px-4 border">URL</th>
						<th class="py-2 px-4 border">Worker ID</th>
						<th class="py-2 px-4 border">Time</th>
						<th class="py-2 px-4 border">Status</th>
					</tr>
				</thead>
				<tbody id="results"></tbody>
			</table>
		</div>
	</div>
	<script>
		const socket = new WebSocket("wss://concurrent-web-crawler-production.up.railway.app/ws");
		const tbody = document.querySelector("#results");
		const progress = document.getElementById("progress");
		const startForm = document.getElementById("startForm");

		ws.onmessage = function(event) {
			progress.textContent = "Crawling in progress...";
			progress.classList.remove("text-red-600");
			progress.classList.add("text-blue-600");
			const result = JSON.parse(event.data);
			const row = document.createElement("tr");
			row.className = "border-b text-center";
			row.innerHTML = "<td class='py-2 px-4 border'>" + result.url + "</td>" +
							"<td class='py-2 px-4 border'>" + result.worker_id + "</td>" +
							"<td class='py-2 px-4 border'>" + new Date(result.time).toLocaleString() + "</td>" +
							"<td class='py-2 px-4 border'>" + result.status + "</td>";
			tbody.appendChild(row);
		};

		ws.onclose = function() {
			progress.textContent = "WebSocket connection closed.";
			progress.classList.remove("text-blue-600");
			progress.classList.add("text-red-600");
		};

		ws.onerror = function(error) {
			console.error("WebSocket error:", error);
			progress.textContent = "Error connecting to WebSocket.";
			progress.classList.remove("text-blue-600");
			progress.classList.add("text-red-600");
		};

		startForm.addEventListener("submit", function(e) {
			e.preventDefault();
			const startUrl = document.getElementById("startUrl").value;
			const numWorkers = document.getElementById("numWorkers").value;
			
			if (!startUrl) {
				alert("Please enter a URL to crawl");
				return;
			}
			
			progress.textContent = "Starting crawler...";
			progress.classList.remove("text-red-600");
			progress.classList.add("text-blue-600");
			
			fetch("/start", {
				method: "POST",
				headers: {
					"Content-Type": "application/json"
				},
				body: JSON.stringify({ url: startUrl, workers: numWorkers })
			})
			.then(response => response.text())
			.then(data => {
				console.log("Crawler started:", data);
			})
			.catch(err => {
				console.error("Error starting crawler:", err);
				progress.textContent = "Error starting crawler.";
				progress.classList.remove("text-blue-600");
				progress.classList.add("text-red-600");
			});
		});

		function stopCrawler() {
			fetch("/stop", { method: "POST" })
				.then(() => {
					console.log("Crawler stopped");
					progress.textContent = "Crawler stopped.";
					progress.classList.remove("text-blue-600");
					progress.classList.add("text-red-600");
				})
				.catch(err => console.error("Error stopping crawler:", err));
		}

		function clearResults() {
			tbody.innerHTML = "";
		}
	</script>
</body>
</html>
	`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	clientsMux.Lock()
	clients[conn] = true
	clientsMux.Unlock()

	// Send existing results to new client
	results.Lock()
	for _, result := range results.Data {
		data, _ := json.Marshal(result)
		conn.WriteMessage(websocket.TextMessage, data)
	}
	results.Unlock()

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			clientsMux.Lock()
			delete(clients, conn)
			clientsMux.Unlock()
			return
		}
	}
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var requestData struct {
		URL     string `json:"url"`
		Workers int    `json:"workers,string"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate URL
	if requestData.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(requestData.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		http.Error(w, "Invalid URL. Must be a valid http or https URL", http.StatusBadRequest)
		return
	}

	// Validate number of workers
	numWorkers := requestData.Workers
	if numWorkers <= 0 {
		numWorkers = 10 // Default to 10 workers
	}

	// First, stop any existing crawl
	stoppedMux.Lock()
	if !stopped {
		stopped = true
		close(urlsChan)
		// Wait for any existing workers to finish
		stoppedMux.Unlock()
		crawlWG.Wait()
	} else {
		stoppedMux.Unlock()
	}

	// Reset for new crawl
	urlsChan = make(chan string, 100)
	results.Lock()
	results.Data = make([]CrawlResult, 0)
	results.Unlock()
	visited := &VisitedURLs{urls: make(map[string]bool)}

	// Start new crawl
	stoppedMux.Lock()
	stopped = false
	stoppedMux.Unlock()

	// Start workers
	log.Printf("Starting %d workers for URL: %s\n", numWorkers, requestData.URL)
	for i := 0; i < numWorkers; i++ {
		crawlWG.Add(1)
		go worker(i, &crawlWG, visited, requestData.URL)
	}

	// Start crawling
	urlsChan <- requestData.URL

	w.Write([]byte("Crawler started"))
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stoppedMux.Lock()
	if !stopped {
		stopped = true
		close(urlsChan) // Close the channel to stop workers
	}
	stoppedMux.Unlock()
	log.Println("Crawler stopped by user")
	w.Write([]byte("Crawler stopped"))
}

func normalizeURL(link, baseURL string) string {
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return link // Return as-is if base URL is invalid
	}
	parsedLink, err := url.Parse(link)
	if err != nil {
		return link // Return as-is if link is invalid
	}
	return parsedBase.ResolveReference(parsedLink).String()
}
