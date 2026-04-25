package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

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

	gs := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+gs.GetUsername(),
		routing.ArmyMovesPrefix+".*",
		pubsub.Transient,
		handlerMove(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to army moves: %v", err)
	}
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(),
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.Durable,
		handlerWar(gs, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to war: %v", err)
	}

	for {
		userInputs := gamelogic.GetInput()
		if userInputs == nil {
			continue
		}

		switch userInputs[0] {
		case "spawn":
			gs.CommandSpawn(userInputs)
		case "move":
			move, err := gs.CommandMove(userInputs)
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
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			if len(userInputs) < 2 {
				fmt.Println("You need a number to spam nitwit!")
				continue
			}

			i, err := strconv.Atoi(userInputs[1])
			if err != nil {
				fmt.Println("That's not a number nitwit!")
				continue
			}

			for range i {
				message := gamelogic.GetMaliciousLog()
				err := pubsub.PublishGob(
					publishCh,
					routing.ExchangePerilTopic,
					routing.GameLogSlug+"."+gs.GetUsername(),
					routing.GameLog{
						CurrentTime: time.Now().UTC(),
						Message:     message,
						Username:    gs.GetUsername(),
					},
				)
				if err != nil {
					fmt.Printf("Could not publish log: %v\n", err)
				}
			}
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Not a valid command")
		}

	}
}
