package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"mqttGo/rabbitmq"
	"mqttGo/store"
)

const (
	BrokerAddress   = "tcp://mqtt.eclipseprojects.io:1883"
	ClientID        = "Smart_Home_Subscriber"
	xorKey          = 0x5A
	RabbitMQURL     = "amqp://guest:guest@localhost:5672"
	WorkerCount     = 4
	MessageBuffer   = 100
	ShutdownTimeout = 10 * time.Second
)

type SensorData struct {
	Topic       string    `json:"topic"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	CO2         float64   `json:"co2"`
	Timestamp   time.Time `json:"timestamp"`
}

type Message struct {
	Topic    string
	Payload  string
	Received time.Time
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup DB
	db, err := store.NewDB()
	if err != nil {
		log.Fatalf("DB init error: %v", err)
	}
	defer db.Close()

	// Setup RabbitMQ
	rmqClient, err := rabbitmq.NewClient(RabbitMQURL)
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer rmqClient.Close()

	// Topics to subscribe to
	topics := []string{"AM300/OUTSIDE", "AM300/INSIDE"}

	// Channels per topic
	topicChannels := make(map[string]chan Message)
	var wg sync.WaitGroup

	for _, topic := range topics {
		ch := make(chan Message, MessageBuffer)
		topicChannels[topic] = ch

		for range WorkerCount {
			wg.Add(1)
			go worker(ctx, &wg, ch, db, rmqClient)
		}
	}

	// Setup shared MQTT client with single handler
	client := setupMQTTClient(topics, topicChannels)
	defer client.Disconnect(250)

	// Wait for shutdown signal
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	// Graceful shutdown
	log.Println("Starting graceful shutdown...")
	cancel()

	for _, ch := range topicChannels {
		close(ch)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers finished")
	case <-time.After(ShutdownTimeout):
		log.Println("Shutdown timeout reached")
	}

	log.Println("Shutdown complete")
}

func setupMQTTClient(topics []string, topicChannels map[string]chan Message) mqtt.Client {
	opts := mqtt.NewClientOptions().
		AddBroker(BrokerAddress).
		SetClientID(ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetKeepAlive(30 * time.Second).
		SetCleanSession(true).
		SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
			topic := msg.Topic()
			ch, ok := topicChannels[topic]
			if !ok {
				log.Printf("Received message for unregistered topic: %s", topic)
				return
			}
			select {
			case ch <- Message{
				Topic:    topic,
				Payload:  string(msg.Payload()),
				Received: time.Now(),
			}:
			default:
				log.Printf("WARNING: Buffer full for topic %s, dropping message", topic)
			}
		})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connection error: %v", token.Error())
	}

	for _, topic := range topics {
		if token := client.Subscribe(topic, 2, nil); token.Wait() && token.Error() != nil {
			log.Fatalf("Failed to subscribe to %s: %v", topic, token.Error())
		} else {
			log.Printf("Subscribed to topic: %s", topic)
		}
	}

	return client
}

func worker(ctx context.Context, wg *sync.WaitGroup, messages <-chan Message, db *store.DB, rmqClient *rabbitmq.Client) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			processMessage(msg, db, rmqClient)
		}
	}
}

func processMessage(msg Message, db *store.DB, rmqClient *rabbitmq.Client) {
	start := time.Now()

	decoded, err := decodeSensorPayload(msg.Payload)
	if err != nil {
		log.Printf("Decode error [%s]: %v", msg.Topic, err)
		return
	}

	if err := db.SaveDecoded(msg.Topic, decoded); err != nil {
		log.Printf("DB save error [%s]: %v", msg.Topic, err)
	}

	sensorData := SensorData{
		Topic:       msg.Topic,
		Temperature: decoded.Temperature,
		Humidity:    decoded.Humidity,
		CO2:         decoded.CO2,
		Timestamp:   msg.Received,
	}

	if err := publishToRabbitMQ(rmqClient, sensorData); err != nil {
		log.Printf("RabbitMQ publish error [%s]: %v", msg.Topic, err)
	}

	log.Printf("Processed %s in %v", msg.Topic, time.Since(start))
}

func publishToRabbitMQ(client *rabbitmq.Client, data SensorData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	return client.Publish(data.Topic, jsonData)
}

func decodeSensorPayload(hexPayload string) (*store.DecodedReading, error) {
	raw, err := hex.DecodeString(hexPayload)
	if err != nil {
		return nil, fmt.Errorf("hex decode error: %w", err)
	}

	for i := range raw {
		raw[i] ^= xorKey
	}

	var temp, hum, co2 float64
	for i := 0; i < len(raw)-3; i += 4 {
		channel := raw[i]
		typ := raw[i+1]
		val := uint16(raw[i+2])<<8 | uint16(raw[i+3])

		switch {
		case channel == 0x03 && typ == 0x67:
			temp = float64(val) / 10.0
		case channel == 0x04 && typ == 0x68:
			hum = float64(val) / 2.0
		case channel == 0x07 && typ == 0x7d:
			co2 = float64(val)
		}
	}

	return &store.DecodedReading{
		Temperature: temp,
		Humidity:    hum,
		CO2:         co2,
	}, nil
}
