package main

import (
	"database/sql"
	"encoding/json"
	"log"

	"word-ladder/events"
)

// ConsumeEvents reads game events from RabbitMQ forever and saves them
// in SQLite. It runs in its own goroutine (started in main).
func ConsumeEvents(rabbitURL string, db *sql.DB) {
	conn := events.Connect(rabbitURL)
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open RabbitMQ channel: %v", err)
	}
	defer ch.Close()

	events.DeclareQueue(ch)

	messages, err := ch.Consume(
		events.QueueName, // queue
		"stats-server",   // consumer name
		false,            // auto-ack: false, we ack after saving to the database
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		log.Fatalf("failed to start consuming: %v", err)
	}

	log.Printf("waiting for game events on queue '%s'", events.QueueName)

	for msg := range messages {
		var event events.GameEvent

		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("invalid message, discarding: %s", msg.Body)
			msg.Ack(false)
			continue
		}

		event.Username = cleanUsername(event.Username)

		if err := SaveResult(db, event); err != nil {
			log.Printf("could not save event: %v", err)
			msg.Nack(false, true) // put it back in the queue
			continue
		}

		msg.Ack(false)
		log.Printf("received event: game %d, %s, %s, %d turns -> saved to SQLite",
			event.GameID, event.Username, event.Result, event.Turns)
	}

	log.Fatal("RabbitMQ connection closed")
}
