package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

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

	gamelogic.PrintServerHelp()
	for {
		userInputs := gamelogic.GetInput()
		if userInputs == nil {
			continue
		}

		if userInputs[0] == "pause" {
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
		} else if userInputs[0] == "resume" {
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
		} else if userInputs[0] == "quit" {
			fmt.Println("Quiting game...")
			break
		} else {
			fmt.Println("Not a valid command")
		}

	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("\nClossing connection to RabbitMQ")
}
