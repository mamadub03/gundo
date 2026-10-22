package main

import (
	"encoding/json"
	"fmt"
	"time"
	"uuid"
)

func generateTaskID() string {
	// generate a new UUID
	return uuid.New().String()
}

func createTaskMessage(action string) GundoMessage {

	// create a new GundoMessage struct with the provided action and a generated task ID
	message := GundoMessage{
		Type:      "task",
		TaskID:    generateTaskID(),
		Action:    action,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	return message
}

func parseGundoMessage(body string) (GundoMessage, error) {
	var message GundoMessage

	//  unmarshals and packs JSON back into the GundoMessage struct
	err := json.Unmarshal([]byte(body), &message)
	if err != nil {
		return GundoMessage{}, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return message, nil

}
