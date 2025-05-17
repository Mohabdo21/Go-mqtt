package store

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

type DecodedReading struct {
	Temperature float64
	Humidity    float64
	CO2         float64
}

func NewDB() (*DB, error) {
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		connStr = "postgres://mqtt_user:mqtt_pass@localhost:5432/mqtt_data"
	}

	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	log.Println("Connected to TimescaleDB")
	return &DB{Pool: pool}, nil
}

func (db *DB) SaveDecoded(topic string, reading *DecodedReading) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx,
		`INSERT INTO temperature_readings (topic, temperature, humidity, co2) VALUES ($1, $2, $3, $4)`,
		topic, reading.Temperature, reading.Humidity, reading.CO2,
	)
	return err
}

func (db *DB) Close() {
	db.Pool.Close()
}
