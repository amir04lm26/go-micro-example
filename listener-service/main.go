package main

import (
	"errors"
	"listener/event"
	"log"
	"math"
	"net"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// NOTE: Install third party libraries
/*
	go get github.com/rabbitmq/amqp091-go
*/

func main() {
	// try to connect to rabbitmq
	rabbitConn, err := connectToRabbitMQ()
	if err != nil {
		log.Fatalln(err)
	}
	defer rabbitConn.Close()

	// start listening for messages
	log.Println("Listening for and consuming RabbitMQ messages...")

	// create consumer
	consumer, err := event.NewConsumer(rabbitConn)
	if err != nil {
		log.Panicln(err)
	}

	// watch the queue and consume events
	err = consumer.Listen([]string{"log.INFO", "log.WARNING", "log.ERROR"})
	if err != nil {
		log.Fatalln(err)
	}
}

func connectToRabbitMQ() (*amqp.Connection, error) {
	var counts int64
	var backOff = 1 * time.Second
	var connection *amqp.Connection

	// don't continue until rabbit is ready
	for {
		c, err := amqp.Dial("amqp://root:123456@rabbitmq")
		if err == nil {
			log.Println("Connected to RabbitMQ!")
			connection = c
			break
		}

		var netErr net.Error
		if errors.As(err, &netErr) {
			log.Println("RabbitMQ not yet ready...")
		} else {
			log.Println("Non-network error:", err)
			return nil, err
		}

		counts++
		if counts > 5 {
			log.Println(err)
			return nil, err
		}

		backOff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		log.Println("backing off...")
		time.Sleep(backOff)
	}

	return connection, nil
}
