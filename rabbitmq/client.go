package rabbitmq

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchangeName = "sensor_data"
const exchangeType = "topic"

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewClient initializes and returns a RabbitMQ client
func NewClient(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := channel.ExchangeDeclare(
		exchangeName,
		exchangeType,
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &Client{conn: conn, channel: channel}, nil
}

// Publish sends a message to the given routing key (topic)
func (c *Client) Publish(topic string, body []byte) error {
	return c.channel.Publish(
		exchangeName,
		topic,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
}

// Consume sets up a subscription to messages using a routing key
func (c *Client) Consume(topic string, handler func([]byte)) error {
	queue, err := c.channel.QueueDeclare(
		"",    // empty = generated name (exclusive queue)
		false, // durable
		true,  // autoDelete
		true,  // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		return fmt.Errorf("queue declare failed: %w", err)
	}

	err = c.channel.QueueBind(
		queue.Name,
		topic,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("queue bind failed: %w", err)
	}

	msgs, err := c.channel.Consume(
		queue.Name,
		"",    // consumer
		true,  // auto-ack
		true,  // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume failed: %w", err)
	}

	go func() {
		for msg := range msgs {
			handler(msg.Body)
		}
	}()

	return nil
}

// Close gracefully closes the channel and connection
func (c *Client) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
