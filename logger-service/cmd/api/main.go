package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NOTE: Install mongo driver
/*
	go get go.mongodb.org/mongo-driver/mongo
	go get go.mongodb.org/mongo-driver/mongo/options
	go get go.mongodb.org/mongo-driver/bson
*/

// NOTE: Running the service
/*
	go run ./cmd/api
*/

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

const (
	webPort  = "80"
	rpcPort  = "5001"
	mongoURL = "mongodb://mongo:27017"
	grpcPort = "50001"
)

var client *mongo.Client

type Config struct {
	Models data.Models
}

func main() {
	// connect to mongo
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Panicln(err)
	}
	client = mongoClient

	// create a context in order to disconnect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// close connection
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	app := Config{
		Models: data.New(mongoClient),
	}

	// register the RPC server
	err = rpc.Register(new(RPCServer)) // * Equivalent to `&RPCServer{}`
	// new() is a built-in function that allocates memory for a zeroed value of a specified type and returns a pointer to it.
	if err != nil {
		log.Panicln(err)
	}
	go app.rpcListen()

	// register gRPC server
	go app.gRPCListen()

	// start web server
	app.serveHttp()
}

func (app *Config) serveHttp() {
	log.Println("Starting service on port", webPort)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func (app *Config) rpcListen() {
	log.Println("Starting RPC server on port", rpcPort)

	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", rpcPort))
	if err != nil {
		log.Panicln(err)
	}
	defer listen.Close()

	for {
		rpcConn, err := listen.Accept()
		if err != nil {
			continue
		}

		go rpc.ServeConn(rpcConn)
	}
}

func connectToMongo() (*mongo.Client, error) {
	// create connection options
	clientOptions := options.Client().ApplyURI(mongoURL)
	// ! This should be implemented using environment variables
	clientOptions.SetAuth(options.Credential{
		Username: "root",
		Password: "123456",
	})

	// connect
	c, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Println("Error Connecting:", err)
		return nil, err
	}

	return c, nil
}
