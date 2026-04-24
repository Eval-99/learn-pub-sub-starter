package pubsub

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string

const (
	Durable   SimpleQueueType = "durable"
	Transient SimpleQueueType = "transient"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{ContentType: "application/json", Body: data},
	)
	if err != nil {
		return err
	}

	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	amqpChan, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ channel: %v", err)
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	var isDurable bool
	var isAutoDelete bool
	var isExclusive bool
	switch queueType {
	case Durable:
		isDurable = true
		isAutoDelete = false
		isExclusive = false
	case Transient:
		isDurable = false
		isAutoDelete = true
		isExclusive = true
	default:
		log.Fatalf("Invalid queueType, values should be Durable or Transient: %v", err)
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	amqpQueue, err := amqpChan.QueueDeclare(queueName, isDurable, isAutoDelete, isExclusive, false, nil)
	if err != nil {
		log.Fatalf("Failed to queue: %v", err)
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	err = amqpChan.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	return amqpChan, amqpQueue, nil
}
