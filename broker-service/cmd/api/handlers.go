package main

import (
	"broker/event"
	"broker/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/rpc"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	// out, _ := json.MarshalIndent(payload, "", "\t")
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusAccepted)
	// w.Write(out)

	app.writeJson(w, http.StatusOK, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.Authenticate(w, requestPayload.Auth)
	case "log":
		// app.LogItem(w, requestPayload.Log)
		// app.logEventViaRabbit(w, requestPayload.Log)
		app.logEventViaRPC(w, requestPayload.Log)
	case "mail":
		app.SendMail(w, requestPayload.Mail)
	default:
		app.errorJson(w, errors.New("unknown action"))
	}
}

func (app *Config) Authenticate(w http.ResponseWriter, a AuthPayload) {
	// create some json we'll send to the auth microservice
	jsonData, err := json.Marshal(a)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	// call the service
	request, err := http.NewRequest(http.MethodPost, "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJson(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJson(w, err)
		return
	}
	defer response.Body.Close()
	fmt.Println(response.Status)

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode != http.StatusOK {
		app.errorJson(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusOK {
		app.errorJson(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response body into
	var jsonFromService jsonResponse

	// decode the json from auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	fmt.Println(jsonFromService)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJson(w, err, http.StatusUnauthorized)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Authenticated!",
		Data:    jsonFromService.Data,
	}

	app.writeJson(w, http.StatusOK, payload)
}

func (app *Config) LogItem(w http.ResponseWriter, entry LogPayload) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest(http.MethodPost, logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJson(w, err)
		return
	}

	request.Header.Set("Content-Type", "Application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		app.errorJson(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		app.errorJson(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged"

	app.writeJson(w, http.StatusOK, payload)
}

func (app *Config) SendMail(w http.ResponseWriter, msg MailPayload) {
	json, err := json.Marshal(msg)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	// call the mail service
	mailServiceURL := "http://mail-service/send"

	// post to mail service
	request, err := http.NewRequest(http.MethodPost, mailServiceURL, bytes.NewBuffer(json))
	if err != nil {
		app.errorJson(w, err)
		return
	}

	request.Header.Set("Content-Type", "Application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		app.errorJson(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get right status code
	if response.StatusCode != http.StatusOK {
		app.errorJson(w, errors.New("error calling mail service"), http.StatusInternalServerError)
		return
	}

	// send back json
	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Message send to %s", msg.To),
	}

	app.writeJson(w, http.StatusOK, payload)
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		app.errorJson(w, err, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "logged via RabbitMQ",
	}
	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.rabit)
	if err != nil {
		return err
	}

	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	j, err := json.Marshal(&payload)
	if err != nil {
		return err
	}

	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}

	return nil
}

type RPCPayload struct { // * Must be exactly the same as the server type
	Name string
	Data string
}

func (app *Config) logEventViaRPC(w http.ResponseWriter, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		app.errorJson(w, err, http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// rpcPayload := RPCPayload{// ! linter error
	// 	Name: l.Name,
	// 	Data: l.Data,
	// }
	rpcPayload := RPCPayload(l)

	var result string
	err = client.Call("RPCServer.LogInfo", rpcPayload, &result)
	// * `RPCServer` is the type that is created on the rpc server
	// * `LogInfo` is the function name on the server
	if err != nil {
		app.errorJson(w, err, http.StatusInternalServerError)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}

	app.writeJson(w, http.StatusOK, payload)
}

func (app *Config) LogViaGRPC(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	// conn, err := grpc.Dial("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	conn, err := grpc.NewClient("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		app.errorJson(w, err)
		return
	}
	defer conn.Close()

	c := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})
	if err != nil {
		app.errorJson(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "logged",
	}
	app.writeJson(w, http.StatusOK, payload)
}
