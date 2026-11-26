package events

import (
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connect connects to RabbitMQ. RabbitMQ can take a few seconds to start
// (especially in Docker), so we try several times before giving up.
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

// DeclareQueue creates the game_events queue if it does not exist yet.
// Both the publisher and the consumer call this with the same settings.
func DeclareQueue(ch *amqp.Channel) {
	_, err := ch.QueueDeclare(
		QueueName, // name
		true,      // durable: the queue survives a RabbitMQ restart
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Fatalf("failed to declare queue: %v", err)
	}
}
