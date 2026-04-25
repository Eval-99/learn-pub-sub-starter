package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	const connectionString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatalf("Failed connection to RabbitMQ: %v", err)
		return
	}
	defer conn.Close()
	fmt.Println("Peril game client connected to RabbitMQ!")

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Could not get username: %v", err)
		return
	}

	userState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(userState),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
		return
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+username,
		routing.ArmyMovesPrefix+".*",
		pubsub.Transient,
		func(move gamelogic.ArmyMove) pubsub.Acktype {
			outcome := userState.HandleMove(move)

			var result pubsub.Acktype

			switch outcome {
			case gamelogic.MoveOutComeSafe:
				result = pubsub.Ack
			case gamelogic.MoveOutcomeMakeWar:
				result = pubsub.Ack
			case gamelogic.MoveOutcomeSamePlayer:
				result = pubsub.NackDiscard
			}

			return result
		},
	)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
		return
	}

	for {
		userInputs := gamelogic.GetInput()
		if userInputs == nil {
			continue
		}

		switch userInputs[0] {
		case "spawn":
			userState.CommandSpawn(userInputs)
		case "move":
			move, err := userState.CommandMove(userInputs)
			if err != nil {
				fmt.Println(err)
				continue
			}

			err = pubsub.PublishJSON(
				publishCh,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+username,
				move,
			)
			if err != nil {
				fmt.Printf("error publishing move: %v\n", err)
				continue
			}

		case "status":
			userState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Not a valid command")
		}

	}
}
