package main

import (
	"fmt"
	"testing"
)

func TestSendTask(t *testing.T) {
	const issueNumber = 22
	const clientID = "cec362f0-b7a3-464e-8396-1fed0b946a8a"

	task := createTaskMessage("get_hostname")
	task.TargetClientID = clientID

	err := createComment(issueNumber, task)
	if err != nil {
		t.Fatalf("error sending task: %v", err)
	}

	fmt.Println("Task sent successfully")
	fmt.Printf("Task ID: %s\n", task.TaskID)
	fmt.Printf("Target:  %s\n", task.TargetClientID)
	fmt.Printf("Action:  %s\n", task.Action)
}
