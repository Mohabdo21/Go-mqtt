package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// SensorData represents structured sensor information
type SensorData struct {
	Topic       string    `json:"topic"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	CO2         float64   `json:"co2"`
	Timestamp   time.Time `json:"timestamp"`
}

// Broker manages SSE clients and broadcasts SensorData
type Broker struct {
	clients        map[chan SensorData]bool
	newClients     chan chan SensorData
	closedClients  chan chan SensorData
	sensorData     chan SensorData
	mutex          sync.RWMutex
	isRunning      bool
	notifyShutdown chan struct{}
}

// NewBroker initializes a new SSE broker
func NewBroker() *Broker {
	return &Broker{
		clients:        make(map[chan SensorData]bool),
		newClients:     make(chan chan SensorData),
		closedClients:  make(chan chan SensorData),
		sensorData:     make(chan SensorData, 100),
		notifyShutdown: make(chan struct{}),
	}
}

// Start launches the broker loop
func (b *Broker) Start() {
	b.mutex.Lock()
	if b.isRunning {
		b.mutex.Unlock()
		return
	}
	b.isRunning = true
	b.mutex.Unlock()

	go func() {
		for {
			select {
			case client := <-b.newClients:
				b.mutex.Lock()
				b.clients[client] = true
				b.mutex.Unlock()
				log.Printf("Client added. Total: %d", len(b.clients))

			case client := <-b.closedClients:
				b.mutex.Lock()
				delete(b.clients, client)
				close(client)
				b.mutex.Unlock()
				log.Printf("Client removed. Total: %d", len(b.clients))

			case data := <-b.sensorData:
				b.mutex.RLock()
				for client := range b.clients {
					select {
					case client <- data:
					default:
						log.Println("Client buffer full, skipping...")
					}
				}
				b.mutex.RUnlock()

			case <-b.notifyShutdown:
				b.mutex.Lock()
				for client := range b.clients {
					delete(b.clients, client)
					close(client)
				}
				b.isRunning = false
				b.mutex.Unlock()
				return
			}
		}
	}()
}

// Stop gracefully shuts down the broker
func (b *Broker) Stop() {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if b.isRunning {
		b.notifyShutdown <- struct{}{}
	}
}

// Publish adds new data to broadcast queue
func (b *Broker) Publish(data SensorData) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if b.isRunning {
		b.sensorData <- data
	}
}

// ServeHTTP satisfies http.Handler to support SSE
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan SensorData, 10)
	b.newClients <- clientChan

	notify := r.Context().Done()

	go func() {
		<-notify
		b.closedClients <- clientChan
		log.Println("Client connection closed")
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Initial keep-alive
	fmt.Fprintf(w, "event: keepalive\ndata: Connected\n\n")
	flusher.Flush()

	for data := range clientChan {
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Printf("Marshal error: %v", err)
			continue
		}
		fmt.Fprintf(w, "event: sensor-update\ndata: %s\n\n", jsonData)
		flusher.Flush()
	}
}
