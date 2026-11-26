package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"word-ladder/events"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher sends finished game events to RabbitMQ.
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.Mutex // an AMQP channel should not be used by two goroutines at once
}

func NewPublisher(url string) *Publisher {
	conn := events.Connect(url)

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open RabbitMQ channel: %v", err)
	}

	events.DeclareQueue(ch)

	return &Publisher{
		conn:    conn,
		channel: ch,
	}
}

func (p *Publisher) Publish(event events.GameEvent) {
	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("could not convert event to JSON: %v", err)
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Empty exchange name = default exchange, which sends the message
	// directly to the queue with the same name as the routing key.
	err = p.channel.PublishWithContext(ctx,
		"",               // exchange
		events.QueueName, // routing key (queue name)
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		log.Printf("could not publish event: %v", err)
		return
	}

	log.Printf("published event to RabbitMQ: %s", body)
}

func (p *Publisher) Close() {
	p.channel.Close()
	p.conn.Close()
}
