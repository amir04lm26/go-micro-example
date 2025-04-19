package main

import (
	"errors"
	"log-service/data"
	"net/http"
)

type JSONPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {
	// read json into var
	var requestPayload JSONPayload
	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		app.errorJson(w, errors.New(http.StatusText(http.StatusBadRequest)))
	}

	// insert data
	event := data.LogEntry{
		Name: requestPayload.Name,
		Data: requestPayload.Data,
	}
	err = app.Models.LogEntry.Insert(event)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	resp := jsonResponse{
		Error:   false,
		Message: "Logged",
	}

	app.writeJson(w, http.StatusOK, resp)
}
