package notifier

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mijara/statspout/log"
	"github.com/mijara/statspoutalarm/detections"
	"github.com/streadway/amqp"
)

type Priority string

// Predefined priority levels.
const (
	INFO     = Priority("INFO")
	CRITICAL = Priority("CRITICAL")
	WARNING  = Priority("WARNING")
)

// RabbitMQ is a notifier that sends messages to RabbitMQ.
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   *amqp.Queue
}

// Alarm represent and ALMA Alarm for ElasticSearch indexing.
type Alarm struct {
	Timestamp time.Time `json:"@timestamp"`
	Name      string    `json:"path"`
	Priority  Priority  `json:"priority"`

	Body struct {
		Message string `json:"message"`
	} `json:"body"`
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
func (rmq *RabbitMQ) Notify(detections []*detections.Detection) {
	for _, detection := range detections {
		alarm := Alarm{
			Timestamp: detection.Timestamp.UTC(),
			Name:      fmt.Sprintf("OFFLINE/%s/%s", detection.ContainerName, detection.Resource),
			Priority:  WARNING,
		}

		alarm.Body.Message = detection.Message

		body, err := json.Marshal(alarm)
		if err != nil {
			log.Error.Println("error marshaling alarm json: " + err.Error())
			return
		}

		rmq.channel.Publish(
			"",
			rmq.queue.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        body,
			})
	}
}

// Close the RabbitMQ client.
func (rmq *RabbitMQ) Close() {
	rmq.channel.Close()
	rmq.conn.Close()
}
