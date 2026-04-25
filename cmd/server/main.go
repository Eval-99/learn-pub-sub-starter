package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {
	const connectionString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatalf("Failed connection to RabbitMQ: %v", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connection to RabbitMQ successful")

	amqpChan, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ channel: %v", err)
		return
	}

	err = pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.Durable,
		func(gameLog routing.GameLog) pubsub.Acktype {
			defer fmt.Print("> ")
			err := gamelogic.WriteLog(gameLog)
			if err != nil {
				fmt.Printf("error writing log: %v\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		},
	)
	if err != nil {
		log.Fatalf("Could not subscribe to log: %v", err)
		return
	}

	gamelogic.PrintServerHelp()
	for {
		userInputs := gamelogic.GetInput()
		if userInputs == nil {
			continue
		}

		switch userInputs[0] {
		case "pause":
			fmt.Println("Pausing game...")
			err = pubsub.PublishJSON(
				amqpChan,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{IsPaused: true},
			)
			if err != nil {
				log.Fatalf("Failed to publish JSON: %v", err)
				return
			}
		case "resume":
			fmt.Println("Resuming game...")
			err = pubsub.PublishJSON(
				amqpChan,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{IsPaused: false},
			)
			if err != nil {
				log.Fatalf("Failed to publish JSON: %v", err)
				return
			}
		case "quit":
			fmt.Println("Quiting game...")
			return
		default:
			fmt.Println("Not a valid command")
		}
	}
}
