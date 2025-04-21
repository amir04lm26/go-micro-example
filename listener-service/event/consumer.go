package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn      *amqp.Connection
	queueName string
}

func NewConsumer(conn *amqp.Connection) (Consumer, error) {
	consumer := Consumer{
		conn: conn,
	}

	if err := consumer.setup(); err != nil {
		return Consumer{}, err
	}

	return consumer, nil
}

func (consumer *Consumer) setup() error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		return err
	}

	return declareExchange(channel)
}

type Payload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (consumer *Consumer) Listen(topics []string) error {
	ch, err := consumer.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := declareRandomQueue(ch)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		err := ch.QueueBind(
			q.Name,
			topic,
			"logs_topic",
			false,
			nil,
		)

		if err != nil {
			return err
		}
	}

	messages, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil
	}

	forever := make(chan bool)
	defer close(forever)
	go func() {
		for msg := range messages {
			var payload Payload
			json.Unmarshal(msg.Body, &payload)

			go handlePayload(&payload)
		}
	}()

	fmt.Printf("Waiting for message on [Exchange, Queue] [logs_topic, %s]\r\n", q.Name)

	<-forever

	return nil
}

func handlePayload(payload *Payload) {
	switch payload.Name {
	case "log", "event":
		// log whatever we get
		if err := logEvent(payload); err != nil {
			log.Println(err)
		}

	case "auth":
		// authenticate
	}
}

func logEvent(payload *Payload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest(http.MethodPost, logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "Application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return err
	}

	return nil
}
