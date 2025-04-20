package main

import (
	"net/http"
)

type mailMessage struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (app *Config) SendMail(w http.ResponseWriter, r *http.Request) {
	var reqPayload mailMessage

	err := app.readJson(w, r, &reqPayload)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	msg := Message{
		From:    reqPayload.From,
		To:      reqPayload.To,
		Subject: reqPayload.Subject,
		Data:    reqPayload.Message,
	}

	err = app.Mailer.SendSMTPMessage(&msg)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "send to " + reqPayload.To,
	}

	app.writeJson(w, http.StatusOK, payload)
}
