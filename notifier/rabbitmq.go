package notifier

import (
	"github.com/mijara/statspout/log"
	"github.com/streadway/amqp"
)

// RabbitMQ is a notifier that sends messages to RabbitMQ.
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   *amqp.Queue
}

// NewRabbitMQ initializes a new instance of the RabbitMQ notifier, creating
// the client and queue.
func NewRabbitMQ(uri, queueName string) *RabbitMQ {
	conn, err := amqp.Dial(uri)
	if err != nil {
		log.Error.Println("error while initializing RabbitMQ notifier: " + err.Error())
		return nil
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Error.Println("failed to open RabbitMQ channel: " + err.Error())
		return nil
	}

	queue, err := channel.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Error.Println("failed to declare RabbitMQ queue: " + err.Error())
		return nil
	}

	return &RabbitMQ{
		conn:    conn,
		channel: channel,
		queue:   &queue,
	}
}

// Notify sends a each message to a RabbitMQ broker.
func (rmq *RabbitMQ) Notify(messages []string) {
	for _, message := range messages {
		rmq.channel.Publish(
			"",
			rmq.queue.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(message),
			})
	}
}

// Close the RabbitMQ client.
func (rmq *RabbitMQ) Close() {
	rmq.channel.Close()
	rmq.conn.Close()
}
