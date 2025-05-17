package main

import (
	"mqttGo/mqttcommon"
	"time"
)

// main function initializes the MQTT client and starts publishing data.
func main() {
	client := mqttcommon.CreateClient("AM300_Simulator")
	if err := mqttcommon.ConnectClient(client); err != nil {
		panic(err)
	}

	mqttcommon.PublishEncryptedReading(client, "AM300/INSIDE", time.Second)
}
