# Concurrent Web Crawler

## Introduction
The Concurrent Web Crawler is a Go-based application designed to crawl web pages efficiently using Go's powerful concurrency features. This project was undertaken to deepen my understanding of Go's concurrency model, specifically Goroutines, Channels, and synchronization primitives.

## Objectives
- **Learn Go's Concurrency Model**: Gain hands-on experience with Goroutines and Channels.
- **Implement Synchronization**: Understand and apply synchronization tools like Mutexes and WaitGroups.
- **Build a Scalable Application**: Create a web crawler that can efficiently handle multiple tasks concurrently.
- **Respect Web Crawling Ethics**: Implement features to respect robots.txt and avoid overloading servers.

## Features
- **Concurrent Crawling**: Utilizes Goroutines to fetch and parse web pages concurrently.
- **URL Parsing and Normalization**: Extracts and normalizes links from HTML content.
- **Visited URLs Tracking**: Keeps track of visited URLs to prevent duplicate processing.
- **Robots.txt Compliance**: Checks and respects robots.txt directives for each domain.
- **Rate Limiting**: Implements rate limiting to prevent overwhelming web servers.
- **Graceful Shutdown**: Handles system signals to ensure the crawler can be stopped gracefully.

## Project Structure
```
concurrent-web-crawler/
├── fetch.go
├── go.mod
├── go.sum
├── main.go
├── parse.go
├── README.md
└── robots.go
```

## Installation and Usage
### Installation
#### Clone the Repository
```sh
git clone https://github.com/PoowadolDev/Concurrent-Web-Crawler.git
cd concurrent-web-crawler
```

#### Install Dependencies
```sh
go mod download
```

### Running the Crawler
```sh
go run .
```

## What I Learned
### Go's Concurrency Model
- **Goroutines**: Learned how to launch lightweight threads using Goroutines, allowing functions to run concurrently.
- **Channels**: Understood how to use Channels for communication between Goroutines, enabling safe data transfer without explicit locking.

### Synchronization Primitives
- **WaitGroups**: Used `sync.WaitGroup` to wait for a collection of Goroutines to finish executing.
- **Mutexes**: Applied `sync.Mutex` to protect shared resources and prevent race conditions.

### Web Crawling Techniques
- **HTTP Requests**: Gained experience with the `net/http` package to perform HTTP requests and handle responses.
- **HTML Parsing**: Utilized the `goquery` library to parse HTML documents and extract links efficiently.
- **URL Normalization**: Learned to normalize and resolve relative URLs to absolute URLs, ensuring accurate crawling paths.

### Ethical Crawling Practices
- **Robots.txt Compliance**: Implemented functionality to read and respect `robots.txt` files, adhering to website crawling policies.
- **Rate Limiting**: Introduced rate limiting to control the frequency of HTTP requests, preventing server overloads.
- **Domain Restrictions**: Limited crawling to specific domains to avoid unintended crawling of external sites.

### Error Handling and Logging
- **Robust Error Handling**: Developed strategies for retrying failed requests and handling various types of errors gracefully.
- **Logging**: Employed the `log` package to record significant events and errors, aiding in debugging and monitoring.

## Conclusion
This project provided a comprehensive exploration of Go's concurrency features and practical application in building a real-world tool. By developing the Concurrent Web Crawler, I not only enhanced my technical skills but also gained valuable insights into software design, ethical considerations, and best practices in Go programming.