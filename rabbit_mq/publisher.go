package rabbit_mq

import (
	ampq "github.com/rabbitmq/amqp091-go"
	"log"
)

type Publisher[T any] interface {
	Publish(data T) error
}

type BasePublisher struct {
	Ch *ampq.Channel
}

func (bqc *BasePublisher) Publish(msg string) error {
	log.Print("Base publisher received message: ", msg)
	return nil
}
