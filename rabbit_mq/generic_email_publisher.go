package rabbit_mq

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"log"
	"ryg-task-service/gen_proto/email_service"
)

const (
	genericEmailRoutingKey = "generic_email"
)

type GenericEmailPublisher struct {
	exchangeName string
	BasePublisher
}

func NewGenericEmailQueuePublisher(ch *amqp.Channel, exchangeName string) *GenericEmailPublisher {
	return &GenericEmailPublisher{
		exchangeName: exchangeName,
		BasePublisher: BasePublisher{
			Ch: ch,
		},
	}
}

func (c *GenericEmailPublisher) Publish(data *email_service.GenericEmail) error {
	log.Printf("GenericEmailPublisher received message: %v", data)

	body, err := proto.Marshal(data)
	if err != nil {
		return err
	}

	err = c.Ch.Publish(
		c.exchangeName,         // exchange name
		genericEmailRoutingKey, // routing key (dynamic for topic exchange)
		false,                  // mandatory
		false,                  // immediate
		amqp.Publishing{
			ContentType: "application/protobuf",
			Body:        body,
		},
	)
	if err != nil {
		return err
	}

	return nil
}
