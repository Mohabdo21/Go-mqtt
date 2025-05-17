package mqttcommon

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	BrokerAddress = "tcp://mqtt.eclipseprojects.io:1883"
	xorKey        = 0x5A
)

type SensorReading struct {
	Temperature float64
	Humidity    float64
	CO2         float64
}

// CreateClient initializes a new MQTT client with the given client ID.
func CreateClient(clientID string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(BrokerAddress)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	return mqtt.NewClient(opts)
}

// ConnectClient connects the MQTT client to the broker.
func ConnectClient(client mqtt.Client) error {
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("connection failed: %v", token.Error())
	}
	return nil
}

// PublishEncryptedReading publishes encrypted sensor data mimicking Milesight format.
func PublishEncryptedReading(client mqtt.Client, topic string, interval time.Duration) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			reading := generateRandomReading()
			payload := encodeSensorPayload(reading)
			encrypted := xorEncrypt(payload, xorKey)
			hexData := hex.EncodeToString(encrypted)

			token := client.Publish(topic, 0, false, hexData)
			if token.Wait() && token.Error() != nil {
				fmt.Printf("Publish error: %v\n", token.Error())
				continue
			}
			fmt.Printf("Published encrypted data: %s -> Topic: %s\n", hexData, topic)
		case <-sigChan:
			fmt.Println("Exiting...")
			client.Disconnect(250)
			return
		}
	}
}

// generateRandomReading generates a random sensor reading.
func generateRandomReading() SensorReading {
	return SensorReading{
		Temperature: rand.Float64()*10 + 20,   // 20.0–30.0 °C
		Humidity:    rand.Float64()*20 + 40,   // 40–60 %RH
		CO2:         rand.Float64()*600 + 400, // 400–1000 ppm
	}
}

// encodeSensorPayload encodes the sensor reading into a byte array.
func encodeSensorPayload(r SensorReading) []byte {
	buf := []byte{}

	// Temperature - Channel 3, Type 0x67, Unit 0.1°C
	temp := uint16(r.Temperature * 10)
	buf = append(buf, 0x03, 0x67, byte(temp>>8), byte(temp&0xFF))

	// Humidity - Channel 4, Type 0x68, Unit 0.5%RH
	hum := uint16(r.Humidity * 2)
	buf = append(buf, 0x04, 0x68, byte(hum>>8), byte(hum&0xFF))

	// CO2 - Channel 7, Type 0x7d, Unit 1 ppm
	co2 := uint16(r.CO2)
	buf = append(buf, 0x07, 0x7d, byte(co2>>8), byte(co2&0xFF))

	return buf
}

// xorEncrypt encrypts the data using XOR with the given key.
func xorEncrypt(data []byte, key byte) []byte {
	encrypted := make([]byte, len(data))
	for i, b := range data {
		encrypted[i] = b ^ key
	}
	return encrypted
}
