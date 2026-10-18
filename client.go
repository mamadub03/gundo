package main

import (
	"os"
	"os/user"
	"runtime"
	"time"
	"uuid"
)

// fixed client version rep
const clientVersion = "0.1.0"

// in memory state that payload/process is running. It gives us a place to handle client specific behavior

type Client struct {
	ID        string
	Version   string
	StartedAt time.Time
	// set of task ID already handled to avoid duplicate execution
	Processed map[string]struct{}
}

// function to create a new client struct in memory and returns its address.
// that way we can actually modify the values in memory rather than a copy
func newClient() *Client {
	return &Client{
		ID:        uuid.New().String(),
		Version:   clientVersion,
		StartedAt: time.Now().UTC(),
		Processed: make(map[string]struct{}),
	}
}

/*

Client Behaviour:

1. GundoMessage Task recieved
2. client handles/executes task (client.handleTask(gundotask))
3. inspects the task action (ping, echo, stauts, etc.)
4. return the result as a GundoMessage onto repo
*/

func (c *Client) handleTask(task GundoMessage) GundoMessage {

	// Every result needs these common fields.
	// take task from param --> drop it in to start building result return struct --> populate struct with output and return it
	result := GundoMessage{
		Type:      "result",
		TaskID:    task.TaskID,
		ClientID:  c.ID,
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Run only explicitly supported Gundo actions.
	switch task.Action {

	case "ping":
		result.Output = "pong"

	case "get_hostname":
		hostname, err := os.Hostname()
		if err != nil {
			result.Status = "error"
			result.Output = err.Error()
			break
		}

		result.Output = hostname

	case "get_os":
		result.Output = runtime.GOOS

	case "get_user":
		currentUser, err := user.Current()
		if err != nil {
			result.Status = "error"
			result.Output = err.Error()
			break
		}

		result.Output = currentUser.Username

	default:
		result.Status = "error"
		result.Output = "unsupported action"
	}

	return result
}
