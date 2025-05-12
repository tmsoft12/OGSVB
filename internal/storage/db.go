package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v4/pgxpool"
)

var DbPool *pgxpool.Pool

func InitDB(connString string) error {
	var err error

	DbPool, err = pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	if err := DbPool.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	log.Println("Database connection successful!")

	var test string
	err = DbPool.QueryRow(context.Background(), "SELECT 'OK'").Scan(&test)
	if err != nil || test != "OK" {
		log.Printf("Database connection verified but unexpected test result. Test value: %s\n", test)
		return fmt.Errorf("database connection test failed: %v", err)
	}

	log.Println("Database connection and test query successful!")
	return nil
}

func SaveEventToDB(topic, value, timestamp string) error {
	query := `INSERT INTO events (topic, value, timestamp) VALUES ($1, $2, $3)`
	cmdTag, err := DbPool.Exec(context.Background(), query, topic, value, timestamp)
	if err != nil {
		log.Printf("Data save error: %v\n", err)
		return fmt.Errorf("data save error: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Println("No data inserted into database!")
		return fmt.Errorf("no data inserted into database!")
	}
	fmt.Println("Database connection established, data being saved...")

	log.Printf("Data saved: Topic: %s, Value: %s, Timestamp: %s\n", topic, value, timestamp)

	message := fmt.Sprintf("Topic: %s, Value: %s, Timestamp: %s", topic, value, timestamp)
	BroadcastCh <- message
	return nil
}
