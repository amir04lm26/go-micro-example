package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// NOTE: Install grpc tools
/*
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	brew install protobuf
*/

// NOTE: Use protoc tools
/*
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative logs.proto
*/

const webPort = "80"

type Config struct {
	rabit *amqp.Connection
}

func main() {
	// try to connect to rabbitmq
	rabbitConn, err := connectToRabbitMQ()
	if err != nil {
		log.Fatalln(err)
	}
	defer rabbitConn.Close()

	app := Config{
		rabit: rabbitConn,
	}

	log.Printf("Starting broker service on port %s\r\n", webPort)

	// define http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	// start the server
	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
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
