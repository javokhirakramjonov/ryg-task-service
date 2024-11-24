package rabbit_mq

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"ryg-task-service/conf"
)

const exchangeName = "email_service_topics"

type PublisherManager struct {
	conn                       *amqp.Connection
	ch                         *amqp.Channel
	GenericEmailQueuePublisher *GenericEmailPublisher
}

func NewPublisherManager(cnf conf.RabbitMQConfig) PublisherManager {
	url := "amqp://" + cnf.User + ":" + cnf.Password + "@" + cnf.Host + ":" + cnf.Port + "/"
	conn, err := amqp.Dial(url)
	failOnError(err, "Failed to connect to RabbitMQ")
	log.Printf("Connected to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	log.Printf("Opened a channel")

	err = ch.ExchangeDeclare(
		exchangeName, // exchange name
		"topic",      // exchange type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	genericEmailPublisher := NewGenericEmailQueuePublisher(ch, exchangeName)

	return PublisherManager{
		conn:                       conn,
		ch:                         ch,
		GenericEmailQueuePublisher: genericEmailPublisher,
	}
}

func (qcm *PublisherManager) Close() {
	err := qcm.ch.Close()
	failOnError(err, "Failed to close channel")

	err = qcm.conn.Close()
	failOnError(err, "Failed to close connection")
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
