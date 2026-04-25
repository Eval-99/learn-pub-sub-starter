package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string

const (
	Durable   SimpleQueueType = "durable"
	Transient SimpleQueueType = "transient"
)

type Acktype int

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
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

	params := amqp.Table{"x-dead-letter-exchange": routing.ExchangePerilDeadLetter}

	amqpQueue, err := amqpChan.QueueDeclare(queueName, isDurable, isAutoDelete, isExclusive, false, params)
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

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	amqpChan, amqpQueue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	deliveries, err := amqpChan.Consume(amqpQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for delivery := range deliveries {
			var container T
			err = json.Unmarshal(delivery.Body, &container)
			if err != nil {
				fmt.Println(err)
				continue
			}

			result := handler(container)
			switch result {
			case Ack:
				delivery.Ack(false)
			case NackRequeue:
				delivery.Nack(false, true)
			case NackDiscard:
				delivery.Nack(false, false)
			}
		}
	}()

	return nil
}
