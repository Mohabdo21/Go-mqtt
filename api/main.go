package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"mqttGo/rabbitmq"
	"mqttGo/sse"

	_ "github.com/lib/pq"
)

type TemperatureReading struct {
	Topic      string    `json:"topic"`
	Payload    string    `json:"payload"`
	ReceivedAt time.Time `json:"received_at"`
}

// startRabbitMQConsumer starts a RabbitMQ consumer for the specified topics
func startRabbitMQConsumer(rmqClient *rabbitmq.Client, broker *sse.Broker, topics []string) {
	for _, topic := range topics {
		go func(t string) {
			err := rmqClient.Consume(t, func(body []byte) {
				var data sse.SensorData
				if err := json.Unmarshal(body, &data); err != nil {
					log.Printf("Error unmarshal sensor data: %v", err)
					return
				}
				broker.Publish(data)
			})
			if err != nil {
				log.Printf("Failed to start RabbitMQ consumer for topic %s: %v", t, err)
			}
		}(topic)
	}
}

func main() {
	// Initialize the database connection
	db, err := sql.Open("postgres", "host=localhost port=5432 user=mqtt_user password=mqtt_pass dbname=mqtt_data sslmode=disable")
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer db.Close()

	// Create and start the SSE broker
	broker := sse.NewBroker()
	broker.Start()
	defer broker.Stop()

	rmqClient, err := rabbitmq.NewClient("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmqClient.Close()

	topics := []string{"AM300/OUTSIDE", "AM300/INSIDE"}
	startRabbitMQConsumer(rmqClient, broker, topics)

	// Handler for historical data
	http.HandleFunc("/api/readings", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT topic, payload, received_at
			FROM temperature_readings
			ORDER BY received_at DESC
			LIMIT 100
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var readings []TemperatureReading
		for rows.Next() {
			var r TemperatureReading
			if err := rows.Scan(&r.Topic, &r.Payload, &r.ReceivedAt); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			readings = append(readings, r)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(readings)
	})

	// Handler for live sensor data via SSE
	http.Handle("/api/sensor-stream", broker)

	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Println("API running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
