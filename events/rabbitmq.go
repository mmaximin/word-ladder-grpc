package events

import (
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Connect(url string) *amqp.Connection {
	for i := 1; i <= 15; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			log.Printf("connected to RabbitMQ at %s", url)
			return conn
		}

		log.Printf("RabbitMQ not ready (attempt %d): %v", i, err)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("could not connect to RabbitMQ at %s", url)
	return nil
}

func DeclareQueue(ch *amqp.Channel) {
	_, err := ch.QueueDeclare(
		QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to declare queue: %v", err)
	}
}
